package bot

import (
	"encoding/json"
	"log"

	"github.com/cherya/memezis-bot/memezis_client"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gocraft/work"
	"github.com/pkg/errors"
)

const ShaurmemesQueue = "shaurmemes"

type tgMessage struct {
	MessageID int `json:"message_id"`
}

func (b *PublisherBot) ShaurmemesConsumer(job *work.Job) error {
	resp, err := b.mc.GetPost(job.ArgInt64("postID"))
	if err != nil {
		return errors.Wrap(err, "can't get post")
	}

	if len(resp.Media) > 1 {
		if resp.Source == memezis_client.SourceMemezisBot {
			media := make([]interface{}, 0, len(resp.Media))
			for i, m := range resp.Media {
				var imp tgbotapi.InputMediaPhoto
				if m.SourceID != "" {
					imp = tgbotapi.NewInputMediaPhoto(m.SourceID)
				} else if m.URL != "" {
					imp = tgbotapi.NewInputMediaPhoto(m.URL)
				} else {
					return errors.New("can't send empty media")
				}
				if i == 0 {
					imp.Caption = resp.Text
				}
				media = append(media, imp)
			}
			msg := tgbotapi.NewMediaGroup(b.publicationChannel, media)
			apiResp, err := b.api.Request(msg)
			if err != nil {
				return errors.Wrap(err, "can't send media group")
			}

			var sentMessages []tgMessage
			err = json.Unmarshal(apiResp.Result, &sentMessages)
			if err != nil {
				return errors.Wrap(err, "can't unmarshal telegram media group response")
			}
		} else {
			return errors.New("can't send media group from links")
		}

		return nil
	}

	if len(resp.Media) == 1 {
		media := resp.Media[0]
		if media.Type == "photo" {
			msg := tgbotapi.NewPhotoShare(b.publicationChannel, media.SourceID)
			msg.Caption = resp.Text
			_, err := b.send(msg)
			if err != nil {
				return errors.Wrap(err, "can't publish proto")
			}
		} else if media.Type == "gif" {
			msg := tgbotapi.NewAnimationShare(b.publicationChannel, media.SourceID)
			msg.Caption = resp.Text
			_, err := b.send(msg)
			if err != nil {
				return errors.Wrap(err, "can't publish gif")
			}
		} else if media.Type == "video" {
			msg := tgbotapi.NewVideoShare(b.publicationChannel, media.SourceID)
			msg.Caption = resp.Text
			_, err := b.send(msg)
			if err != nil {
				return errors.Wrap(err, "can't publish video")
			}
		} else {
			log.Printf("consumer get unsupported media type (%s)", media.Type)
		}
	}

	return nil
}
