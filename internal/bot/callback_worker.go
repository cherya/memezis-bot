package bot

import (
	"context"
	"fmt"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

var callbackAtom *sync.Map

func (b *MemezisBot) callbackWorker(ctx context.Context, id int, callbacks <-chan *tgbotapi.CallbackQuery, errs chan<- error) {
	for callback := range callbacks {
		userID := userFromUpdate(callback)
		ctx = setUserToContext(ctx, userID)

		callbackKey := fmt.Sprintf("%d:%d", userID, callback.Message.MessageID)
		if _, loaded := callbackAtom.LoadOrStore(callbackKey, true); loaded {
			continue
		}

		err := b.callbackQuery(ctx, callback)
		if err != nil {
			errs <- errors.Wrap(err, "callbackWorker: error handling CallbackQuery")
		}

		callbackAtom.Delete(callbackKey)
	}
}
