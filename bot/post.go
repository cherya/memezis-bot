package bot

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"

	mmc "github.com/cherya/memezis-bot/memezis_client"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

type fileData struct {
	FileID string
	SHA    string
}

func (b *PublisherBot) savePhotoPost(ctx context.Context, text string, media []string) (int64, []int64, error) {
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
				log.Println(errors.Wrap(err, "can't get file from telegram"))
				return
			}

			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()

			uploadResp, err := b.mc.UploadMedia(bytes.NewBuffer(bodyBytes), fileID)
			if err != nil {
				log.Println(errors.Wrap(err, "can't upload media"))
				return
			}
			urls[idx] = uploadResp.Filename

			sha := sha1.New()
			if _, err := io.Copy(sha, bytes.NewBuffer(bodyBytes)); err != nil {
				log.Println(errors.Wrap(err, "can't get sha"))
				return
			}

			filenameToFileData.Store(uploadResp.Filename, fileData{
				FileID: fileID,
				SHA:    hex.EncodeToString(sha.Sum(nil)[:20]),
			})
			wg.Done()
		}(i)
	}

	wg.Wait()

	postMedia := make([]mmc.Media, 0, len(media))
	for _, u := range urls {
		d, _ := filenameToFileData.Load(u)
		data := d.(fileData)
		postMedia = append(postMedia, mmc.Media{
			URL:      u,
			Type:     "photo",
			SourceID: data.FileID,
			SHA1:     data.SHA,
		})
	}

	addResp, err := b.mc.AddPost(postMedia, strconv.Itoa(userFromContext(ctx)), text, nil)
	if err != nil {
		return 0, nil, errors.Wrap(err, "can't add post")
	}

	return addResp.ID, addResp.Duplicates, nil
}

func (b *PublisherBot) saveExternalPost(ctx context.Context, text, sourceID, typ string) (int64, error) {
	postMedia := []mmc.Media{{
		Type:     typ,
		SourceID: sourceID,
	}}

	addResp, err := b.mc.AddPost(postMedia, strconv.Itoa(userFromContext(ctx)), text, nil)
	if err != nil {
		return 0, err
	}

	return addResp.ID, nil
}

func (b *PublisherBot) sendMediaGroupPostVoting(post *mmc.GetPostByIDResponse, sender string) (int, error) {
	media := make([]interface{}, 0, len(post.Media))
	for i, m := range post.Media {
		var imp tgbotapi.InputMediaPhoto
		if m.SourceID != "" {
			imp = tgbotapi.NewInputMediaPhoto(m.SourceID)
		} else if m.URL != "" {
			imp = tgbotapi.NewInputMediaPhoto(m.URL)
		} else {
			return 0, errors.New("can't send empty media")
		}
		if i == 0 {
			imp.Caption = textWithSender(post.Text, sender)
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
	voteKb := createVotingKeyboard(post.ID, post.Votes.Up, post.Votes.Down)
	keysMsg.ReplyMarkup = voteKb
	keysMsg.ReplyToMessageID = sentMessages[0].MessageID
	m, err := b.send(keysMsg)
	if err != nil {
		return 0, err
	}
	return m.MessageID, nil
}

func (b *PublisherBot) publishPostVotingByID(ctx context.Context, postID int64, sender string) (int, error) {
	resp, err := b.mc.GetPost(postID)
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

func (b *PublisherBot) publishInternalPostVotingByID(ctx context.Context, postID int64, reservID int) (int, error) {
	resp, err := b.mc.GetPost(postID)
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
	return text + "\n\nприслал @" + sender
}
