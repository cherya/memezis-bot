package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/cherya/memezis/pkg/memezis"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const ShaurmemesQueue, ShaurmemesChannelName = "shaurmemes", "shaurmemes"
const SourceMemezisBot = "memezis_bot"

const ShaurmemesUrl = "https://t.me/shaurmemes/%d"

type tgMessage struct {
	MessageID int   `json:"message_id"`
	Date      int64 `json:"date"`
}

func (b *MemezisBot) ShaurmemesConsumer(value string) {
	var publishID int
	var publishedAt time.Time
	postID, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		log.Error(errors.Wrapf(err, "invalid postID: %s", value))
		return
	}
	resp, err := b.mc.GetPostByID(context.Background(), &memezis.GetPostByIDRequest{
		PostID: postID,
	})
	if err != nil {
		log.Error(errors.Wrapf(err, "can't get post by id (id=%d)", postID))
		return
	}

	if len(resp.Media) > 1 {
		if resp.Source == SourceMemezisBot {
			media := make([]interface{}, 0, len(resp.Media))
			for i, m := range resp.Media {
				var imp tgbotapi.InputMediaPhoto
				if m.SourceID != "" {
					imp = tgbotapi.NewInputMediaPhoto(m.SourceID)
				} else if m.URL != "" {
					imp = tgbotapi.NewInputMediaPhoto(m.URL)
				} else {
					log.Error(errors.New("can't send empty media"))
					return
				}
				if i == 0 {
					imp.Caption = resp.Text
				}
				media = append(media, imp)
			}
			msg := tgbotapi.NewMediaGroup(b.publicationChannel, media)
			apiResp, err := b.api.Request(msg)
			if err != nil {
				log.Error(errors.Wrap(err, "can't send media group"))
				return
			}

			var sentMessages []tgMessage
			err = json.Unmarshal(apiResp.Result, &sentMessages)
			if err != nil {
				log.Error(errors.Wrap(err, "can't unmarshal telegram media group response"))
				return
			}
			publishID = sentMessages[0].MessageID
			publishedAt = time.Unix(sentMessages[0].Date, 0)
		} else {
			log.Error(errors.New("can't send media group from links"))
			return
		}

		return
	}

	if len(resp.Media) == 1 {
		media := resp.Media[0]
		if media.Type == "photo" {
			msg := tgbotapi.NewPhotoUpload(b.publicationChannel, nil)
			msg.FileID = media.URL
			msg.UseExisting = true
			msg.Caption = resp.Text
			resp, err := b.send(msg)
			if err != nil {
				log.Error(errors.Wrap(err, "can't publish proto"))
				return
			}
			publishID = resp.MessageID
			publishedAt = time.Unix(int64(resp.Date), 0)
		} else if media.Type == "gif" {
			msg := tgbotapi.NewAnimationShare(b.publicationChannel, media.SourceID)
			msg.Caption = resp.Text
			resp, err := b.send(msg)
			if err != nil {
				log.Error(errors.Wrap(err, "can't publish gif"))
				return
			}
			publishID = resp.MessageID
			publishedAt = time.Unix(int64(resp.Date), 0)
		} else if media.Type == "video" {
			msg := tgbotapi.NewVideoShare(b.publicationChannel, media.SourceID)
			msg.Caption = resp.Text
			resp, err := b.send(msg)
			if err != nil {
				log.Error(errors.Wrap(err, "can't publish video"))
				return
			}
			publishID = resp.MessageID
			publishedAt = time.Unix(int64(resp.Date), 0)
		} else {
			log.Warnf("consumer get unsupported media type (%s)", media.Type)
		}
	}

	if postID != 0 {
		_, err = b.mc.PublishPost(context.Background(), &memezis.PublishPostRequest{
			PostID:      postID,
			URL:         fmt.Sprintf(ShaurmemesUrl, publishID),
			PublishedTo: ShaurmemesChannelName,
			PublishedAt: toProtoTime(publishedAt),
		})
		if err != nil {
			log.Error(errors.Wrap(err, "can't send post publish status"))
		}
	}

	return
}
