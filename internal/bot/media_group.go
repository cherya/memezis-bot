package bot

import (
	"context"
	"sort"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	log "github.com/sirupsen/logrus"
)

type mediaValue struct {
	FileID    string
	MessageID int
}

type mediaSlice struct {
	sync.Mutex
	Values []mediaValue
}

func (ms *mediaSlice) Len() int {
	return len(ms.Values)
}

func (ms *mediaSlice) Less(a, b int) bool {
	return ms.Values[a].MessageID < ms.Values[b].MessageID
}

func (ms *mediaSlice) Swap(a, b int) {
	ms.Values[a], ms.Values[b] = ms.Values[b], ms.Values[a]
}

func (ms *mediaSlice) Append(v mediaValue) {
	ms.Lock()
	defer ms.Unlock()

	ms.Values = append(ms.Values, v)
}

func (ms *mediaSlice) GetSortedValues() []string {
	sort.Sort(ms)
	vals := make([]string, 0, len(ms.Values))
	for _, v := range ms.Values {
		vals = append(vals, v.FileID)
	}
	return vals
}

var mediaGroups sync.Map

// складываем сообщения по MediaGroupID, через 5 секунд забираем все что получилось
// 2 секунды – огромный запас, кажется что все сообщения из одной группы приходят моментально
// в апи нихуя нет, как делать нормально – неизвестно
func (b *MemezisBot) handleMediaGroup(ctx context.Context, msg *tgbotapi.Message) {
	text := msg.Text
	if text == "" {
		text = msg.Caption
	}

	ms := &mediaSlice{
		Values: []mediaValue{
			{
				FileID:    getFileIDFromMsg(msg),
				MessageID: msg.MessageID,
			},
		},
	}

	if mg, ok := mediaGroups.LoadOrStore(msg.MediaGroupID, ms); !ok {
		go func(mediaGroupID string) {
			time.Sleep(2 * time.Second)
			m := tgbotapi.NewMessage(msg.Chat.ID, getSuccessText())
			m.ReplyToMessageID = msg.MessageID
			_, err := b.send(m)
			if err != nil {
				log.Error("handlePrivateMessage: can't send message", err)
			}

			val, _ := mediaGroups.Load(mediaGroupID)
			media := val.(*mediaSlice)
			postID, _, err := b.savePhotoPost(ctx, text, media.GetSortedValues(), msg.Time())
			if err != nil {
				log.Error("can't save post", err)
				return
			}
			mediaGroups.Delete(mediaGroupID)
			_, err = b.publishPostVotingByID(ctx, postID, getUsername(msg))
			if err != nil {
				log.Error("can't publish post voting", err)
				return
			}
		}(msg.MediaGroupID)
	} else {
		media := mg.(*mediaSlice)
		if text == "" {
			text = msg.Caption
		}
		media.Append(mediaValue{
			FileID:    getFileIDFromMsg(msg),
			MessageID: msg.MessageID,
		})
		mediaGroups.Store(msg.MediaGroupID, media)
	}
}
