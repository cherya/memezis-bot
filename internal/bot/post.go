package bot

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/cherya/memezis/pkg/memezis"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type fileData struct {
	FileID string
	SHA    string
}

type Duplicates struct {
	Complete []int64 // Complete full match
	Likely   []int64 // Likely a bit difference, likely same pic
	Similar  []int64 // Similar similar pics
}

func (b *MemezisBot) savePhotoPost(ctx context.Context, text string, media []string, createdAt time.Time) (int64, *Duplicates, error) {
	files := make([]tgbotapi.File, 0, len(media))
	urls := make([]string, len(media))
	var filenameToFileData sync.Map
	wg := sync.WaitGroup{}

	for _, m := range media {
		f, err := b.api.GetFile(tgbotapi.FileConfig{FileID: m})
		if err != nil {
			return 0, nil, errors.Wrap(err, "can't get file from telegram")
		}
		files = append(files, f)
	}
	wg.Add(len(files))
	for i, f := range files {
		link := f.Link(b.api.Token)
		fileID := f.FileID

		go func(idx int) {
			resp, err := http.Get(link)
			if err != nil {
				log.Error(errors.Wrap(err, "can't get file from telegram"))
				return
			}

			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()

			uploadResp, err := b.upload(ctx, bodyBytes, fileID)
			if err != nil {
				log.Error(errors.Wrap(err, "can't upload media"))
				return
			}
			urls[idx] = uploadResp.GetURL()

			sha := sha1.New()
			if _, err := io.Copy(sha, bytes.NewBuffer(bodyBytes)); err != nil {
				log.Error(errors.Wrap(err, "can't get sha"))
				return
			}

			filenameToFileData.Store(uploadResp.GetURL(), fileData{
				FileID: fileID,
				SHA:    hex.EncodeToString(sha.Sum(nil)[:20]),
			})
			wg.Done()
		}(i)
	}

	wg.Wait()

	postMedia := make([]*memezis.Media, 0, len(media))
	for _, u := range urls {
		d, _ := filenameToFileData.Load(u)
		data := d.(fileData)
		postMedia = append(postMedia, &memezis.Media{
			URL:      u,
			Type:     "photo",
			SourceID: data.FileID,
			SHA1:     data.SHA,
		})
	}

	addResp, err := b.mc.AddPost(ctx, &memezis.AddPostRequest{
		Media:     postMedia,
		AddedBy:   strconv.Itoa(userFromContext(ctx)),
		Text:      text,
		CreatedAt: toProtoTime(time.Now().UTC()),
	})
	if err != nil {
		return 0, nil, errors.Wrap(err, "can't add post")
	}

	return addResp.GetID(), fromProtoDuplicates(addResp.GetDuplicates()), nil
}

func (b MemezisBot) upload(ctx context.Context, image []byte, filename string) (*memezis.UploadMediaResponse, error) {
	stream, err := b.mc.UploadMedia(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "can't open upload stream")
	}

	file := bytes.NewBuffer(image)

	err = stream.Send(&memezis.UploadMediaRequest{
		T: &memezis.UploadMediaRequest_Meta{
			Meta: &memezis.MediaMetadata{
				Filename: filename,
				Type:     memezis.MediaType_JPG,
				Filesize: int64(file.Len()),
			},
		},
	})
	if err != nil {
		log.Error(errors.Wrap(err, "can't send metadata"))
	}

	buf := make([]byte, 1024)
	for {
		n, err := file.Read(buf)
		if err != nil {
			if err != io.EOF {
				return nil, errors.Wrap(err, "can't read from file buffer")
			}
			break
		}

		err = stream.Send(&memezis.UploadMediaRequest{
			T: &memezis.UploadMediaRequest_Image{
				Image: buf[:n],
			},
		})
		if err != nil {
			return nil, errors.Wrap(err, "can't send image chunk")
		}
	}

	uploadResp, err := stream.CloseAndRecv()
	if err != nil {
		return nil, errors.Wrap(err, "can't send image")
	}

	return uploadResp, nil
}

func (b *MemezisBot) saveExternalPost(ctx context.Context, text, sourceID, typ string) (int64, error) {
	postMedia := []*memezis.Media{
		{
			Type:     typ,
			SourceID: sourceID,
		},
	}

	addResp, err := b.mc.AddPost(ctx, &memezis.AddPostRequest{
		Media:     postMedia,
		AddedBy:   strconv.Itoa(userFromContext(ctx)),
		Text:      text,
		CreatedAt: toProtoTime(time.Now().UTC()),
	})
	if err != nil {
		return 0, err
	}

	return addResp.GetID(), nil
}

