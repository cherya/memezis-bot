package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cherya/memezis/pkg/memezis"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/golang/protobuf/ptypes/empty"
	log "github.com/sirupsen/logrus"
)

const (
	// temporary ban
	BAN = "ban"
	// permanent ban
	PERMABAN = "permaban"
	// remove from ban
	UNBAN = "unban"
	// get queue status
	QUEUE = "queue"
	// get gender of the day
	GENDER = "gender"
	// get random meme
	ROFL = "rofl"
)

var adminCommands = map[string]struct{}{
	"ban":      {},
	"permaban": {},
	"unban":    {},
}

func isAdminCommand(cmd string) bool {
	_, ok := adminCommands[cmd]
	return ok
}

func (b *MemezisBot) handleCommand(msg *tgbotapi.Message) {
	cmd := msg.Command()
	// now only ban command need arguments
	toBan := msg.CommandArguments()

	if isAdminCommand(cmd) && msg.From.ID != b.ownerID {
		return
	}

	var m tgbotapi.MessageConfig
	switch cmd {
	case BAN:
		err := b.banHammer.Ban(toBan)
		if err != nil {
			log.Errorf("can't ban %s. error: %s", toBan, err)
			m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("что то пошло не так: %s", err))
		}
		m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("забанен"))
	case PERMABAN:
		err := b.banHammer.Permaban(toBan)
		if err != nil {
			log.Errorf("can't ban %s. error: %s", toBan, err)
			m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("что то пошло не так: %s", err))
		}
		m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("совсем забанен"))
	case UNBAN:
		err := b.banHammer.Unban(toBan)
		if err != nil {
			log.Errorf("can't unban %s. error: %s", toBan, err)
			m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("что то пошло не так: %s", err))
		}
		m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("разбанен"))
	case QUEUE:
		qInfo, err := b.mc.GetQueueInfo(context.Background(), &memezis.GetQueueInfoRequest{Queue: ShaurmemesQueue})
		if err != nil {
			log.Errorf("can't unban %s. error: %s", toBan, err)
			m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("что то пошло не так: %s", err))
			_, err := b.send(m)
			if err != nil {
				log.Errorf("can't answer to command: %s", err)
			}
			return
		}
		if qInfo.Length == 0 {
			m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("в очереди ничего нет"))
		} else {
			due := fromProtoTime(qInfo.DueTime)
			loc, err := time.LoadLocation("Europe/Moscow")
			if err == nil {
				due = due.In(loc)
			}
			m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("постов в очереди: %d \nдо: %s", qInfo.Length, due.Format("15:04 MST")))
		}
	case GENDER:
		g, err := b.wg.Get(strconv.FormatInt(msg.Chat.ID, 10))
		if err != nil {
			m := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("что то пошло не так"))
			_, err := b.send(m)
			if err != nil {
				log.Errorf("can't answer to command: %s", err)
			}
			return
		}
		m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Гендер дня: *%s*", strings.ToUpper(g)))
	case ROFL:
		post, err := b.mc.GetRandomPost(context.Background(), &empty.Empty{})
		if err != nil {
			m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("что то пошло не так"))
			_, err := b.send(m)
			if err != nil {
				log.Errorf("can't answer to command: %s", err)
			}
			return
		}

		p := tgbotapi.NewPhotoUpload(msg.Chat.ID, nil)
		p.FileID = post.Media[0].URL
		p.UseExisting = true
		p.Caption = post.Text + "\n@shaurmemes"
		_, err = b.send(p)
		if err != nil {
			log.Errorf("can't answer to command: %s", err)
		}
	}

	if m.Text != "" {
		m.ReplyToMessageID = msg.MessageID
		m.ParseMode = tgbotapi.ModeMarkdown
		_, err := b.send(m)
		if err != nil {
			log.Errorf("can't answer to command: %s", err)
		}
	}
}
