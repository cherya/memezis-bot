package bot

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gogo/protobuf/types"
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
	if msg.From.UserName != "" {
		return "@" + msg.From.UserName
	}
	return fmt.Sprintf("[%s](tg://user?id=%d)", msg.From.String(), msg.From.ID)
}

func fromProtoTime(timestamp *types.Timestamp) time.Time {
	t, _ := types.TimestampFromProto(timestamp)
	return t
}

func toProtoTime(time time.Time) *types.Timestamp {
	t, _ := types.TimestampProto(time)
	return t
}

func hasDuplicates(d *Duplicates) bool {
	return len(d.Complete) > 0 || len(d.Likely) > 0 || len(d.Similar) > 0
}