func (b *MemezisBot) sendMediaGroupPostVoting(post *memezis.Post, sender string) (int, error) {
	media := make([]interface{}, 0, len(post.GetMedia()))
	for i, m := range post.GetMedia() {
		var imp tgbotapi.InputMediaPhoto
		if m.SourceID != "" {
			imp = tgbotapi.NewInputMediaPhoto(m.SourceID)
		} else if m.URL != "" {
			imp = tgbotapi.NewInputMediaPhoto(m.URL)
		} else {
			return 0, errors.New("can't send empty media")
		}
		if i == 0 {
			imp.Caption = textWithSender(post.GetText(), sender)
		}
		media = append(media, imp)
	}
	msg := tgbotapi.NewMediaGroup(b.suggestionChannel, media)
	msg.DisableNotification = true
	apiResp, err := b.api.Request(msg)
	if err != nil {
		return 0, err
	}

	var sentMessages []tgMessage
	err = json.Unmarshal(apiResp.Result, &sentMessages)
	if err != nil {
		return 0, errors.Wrap(err, "can't unmarshal telegram media group response")
	}

	keysMsg := tgbotapi.NewMessage(b.suggestionChannel, getVotingText())
	voteKb := createVotingKeyboard(post.GetID(), post.GetVotes().GetUp(), post.GetVotes().GetDown())
	keysMsg.ReplyMarkup = voteKb
	keysMsg.ReplyToMessageID = sentMessages[0].MessageID
	m, err := b.send(keysMsg)
	if err != nil {
		return 0, err
	}
	return m.MessageID, nil
}

func (b *MemezisBot) publishPostVotingByID(ctx context.Context, postID int64, sender string) (int, error) {
	resp, err := b.mc.GetPostByID(ctx, &memezis.GetPostByIDRequest{PostID: postID})
	if err != nil {
		return 0, err
	}

	if len(resp.Media) > 1 {
		msgID, err := b.sendMediaGroupPostVoting(resp, sender)
		if err != nil {
			return 0, err
		}
		return msgID, nil
	}

	if len(resp.Media) == 1 {
		media := resp.Media[0]
		if media.Type == "photo" {
			msg := tgbotapi.NewPhotoShare(b.suggestionChannel, media.SourceID)
			msg.Caption = textWithSender(resp.Text, sender)
			msg.DisableNotification = true
			voteKb := createVotingKeyboard(resp.ID, resp.Votes.Up, resp.Votes.Down)
			msg.ReplyMarkup = voteKb
			m, err := b.send(msg)
			if err != nil {
				return 0, err
			}
			return m.MessageID, nil
		}
		if media.Type == "gif" {
			msg := tgbotapi.NewAnimationShare(b.suggestionChannel, media.SourceID)
			msg.Caption = textWithSender(resp.Text, sender)
			voteKb := createVotingKeyboard(resp.ID, resp.Votes.Up, resp.Votes.Down)
			msg.ReplyMarkup = voteKb
			msg.DisableNotification = true
			m, err := b.send(msg)
			if err != nil {
				return 0, err
			}
			return m.MessageID, nil
		}
		if media.Type == "video" {
			msg := tgbotapi.NewVideoShare(b.suggestionChannel, media.SourceID)
			msg.Caption = textWithSender(resp.Text, sender)
			voteKb := createVotingKeyboard(resp.ID, resp.Votes.Up, resp.Votes.Down)
			msg.ReplyMarkup = voteKb
			msg.DisableNotification = true
			m, err := b.send(msg)
			if err != nil {
				return 0, err
			}
			return m.MessageID, nil
		}
	}

	return 0, nil
}

func (b *MemezisBot) publishInternalPostVotingByID(ctx context.Context, postID int64, reservID int) (int, error) {
	resp, err := b.mc.GetPostByID(ctx, &memezis.GetPostByIDRequest{PostID: postID})
	if err != nil {
		return 0, err
	}

	if len(resp.Media) > 1 {
		msgID, err := b.sendMediaGroupPostVoting(resp, "")
		if err != nil {
			return 0, errors.Wrap(err, "can't send media group voting")
		}
		return msgID, nil
	}

	if len(resp.Media) == 1 {
		edited := tgbotapi.NewEditMessageText(b.suggestionChannel, reservID, getVotingText())
		voteKb := createVotingKeyboard(postID, resp.Votes.Up, resp.Votes.Down)
		edited.ReplyMarkup = &voteKb
		m, err := b.send(edited)
		if err != nil {
			return 0, errors.Wrap(err, "can't edit message text")
		}
		return m.MessageID, nil
	}

	return 0, nil
}

func textWithSender(text, sender string) string {
	if sender == "" {
		return text
	}
	return text + "\n\nприслал " + sender
}

func fromProtoDuplicates(duplicates *memezis.Duplicates) *Duplicates {
	c := make([]int64, 0, len(duplicates.Complete))
	for _, d := range duplicates.Complete {
		c = append(c, d)
	}
	l := make([]int64, len(duplicates.Likely))
	for _, d := range duplicates.Likely {
		l = append(l, d)
	}
	s := make([]int64, len(duplicates.Similar))
	for _, d := range duplicates.Similar {
		s = append(s, d)
	}
	return &Duplicates{
		Complete: c,
		Likely:   l,
		Similar:  s,
	}
}