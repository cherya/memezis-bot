package bot

import (
	"context"

	"github.com/cherya/memezis/pkg/memezis"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (b *MemezisBot) processDuplicates(ctx context.Context, postID int64, replyMsgID int) error {
	duplicates, err := b.mc.FindDuplicatesByPostID(ctx, &memezis.FindDuplicatesByPostIDRequest{
		Id:                   postID,
		Limit:                5,
	})
	if err != nil {
		log.Errorf("can't get duplicates for post %d, %s", postID, err)
	}
	if hasDuplicates(duplicates) {
		m := tgbotapi.NewMessage(b.adminChannel, getDuplicatesText(duplicates))
		m.ReplyToMessageID = replyMsgID
		_, err := b.send(m)
		if err != nil {
			return errors.Wrap(err, "processDuplicates: can't send message")
		}
	}

	return nil
}
