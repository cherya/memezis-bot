package bot

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	mmc "github.com/cherya/memezis-bot/memezis_client"

	"github.com/cherya/memezis/pkg/queue"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

type Ban interface {
	Ban(u string) error
	Permaban(u string) error
	Unban(u string) error
	IsBanned(u string) (bool, error)
}

type PublisherBot struct {
	api                *tgbotapi.BotAPI
	mc                 *mmc.Client
	qm                 *queue.Manager
	publicationChannel int64
	suggestionChannel  int64
	ownerID            int
	banHammer          Ban
	limiter            <-chan time.Time
}

func NewBot(token string, queue *queue.Manager, mc *mmc.Client, ban Ban, pubChan, sugChan int64, ownerID int) (*PublisherBot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, errors.Wrap(err, "NewBot: error creating bot api")
	}
	//api.Debug = true

	pb := &PublisherBot{
		api:                api,
		mc:                 mc,
		qm:                 queue,
		publicationChannel: pubChan,
		suggestionChannel:  sugChan,
		ownerID:            ownerID,
		banHammer:          ban,
		limiter:            time.Tick(3 * time.Second),
	}

	pb.qm.ConsumeWithDelay(ShaurmemesQueue, pb.ShaurmemesConsumer)

	return pb, nil
}

func (b *PublisherBot) SetWebhook(config tgbotapi.WebhookConfig) (tgbotapi.APIResponse, error) {

	if config.Certificate == nil {
		p := tgbotapi.Params{}
		p.AddNonEmpty("url", config.URL.String())
		if config.MaxConnections != 0 {
			p.AddNonEmpty("max_connections", strconv.Itoa(config.MaxConnections))
		}

		return b.api.MakeRequest("setWebhook", p)
	}

	params := make(map[string]string)
	params["url"] = config.URL.String()
	if config.MaxConnections != 0 {
		params["max_connections"] = strconv.Itoa(config.MaxConnections)
	}

	resp, err := b.api.UploadFile("setWebhook", params, "certificate", config.Certificate)
	if err != nil {
		return tgbotapi.APIResponse{}, err
	}

	return resp, nil
}

func (b *PublisherBot) updatesFromWebhook() tgbotapi.UpdatesChannel {
	_, err := b.SetWebhook(tgbotapi.NewWebhook("https://telegram7fdf94d0d3314c5aa1b6dd9f04317dd2.duckdns.org/telegram/" + b.api.Token))
	if err != nil {
		log.Fatal(err)
	}
	info, err := b.api.GetWebhookInfo()
	if err != nil {
		log.Fatal(err)
	}
	if info.LastErrorDate != 0 {
		log.Printf("Telegram callback failed: %s", info.LastErrorMessage)
	}
	updates := b.api.ListenForWebhook("/")
	go func() {
		err := http.ListenAndServeTLS("0.0.0.0:8443", "keys/fullchain.pem", "keys/privkey.pem", nil)
		if err != nil {
			log.Println("webhook server error", err)
		}
	}()

	return updates
}

