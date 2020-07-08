package bot

import (
	"encoding/json"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type callbackActionType int

const (
	callbackActionTypeVote      = 1
	callbackActionTypeScheduled = 2
	callbackActionTypeDeclined  = 3
	callbackActionTypeUpVote    = 4
	callbackActionTypeDownVote  = 5
)

type ButtonData struct {
	PostID     int64              `json:"id"`
	ActionType callbackActionType `json:"a"`
	DownVotes  int64              `json:"dv"`
	UpVotes    int64              `json:"uv"`
}

func (b ButtonData) IsUp() bool {
	return b.ActionType == callbackActionTypeUpVote
}

func (b ButtonData) String() string {
	data, _ := json.Marshal(b)
	return string(data)
}

func readButtonData(rawData string) ButtonData {
	var data ButtonData
	_ = json.Unmarshal([]byte(rawData), &data)
	return data
}

func createVotingKeyboard(postID, up, down int64) tgbotapi.InlineKeyboardMarkup {
	upData := ButtonData{postID, callbackActionTypeUpVote, down, up}
	downData := ButtonData{postID, callbackActionTypeDownVote, down, up}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("👍 %d", up), upData.String()),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("👎 %d", down), downData.String()),
		),
	)
}

func createVoteEndKeyboard(postID, up, down int64) tgbotapi.InlineKeyboardMarkup {
	text := "Пост добавлен в очередь ⏱"
	data := ButtonData{
		PostID:     postID,
		ActionType: callbackActionTypeScheduled,
	}
	if up < down {
		text = "Пост отклонен 💩"
		data.ActionType = callbackActionTypeDeclined
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(text, data.String()),
		),
	)
}
