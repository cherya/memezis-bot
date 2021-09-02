package bot

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cherya/memezis-bot/internal/dailyword"
	"github.com/cherya/memezis/pkg/memezis"
	"github.com/cherya/memezis/pkg/queue"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type MemezisBot struct {
	api                *tgbotapi.BotAPI
	mc                 memezis.MemezisClient
	qm                 *queue.Manager
	wg                 *dailyword.WordGenerator
	publicationChannel int64
	adminChannel       int64
	ownerID            int
	banHammer          Ban
	uc                 UserCache
	admins             []tgbotapi.ChatMember
	limiter            <-chan time.Time
	callbackAtom       *sync.Map
}

type Ban interface {
	Ban(u string) error
	Permaban(u string) error
	Unban(u string) error
	IsBanned(u string) (bool, error)
}

type UserCache interface {
	Set(postId int64, name string, id int) error
	GetName(postId int64) (string, error)
	GetID(postId int64) (string, error)
}

const (
	messageWorkersAmount  = 5
	callbackWorkersAmount = 5
)

func NewBot(api *tgbotapi.BotAPI, queue *queue.Manager, mc memezis.MemezisClient, wg *dailyword.WordGenerator, ban Ban, uc UserCache, pubChan, sugChan int64, ownerID int) (*MemezisBot, error) {
	pb := &MemezisBot{
		api:                api,
		mc:                 mc,
		qm:                 queue,
		wg:                 wg,
		publicationChannel: pubChan,
		adminChannel:       sugChan,
		ownerID:            ownerID,
		banHammer:          ban,
		uc:                 uc,
		limiter:            time.Tick(3 * time.Second),
		callbackAtom:       &sync.Map{},
	}

	admins, err := api.GetChatAdministrators(tgbotapi.ChatAdministratorsConfig{
		ChatConfig: tgbotapi.ChatConfig{
			ChatID: sugChan,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "NewBot: can't get channel admins")
	}

	pb.admins = admins

	pb.qm.Consume(context.Background(), ShaurmemesQueue, time.Second*10, pb.ShaurmemesConsumer)

	return pb, nil
}

func (b *MemezisBot) updatesFromPoll() tgbotapi.UpdatesChannel {
	p := tgbotapi.Params{}
	p.AddNonEmpty("url", "https://telegram7fdf94d0d3314c5aa1b6dd9f04317dd2.duckdns.org/telegram/"+b.api.Token)
	_, err := b.api.MakeRequest("deleteWebhook", p)
	if err != nil {
		log.Fatal("can't remove webhook")
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 20
	updates := b.api.GetUpdatesChan(u)

	return updates
}

func (b *MemezisBot) Start() error {
	log.Infof("Authorized on account %s", b.api.Self.UserName)

	updates := b.updatesFromPoll()
	errs := make(chan error)
	messages := make(chan *tgbotapi.Message, 5)
	callback := make(chan *tgbotapi.CallbackQuery, 5)

	for i := 0; i < messageWorkersAmount; i++ {
		go b.messageWorkerWrapper(context.Background(), i, messages, errs)
	}
	for i := 0; i < callbackWorkersAmount; i++ {
		go b.callbackWorker(context.Background(), i, callback, errs)
	}

	for {
		select {
		case err := <-errs:
			fmt.Println("update error:", err)
		case update := <-updates:
			if update.Message != nil {
				if b.isBanned(update.Message) {
					if update.Message.Chat.IsPrivate() {
						_, err := b.send(tgbotapi.NewMessage(update.Message.Chat.ID, getText(TextTypeBanned)))
						if err != nil {
							fmt.Println(errors.Wrap(err, "can't send ban msg"))
						}
					}
					continue
				}
				messages <- update.Message
			}
			if update.CallbackQuery != nil {
				callback <- update.CallbackQuery
			}
		}
	}
}

func (b *MemezisBot) Stop() {
	b.api.StopReceivingUpdates()
}

func (b *MemezisBot) send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	// limit all messages except callbacks
	if _, ok := c.(tgbotapi.CallbackConfig); !ok {
		<-b.limiter
	}
	return b.api.Send(c)
}

func (b *MemezisBot) isAdmin(userID int) bool {
	for _, a := range b.admins {
		if a.User.ID == userID {
			return true
		}
	}
	return false
}