func (b *PublisherBot) updatesFromPoll() tgbotapi.UpdatesChannel {
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

func (b *PublisherBot) messageWorker(ctx context.Context, id int, messages <-chan *tgbotapi.Message, errs chan<- error) {
	for message := range messages {
		//log.Printf("message worker %d takes message %d\n", id, message.MessageID)

		userID := b.userFromUpdate(message)
		ctx = setUserToContext(ctx, userID)

		if message.IsCommand() {
			b.handleCommand(message)
			continue
		}

		if message.Chat.ID == b.suggestionChannel {
			text := message.Text
			if text == "" {
				text = message.Caption
			}
			if !strings.Contains(text, "@"+b.api.Self.UserName) {
				continue
			}
			err := b.handleDirectSuggestionMessage(ctx, message)
			if err != nil {
				errs <- errors.Wrap(err, "error handling message")
			}
		} else if message.Chat.IsPrivate() {

			if !hasMedia(message) {
				_, err := b.send(tgbotapi.NewMessage(message.Chat.ID, unsupportedText))
				if err != nil {
					errs <- errors.Wrap(err, "error sending message")
				}
				continue
			}
			err := b.handlePrivateMessage(ctx, message)
			if err != nil {
				errs <- errors.Wrap(err, "error handling message")
			}
		}
	}
}

func (b *PublisherBot) callbackWorker(ctx context.Context, id int, callbacks <-chan *tgbotapi.CallbackQuery, errs chan<- error) {
	for callback := range callbacks {
		log.Printf("callback worker %d takes callback %d\n", id, callback.Message.MessageID)

		userID := b.userFromUpdate(callback)
		ctx = setUserToContext(ctx, userID)

		err := b.callbackQuery(ctx, callback)
		if err != nil {
			errs <- errors.Wrap(err, "error handling CallbackQuery")
		}
	}
}

func (b *PublisherBot) Start() error {
	log.Printf("Authorized on account %s", b.api.Self.UserName)

	updates := b.updatesFromPoll()
	errs := make(chan error)
	messages := make(chan *tgbotapi.Message)
	callback := make(chan *tgbotapi.CallbackQuery)

	const messageWorkersAmount = 5
	for i := 0; i < messageWorkersAmount; i++ {
		go b.messageWorker(context.Background(), i, messages, errs)
	}
	const callbackWorkersAmount = 5
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
					_, err := b.send(tgbotapi.NewMessage(update.Message.Chat.ID, banText))
					if err != nil {
						fmt.Println(errors.Wrap(err, "can't send ban msg"))
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

func (b *PublisherBot) send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	<-b.limiter
	return b.api.Send(c)
}

func (b *PublisherBot) userFromUpdate(u interface{}) int {
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

func (b *PublisherBot) Stop() {
	b.api.StopReceivingUpdates()
}

func (b *PublisherBot) handleDirectSuggestionMessage(ctx context.Context, msg *tgbotapi.Message) error {
	var postID int64
	var duplicates []int64
	var err error
	var reservMsg tgbotapi.Message

	//TODO: handle media group
	if msg.MediaGroupID != "" {
		return nil
	}

	if msg.ReplyToMessage != nil {
		if hasMedia(msg.ReplyToMessage) {
			msg = msg.ReplyToMessage
		}
	}

	text := msg.Caption

	text = strings.ReplaceAll(text, "@"+b.api.Self.UserName, "")
	text = strings.TrimSpace(text)

	if msg.Photo != nil {
		m := tgbotapi.NewMessage(b.suggestionChannel, "секундочку...")
		m.ReplyToMessageID = msg.MessageID
		reservMsg, err = b.send(m)

		postID, duplicates, err = b.savePhotoPost(ctx, text, []string{getFileIDFromMsg(msg)})
		if err != nil {
			return errors.Wrap(err, "handleDirectSuggestionMessage: can't save post")
		}
		if len(duplicates) > 0 {
			m := tgbotapi.NewMessage(b.suggestionChannel, duplicateText)
			m.ReplyToMessageID = msg.MessageID
			_, err := b.send(m)
			if err != nil {
				return errors.Wrap(err, "handleDirectSuggestionMessage: can't send message")
			}
		}
	} else if msg.Animation != nil {
		m := tgbotapi.NewMessage(b.suggestionChannel, "секундочку...")
		m.ReplyToMessageID = msg.MessageID
		reservMsg, err = b.send(m)

		postID, err = b.saveExternalPost(ctx, text, getFileIDFromMsg(msg), "gif")
		if err != nil {
			return errors.Wrap(err, "handleDirectSuggestionMessage: can't save post")
		}
	} else if msg.Video != nil && msg.MediaGroupID == "" { // TODO handle video group
		m := tgbotapi.NewMessage(b.suggestionChannel, "секундочку...")
		m.ReplyToMessageID = msg.MessageID
		reservMsg, err = b.send(m)

		postID, err = b.saveExternalPost(ctx, text, getFileIDFromMsg(msg), "video")
		if err != nil {
			return errors.Wrap(err, "handleDirectSuggestionMessage: can't save post")
		}
	}

	_, err = b.publishInternalPostVotingByID(ctx, postID, reservMsg.MessageID)
	if err != nil {
		return errors.Wrap(err, "handleDirectSuggestionMessage: can't publish post voting")
	}

	return nil
}

// хэндлер сообщений в личку боту
func (b *PublisherBot) handlePrivateMessage(ctx context.Context, msg *tgbotapi.Message) error {
	text := msg.Text
	if text == "" {
		text = msg.Caption
	}

	if msg.MediaGroupID != "" && msg.Photo != nil {
		b.handleMediaGroup(ctx, msg)
		return nil
	}

	if msg.Photo != nil {
		postID, duplicates, err := b.savePhotoPost(ctx, text, []string{getFileIDFromMsg(msg)})
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't save post")
		}
		msgID, err := b.publishPostVotingByID(ctx, postID, getUsername(msg))
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't publish post voting")
		}
		if len(duplicates) > 0 {
			m := tgbotapi.NewMessage(b.suggestionChannel, duplicateText)
			m.ReplyToMessageID = msgID
			_, err := b.send(m)
			if err != nil {
				return errors.Wrap(err, "handlePrivateMessage: can't send message")
			}
		}
	} else if msg.Animation != nil {
		postID, err := b.saveExternalPost(ctx, text, getFileIDFromMsg(msg), "gif")
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't save post")
		}
		_, err = b.publishPostVotingByID(ctx, postID, getUsername(msg))
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't publish post voting")
		}
	} else if msg.Video != nil && msg.MediaGroupID == "" { // TODO handle video group
		postID, err := b.saveExternalPost(ctx, text, getFileIDFromMsg(msg), "video")
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't save post")
		}
		_, err = b.publishPostVotingByID(ctx, postID, getUsername(msg))
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't publish post voting")
		}
	} else {
		m := tgbotapi.NewMessage(msg.Chat.ID, "я так не умею")
		m.ReplyToMessageID = msg.MessageID
		_, err := b.send(m)
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't send message")
		}
		return nil
	}

	m := tgbotapi.NewMessage(msg.Chat.ID, getSuccessText())
	m.ReplyToMessageID = msg.MessageID
	m.DisableNotification = true
	_, err := b.send(m)
	if err != nil {
		return errors.Wrap(err, "handlePrivateMessage: can't send message")
	}
	return nil
}

