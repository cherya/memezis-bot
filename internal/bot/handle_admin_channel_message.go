package bot

import (
	"context"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (b *MemezisBot) handleAdminChannelMessage(ctx context.Context, msg *tgbotapi.Message) error {
	var postID int64
	var err error
	var reservMsg tgbotapi.Message

	//TODO: handle media group
	if msg.MediaGroupID != "" {
		return nil
	}

	if msg.ReplyToMessage != nil {
		if hasMedia(msg.ReplyToMessage) {
			msg = msg.ReplyToMessage
		}
	}

	text := msg.Caption

	text = strings.ReplaceAll(text, "@"+b.api.Self.UserName, "")
	text = strings.TrimSpace(text)

	if msg.Photo != nil {
		m := tgbotapi.NewMessage(b.adminChannel, "секундочку...")
		m.ReplyToMessageID = msg.MessageID
		reservMsg, err = b.send(m)

		postID, err = b.savePhotoPost(ctx, text, []string{getFileIDFromMsg(msg)}, msg.Time())
		if err != nil {
			return errors.Wrap(err, "handleDirectSuggestionMessage: can't save post")
		}
		err = b.processDuplicates(ctx, postID, msg.MessageID)
		if err != nil {
			log.Errorf("error processing duplicates %s", err)
		}
	} else if msg.Animation != nil {
		m := tgbotapi.NewMessage(b.adminChannel, "секундочку...")
		m.ReplyToMessageID = msg.MessageID
		reservMsg, err = b.send(m)

		postID, err = b.saveExternalPost(ctx, text, getFileIDFromMsg(msg), "gif")
		if err != nil {
			return errors.Wrap(err, "handleDirectSuggestionMessage: can't save post")
		}
	} else if msg.Video != nil && msg.MediaGroupID == "" { // TODO handle video group
		m := tgbotapi.NewMessage(b.adminChannel, "секундочку...")
		m.ReplyToMessageID = msg.MessageID
		reservMsg, err = b.send(m)

		postID, err = b.saveExternalPost(ctx, text, getFileIDFromMsg(msg), "video")
		if err != nil {
			return errors.Wrap(err, "handleDirectSuggestionMessage: can't save post")
		}
	}

	_, err = b.publishInternalPostVotingByID(ctx, postID, reservMsg.MessageID)
	if err != nil {
		return errors.Wrap(err, "handleDirectSuggestionMessage: can't publish post voting")
	}

	return nil
}
