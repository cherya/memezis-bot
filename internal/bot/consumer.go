package bot

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/cherya/memezis/pkg/memezis"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const ShaurmemesQueue = "shaurmemes"
const SourceMemezisBot = "memezis_bot"

type tgMessage struct {
	MessageID int `json:"message_id"`
}

func (b *MemezisBot) ShaurmemesConsumer(value string) {
	postID, err := strconv.Atoi(value)
	if err != nil {
		log.Error(errors.Wrapf(err, "invalid postID: %s", value))
		return
	}
	resp, err := b.mc.GetPostByID(context.Background(), &memezis.GetPostByIDRequest{
		PostID: int64(postID),
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
		} else {
			log.Error(errors.New("can't send media group from links"))
			return
		}

		return
	}

	if len(resp.Media) == 1 {
		media := resp.Media[0]
		if media.Type == "photo" {
			msg := tgbotapi.NewPhotoShare(b.publicationChannel, media.SourceID)
			msg.Caption = resp.Text
			_, err := b.send(msg)
			if err != nil {
				log.Error(errors.Wrap(err, "can't publish proto"))
				return
			}
		} else if media.Type == "gif" {
			msg := tgbotapi.NewAnimationShare(b.publicationChannel, media.SourceID)
			msg.Caption = resp.Text
			_, err := b.send(msg)
			if err != nil {
				log.Error(errors.Wrap(err, "can't publish gif"))
				return
			}
		} else if media.Type == "video" {
			msg := tgbotapi.NewVideoShare(b.publicationChannel, media.SourceID)
			msg.Caption = resp.Text
			_, err := b.send(msg)
			if err != nil {
				log.Error(errors.Wrap(err, "can't publish video"))
				return
			}
		} else {
			log.Warnf("consumer get unsupported media type (%s)", media.Type)
		}
	}

	return
}
