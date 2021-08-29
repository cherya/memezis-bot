package bot

import (
	"fmt"
	"time"

	"github.com/cherya/memezis/pkg/memezis"

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

func hasDuplicates(d *memezis.FindDuplicatesByPostIDResponse) bool {
	if d == nil {
		return false
	}
	return len(d.Complete) > 0 || len(d.Likely) > 0
}

func userFromUpdate(u interface{}) int {
	switch u.(type) {
	case *tgbotapi.Message:
		return u.(*tgbotapi.Message).From.ID
	case *tgbotapi.InlineQuery:
		return u.(*tgbotapi.InlineQuery).From.ID
	case *tgbotapi.CallbackQuery:
		return u.(*tgbotapi.CallbackQuery).From.ID
	case *tgbotapi.ChosenInlineResult:
		return u.(*tgbotapi.ChosenInlineResult).From.ID
	default:
		return 0
	}
}

func mentionUser(msg *tgbotapi.Message, user tgbotapi.User) bool {
	for _, m := range append(msg.Entities, msg.CaptionEntities...) {
		if m.User.ID == user.ID {
			return true
		}
	}
	return false
}
