package bot

import (
	"context"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (b *MemezisBot) handleChatMessage(ctx context.Context, msg *tgbotapi.Message) error {
	var postID int64
	var err error

	sender := getUsername(msg)
	text := msg.Caption
	text = strings.ReplaceAll(text, "@"+b.api.Self.UserName, "")
	text = strings.TrimSpace(text)

	if msg.ReplyToMessage == nil || !hasMedia(msg.ReplyToMessage) {
		return nil
	}

	msg = msg.ReplyToMessage

	if msg.Photo != nil {
		m := tgbotapi.NewMessage(msg.Chat.ID, "украл")
		m.ReplyToMessageID = msg.MessageID
		_, err = b.send(m)

		postID, err = b.savePhotoPost(ctx, text, []string{getFileIDFromMsg(msg)}, msg.Time())
		if err != nil {
			return errors.Wrap(err, "handleChatMessage: can't save post")
		}
	} else if msg.Animation != nil {
		m := tgbotapi.NewMessage(msg.Chat.ID, "украл")
		m.ReplyToMessageID = msg.MessageID
		_, err = b.send(m)

		postID, err = b.saveExternalPost(ctx, text, getFileIDFromMsg(msg), "gif")
		if err != nil {
			return errors.Wrap(err, "handleChatMessage: can't save post")
		}
	} else if msg.Video != nil && msg.MediaGroupID == "" { // TODO handle video group
		m := tgbotapi.NewMessage(msg.Chat.ID, "украл")
		m.ReplyToMessageID = msg.MessageID
		_, err = b.send(m)

		postID, err = b.saveExternalPost(ctx, text, getFileIDFromMsg(msg), "video")
		if err != nil {
			return errors.Wrap(err, "handleChatMessage: can't save post")
		}
	}

	votingMsgID, err := b.publishPostVotingByID(ctx, postID, sender)
	err = b.processDuplicates(ctx, postID, votingMsgID)
	if err != nil {
		log.Errorf("error processing duplicates %s", err)
	}

	if err != nil {
		return errors.Wrap(err, "handleChatMessage: can't publish post voting")
	}

	return nil
}
