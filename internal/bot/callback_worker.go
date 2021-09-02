package bot

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

func (b *MemezisBot) callbackWorker(ctx context.Context, id int, callbacks <-chan *tgbotapi.CallbackQuery, errs chan<- error) {
	for callback := range callbacks {
		userID := userFromUpdate(callback)
		ctx = setUserToContext(ctx, userID)

		callbackKey := fmt.Sprintf("%d:%d", userID, callback.Message.MessageID)
		if _, loaded := b.callbackAtom.LoadOrStore(callbackKey, true); loaded {
			continue
		}

		err := b.callbackQuery(ctx, callback)
		if err != nil {
			errs <- errors.Wrap(err, "callbackWorker: error handling CallbackQuery")
		}

		b.callbackAtom.Delete(callbackKey)
	}
}
