package bot

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

func (b *MemezisBot) messageWorkerWrapper(ctx context.Context, id int, messages <-chan *tgbotapi.Message, errs chan<- error) {
	for message := range messages {
		err := b.messageWorker(ctx, id, message)
		if err != nil {
			errs <- err
		}
	}
}

func (b *MemezisBot) messageWorker(ctx context.Context, id int, message *tgbotapi.Message) error {
	userID := userFromUpdate(message)
	ctx = setUserToContext(ctx, userID)
	if message.IsCommand() {
		b.handleCommand(message)
		return nil
	}
	if message.Chat.ID == b.adminChannel {
		text := message.Text
		if text == "" {
			text = message.Caption
		}
		if !mentionUser(message, b.api.Self) {
			return nil
		}
		err := b.handleAdminChannelMessage(ctx, message)
		return errors.Wrap(err, "messageWorker: error handling message")
	} else if message.Chat.IsPrivate() {
		if !hasMedia(message) {
			_, err := b.send(tgbotapi.NewMessage(message.Chat.ID, getText(TextTypeUnsupported)))
			return errors.Wrap(err, "messageWorker: error sending message")
		}
		err := b.handlePrivateMessage(ctx, message)
		return errors.Wrap(err, "messageWorker: error handling message")
	} else {
		if !mentionUser(message, b.api.Self) {
			return nil
		}
		err := b.handleChatMessage(ctx, message)
		return errors.Wrap(err, "messageWorker: error handling chat message")
	}
}
