package bot

import (
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

func (b *MemezisBot) isBanned(message *tgbotapi.Message) bool {
	isBannedID, err := b.banHammer.IsBanned(strconv.Itoa(message.From.ID))
	if err != nil {
		fmt.Println(errors.Wrap(err, "can't check ban status"))
		return false
	}
	isBannedUsername, err := b.banHammer.IsBanned("@" + message.From.UserName)
	if err != nil {
		fmt.Println(errors.Wrap(err, "can't check ban status"))
		return false
	}
	return isBannedID || isBannedUsername
}
