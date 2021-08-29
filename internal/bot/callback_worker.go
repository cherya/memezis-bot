package bot

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

func (b *MemezisBot) callbackWorker(ctx context.Context, id int, callbacks <-chan *tgbotapi.CallbackQuery, errs chan<- error) {
	for callback := range callbacks {
		userID := userFromUpdate(callback)
		ctx = setUserToContext(ctx, userID)

		err := b.callbackQuery(ctx, callback)
		if err != nil {
			errs <- errors.Wrap(err, "callbackWorker: error handling CallbackQuery")
		}
	}
}
