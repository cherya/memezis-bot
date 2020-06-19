package bot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func getFileIDFromMsg(message *tgbotapi.Message) string {
	if message.Photo != nil && len(message.Photo) != 0 {
		return message.Photo[len(message.Photo)-1].FileID
	}
	if message.Animation != nil {
		return message.Animation.FileID
	}
	if message.Video != nil {
		return message.Video.FileID
	}
	return ""
}

func hasMedia(msg *tgbotapi.Message) bool {
	return msg.Photo != nil || msg.Video != nil || msg.Animation != nil
}

func getUsername(msg *tgbotapi.Message) string {
	if msg.From.UserName {
		return "@" + msg.From.UserName
	}
	return fmt.Sprintf("[%s](tg://user?id=%s)", msg.From.String(), msg.From.ID)
}
