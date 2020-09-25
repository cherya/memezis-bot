package bot

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/cherya/memezis-bot/internal/dailyword"
	log "github.com/sirupsen/logrus"

	"github.com/cherya/memezis/pkg/memezis"
	"github.com/cherya/memezis/pkg/queue"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

type MemezisBot struct {
	api                *tgbotapi.BotAPI
	mc                 memezis.MemezisClient
	qm                 *queue.Manager
	wg                 *dailyword.WordGenerator
	publicationChannel int64
	suggestionChannel  int64
	ownerID            int
	banHammer          Ban
	limiter            <-chan time.Time
}

type Ban interface {
	Ban(u string) error
	Permaban(u string) error
	Unban(u string) error
	IsBanned(u string) (bool, error)
}

func NewBot(token string, queue *queue.Manager, mc memezis.MemezisClient, wg *dailyword.WordGenerator, ban Ban, pubChan, sugChan int64, ownerID int) (*MemezisBot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, errors.Wrap(err, "NewBot: error creating bot api")
	}
	//api.Debug = true

	pb := &MemezisBot{
		api:                api,
		mc:                 mc,
		qm:                 queue,
		wg:                 wg,
		publicationChannel: pubChan,
		suggestionChannel:  sugChan,
		ownerID:            ownerID,
		banHammer:          ban,
		limiter:            time.Tick(3 * time.Second),
	}

	pb.qm.Consume(context.Background(), ShaurmemesQueue, time.Second*10, pb.ShaurmemesConsumer)

	return pb, nil
}

func (b *MemezisBot) SetWebhook(config tgbotapi.WebhookConfig) (tgbotapi.APIResponse, error) {

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

func (b *MemezisBot) updatesFromWebhook() tgbotapi.UpdatesChannel {
	_, err := b.SetWebhook(tgbotapi.NewWebhook("https://telegram7fdf94d0d3314c5aa1b6dd9f04317dd2.duckdns.org/telegram/" + b.api.Token))
	if err != nil {
		log.Fatal(err)
	}
	info, err := b.api.GetWebhookInfo()
	if err != nil {
		log.Fatal(err)
	}
	if info.LastErrorDate != 0 {
		log.Error("Telegram callback failed: %s", info.LastErrorMessage)
	}
	updates := b.api.ListenForWebhook("/")
	go func() {
		err := http.ListenAndServeTLS("0.0.0.0:8443", "keys/fullchain.pem", "keys/privkey.pem", nil)
		if err != nil {
			log.Error("webhook server error", err)
		}
	}()

	return updates
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

func (b *MemezisBot) messageWorker(ctx context.Context, id int, messages <-chan *tgbotapi.Message, errs chan<- error) {
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
				f := tgbotapi.NewForward(b.suggestionChannel, message.Chat.ID, message.MessageID)
				_, err = b.send(f)
				if err != nil {
					errs <- errors.Wrap(err, "error sending message")
				}
				continue
			}
			err := b.handlePrivateMessage(ctx, message)
			if err != nil {
				errs <- errors.Wrap(err, "error handling message")
			}
		} else {
			if !strings.Contains(message.Text, "@"+b.api.Self.UserName) {
				continue
			}
			err := b.handleChatMessage(ctx, message)
			if err != nil {
				errs <- errors.Wrap(err, "error handling chat message")
			}
		}
	}
}

func (b *MemezisBot) callbackWorker(ctx context.Context, id int, callbacks <-chan *tgbotapi.CallbackQuery, errs chan<- error) {
	for callback := range callbacks {
		//log.Printf("callback worker %d takes callback %d\n", id, callback.Message.MessageID)

		userID := b.userFromUpdate(callback)
		ctx = setUserToContext(ctx, userID)

		err := b.callbackQuery(ctx, callback)
		if err != nil {
			errs <- errors.Wrap(err, "error handling CallbackQuery")
		}
	}
}

func (b *MemezisBot) Start() error {
	log.Infof("Authorized on account %s", b.api.Self.UserName)

	updates := b.updatesFromPoll()
	errs := make(chan error)
	messages := make(chan *tgbotapi.Message, 5)
	callback := make(chan *tgbotapi.CallbackQuery, 5)

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

func (b *MemezisBot) send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	// limit all messages except callbacks
	if _, ok := c.(tgbotapi.CallbackConfig); !ok {
		<-b.limiter
	}
	return b.api.Send(c)
}