func (b *PublisherBot) handleCommand(msg *tgbotapi.Message) {
	cmd := msg.Command()
	// only ban command need arguments
	toBan := msg.CommandArguments()

	if isAdminCommand(cmd) && msg.From.ID != b.ownerID {
		return
	}

	var m tgbotapi.MessageConfig
	switch cmd {
	case "ban":
		err := b.banHammer.Ban(toBan)
		if err != nil {
			log.Printf("can't ban %s. error: %s", toBan, err)
			m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("что то пошло не так: %s", err))
		}
		m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("забанен"))
	case "permaban":
		err := b.banHammer.Permaban(toBan)
		if err != nil {
			log.Printf("can't permaban %s. error: %s", toBan, err)
			m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("что то пошло не так: %s", err))
		}
		m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("совсем забанен"))
	case "unban":
		err := b.banHammer.Unban(toBan)
		if err != nil {
			log.Printf("can't unban %s. error: %s", toBan, err)
			m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("что то пошло не так: %s", err))
		}
		m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("разбанен"))
	case "queue":
		qInfo, err := b.mc.QueueInfo(ShaurmemesQueue)
		if err != nil {
			log.Printf("can't unban %s. error: %s", toBan, err)
			m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("что то пошло не так: %s", err))
		}
		if qInfo.Length == 0 {
			m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("в очереди ничего нет"))
		} else {
			m = tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("постов в очереди: %d \nдо: %s", qInfo.Length, qInfo.DueTime.Format("15:04")))
		}
	}

	m.ReplyToMessageID = msg.MessageID
	_, err := b.send(m)
	if err != nil {
		log.Printf("can't ansewer to comand: %s", err)
	}
}

var adminCommands = map[string]struct{}{
	"ban":      {},
	"permaban": {},
	"unban":    {},
}

func isAdminCommand(cmd string) bool {
	_, ok := adminCommands[cmd]
	return ok
}

func (b *PublisherBot) isBanned(message *tgbotapi.Message) bool {
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
