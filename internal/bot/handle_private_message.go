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

	if msg.Photo != nil {
		previewMessage := tgbotapi.NewPhotoShare(msg.Chat.ID, getFileIDFromMsg(msg))
		previewMessage.Caption = msg.Caption
		previewMessage.ReplyMarkup = createConfirmationKeyboard(text)
		_, err := b.send(previewMessage)
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't send preview message")
		}
	} else if msg.Animation != nil {
		previewMessage := tgbotapi.NewAnimationShare(msg.Chat.ID, getFileIDFromMsg(msg))
		previewMessage.Caption = msg.Caption
		previewMessage.ReplyMarkup = createConfirmationKeyboard(text)
		_, err := b.send(previewMessage)
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't send preview message")
		}
	} else if msg.Video != nil {
		previewMessage := tgbotapi.NewVideoShare(msg.Chat.ID, getFileIDFromMsg(msg))
		previewMessage.Caption = msg.Caption
		previewMessage.ReplyMarkup = createConfirmationKeyboard(text)
		_, err := b.send(previewMessage)
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't send preview message")
		}
	}
	return nil
}

func (b *MemezisBot) uploadNewPost(ctx context.Context, msg *tgbotapi.Message, sentBy *tgbotapi.User) (int64, error) {
	text := msg.Text
	if text == "" {
		text = msg.Caption
	}
	msg.From = sentBy

	var err error
	var postID int64

	if msg.Photo != nil {
		postID, err = b.savePhotoPost(ctx, text, []string{getFileIDFromMsg(msg)}, msg.Time())
		if err != nil {
			return 0, errors.Wrap(err, "handlePrivateMessage: can't save post")
		}
		votingMsgID, err := b.publishPostVotingByID(ctx, postID, getUsername(msg))
		if err != nil {
			return 0, errors.Wrap(err, "handlePrivateMessage: can't publish post voting")
		}
		err = b.processDuplicates(ctx, postID, votingMsgID)
		if err != nil {
			log.Errorf("error processing duplicates %s", err)
		}
	} else if msg.Animation != nil {
		postID, err = b.saveExternalPost(ctx, text, getFileIDFromMsg(msg), "gif")
		if err != nil {
			return 0, errors.Wrap(err, "handlePrivateMessage: can't save post")
		}
		_, err = b.publishPostVotingByID(ctx, postID, getUsername(msg))
		if err != nil {
			return 0, errors.Wrap(err, "handlePrivateMessage: can't publish post voting")
		}
	} else if msg.Video != nil { // TODO handle video group
		postID, err = b.saveExternalPost(ctx, text, getFileIDFromMsg(msg), "video")
		if err != nil {
			return 0, errors.Wrap(err, "handlePrivateMessage: can't save post")
		}
		_, err = b.publishPostVotingByID(ctx, postID, getUsername(msg))
		if err != nil {
			return 0, errors.Wrap(err, "handlePrivateMessage: can't publish post voting")
		}
	} else {
		m := tgbotapi.NewMessage(msg.Chat.ID, "я так не умею")
		m.ReplyToMessageID = msg.MessageID
		_, err := b.send(m)
		if err != nil {
			return 0, errors.Wrap(err, "handlePrivateMessage: can't send message")
		}
		return 0, nil
	}
	return postID, nil
}