func (b *MemezisBot) userFromUpdate(u interface{}) int {
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

func (b *MemezisBot) Stop() {
	b.api.StopReceivingUpdates()
}

func (b *MemezisBot) handleDirectSuggestionMessage(ctx context.Context, msg *tgbotapi.Message) error {
	var postID int64
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

		postID, err = b.savePhotoPost(ctx, text, []string{getFileIDFromMsg(msg)}, msg.Time())
		if err != nil {
			return errors.Wrap(err, "handleDirectSuggestionMessage: can't save post")
		}
		err = b.processDuplicates(ctx, postID, msg.MessageID)
		if err != nil {
			log.Errorf("error processing duplicates %s", err)
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

func (b *MemezisBot) handleChatMessage(ctx context.Context, msg *tgbotapi.Message) error {
	var postID int64
	var err error

	sender := getUsername(msg)
	text := msg.Caption
	text = strings.ReplaceAll(text, "@"+b.api.Self.UserName, "")
	text = strings.TrimSpace(text)

	if msg.ReplyToMessage == nil || !hasMedia(msg.ReplyToMessage) {
		return nil
	}

	msg = msg.ReplyToMessage

	if msg.Photo != nil {
		m := tgbotapi.NewMessage(msg.Chat.ID, "украл")
		m.ReplyToMessageID = msg.MessageID
		_, err = b.send(m)

		postID, err = b.savePhotoPost(ctx, text, []string{getFileIDFromMsg(msg)}, msg.Time())
		if err != nil {
			return errors.Wrap(err, "handleChatMessage: can't save post")
		}
	} else if msg.Animation != nil {
		m := tgbotapi.NewMessage(msg.Chat.ID, "украл")
		m.ReplyToMessageID = msg.MessageID
		_, err = b.send(m)

		postID, err = b.saveExternalPost(ctx, text, getFileIDFromMsg(msg), "gif")
		if err != nil {
			return errors.Wrap(err, "handleChatMessage: can't save post")
		}
	} else if msg.Video != nil && msg.MediaGroupID == "" { // TODO handle video group
		m := tgbotapi.NewMessage(msg.Chat.ID, "украл")
		m.ReplyToMessageID = msg.MessageID
		_, err = b.send(m)

		postID, err = b.saveExternalPost(ctx, text, getFileIDFromMsg(msg), "video")
		if err != nil {
			return errors.Wrap(err, "handleChatMessage: can't save post")
		}
	}

	votingMsgID, err := b.publishPostVotingByID(ctx, postID, sender)
	err = b.processDuplicates(ctx, postID, votingMsgID)
	if err != nil {
		log.Errorf("error processing duplicates %s", err)
	}

	if err != nil {
		return errors.Wrap(err, "handleChatMessage: can't publish post voting")
	}

	return nil
}

// хэндлер сообщений (не команд) в личку боту
func (b *MemezisBot) handlePrivateMessage(ctx context.Context, msg *tgbotapi.Message) error {
	text := msg.Text
	if text == "" {
		text = msg.Caption
	}

	if msg.MediaGroupID != "" && msg.Photo != nil {
		b.handleMediaGroup(ctx, msg)
		return nil
	}

	m := tgbotapi.NewMessage(msg.Chat.ID, getSuccessText())
	m.ReplyToMessageID = msg.MessageID
	_, err := b.send(m)
	if err != nil {
		return errors.Wrap(err, "handlePrivateMessage: can't send message")
	}

	if msg.Photo != nil {
		postID, err := b.savePhotoPost(ctx, text, []string{getFileIDFromMsg(msg)}, msg.Time())
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't save post")
		}
		votingMsgID, err := b.publishPostVotingByID(ctx, postID, getUsername(msg))
		if err != nil {
			return errors.Wrap(err, "handlePrivateMessage: can't publish post voting")
		}
		err = b.processDuplicates(ctx, postID, votingMsgID)
		if err != nil {
			log.Errorf("error processing duplicates %s", err)
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

	return nil
}

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

var adminCommands = map[string]struct{}{
	"ban":      {},
	"permaban": {},
	"unban":    {},
}

func isAdminCommand(cmd string) bool {
	_, ok := adminCommands[cmd]
	return ok
}

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

func (b *MemezisBot) processDuplicates(ctx context.Context, postID int64, replyMsgID int) error {
	duplicates, err := b.mc.FindDuplicatesByPostID(ctx, &memezis.FindDuplicatesByPostIDRequest{Id: postID})
	if err != nil {
		log.Errorf("can't get duplicates for post %d, %s", postID, err)
	}
	if hasDuplicates(duplicates) {
		m := tgbotapi.NewMessage(b.suggestionChannel, getDuplicatesText(duplicates))
		m.ReplyToMessageID = replyMsgID
		_, err := b.send(m)
		if err != nil {
			return errors.Wrap(err, "processDuplicates: can't send message")
		}
	}

	return nil
}
