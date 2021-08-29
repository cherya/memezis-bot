package bot

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// хэндлер сообщений (не команд) в личку боту
func (b *MemezisBot) handlePrivateMessage(ctx context.Context, msg *tgbotapi.Message) error {
	text := msg.Text
	if text == "" {
		text = msg.Caption
	}

	if msg.MediaGroupID != "" && msg.Photo != nil {
		b.handleMediaGroup(ctx, msg)
		return nil
	}

	m := tgbotapi.NewMessage(msg.Chat.ID, getText(TextTypeSucessUpload))
	m.ReplyToMessageID = msg.MessageID
	_, err := b.send(m)
	if err != nil {
		return errors.Wrap(err, "handlePrivateMessage: can't send message")
	}

	if msg.Photo != nil {
		postID, err := b.savePhotoPost(ctx, text, []string{getFileIDFromMsg(msg)}, msg.Time())
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't save post")
		}
		votingMsgID, err := b.publishPostVotingByID(ctx, postID, getUsername(msg))
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't publish post voting")
		}
		err = b.processDuplicates(ctx, postID, votingMsgID)
		if err != nil {
			log.Errorf("error processing duplicates %s", err)
		}
	} else if msg.Animation != nil {
		postID, err := b.saveExternalPost(ctx, text, getFileIDFromMsg(msg), "gif")
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't save post")
		}
		_, err = b.publishPostVotingByID(ctx, postID, getUsername(msg))
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't publish post voting")
		}
	} else if msg.Video != nil && msg.MediaGroupID == "" { // TODO handle video group
		postID, err := b.saveExternalPost(ctx, text, getFileIDFromMsg(msg), "video")
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't save post")
		}
		_, err = b.publishPostVotingByID(ctx, postID, getUsername(msg))
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't publish post voting")
		}
	} else {
		m := tgbotapi.NewMessage(msg.Chat.ID, "я так не умею")
		m.ReplyToMessageID = msg.MessageID
		_, err := b.send(m)
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't send message")
		}
		return nil
	}

	return nil
}
