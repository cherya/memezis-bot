package bot

import (
	"encoding/json"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type callbackActionType int

const (
	callbackActionTypeVote = iota
	callbackActionTypeScheduled
	callbackActionTypeDeclined
	callbackActionTypeUpVote
	callbackActionTypeDownVote
	callbackActionTypeConfirmUpload
	callbackActionTypeConfirmUploadAnon
	callbackActionTypeDeclineUpload
	callbackActionTypeRemoveCaption
	callbackActionTypeOther
)

type ButtonData struct {
	PostID     int64              `json:"id"`
	ActionType callbackActionType `json:"a"`
	DownVotes  int64              `json:"dv"`
	UpVotes    int64              `json:"uv"`
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

func createConfirmationKeyboard(text string) tgbotapi.InlineKeyboardMarkup {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟢 отправить", ButtonData{ActionType: callbackActionTypeConfirmUpload}.String()),
			tgbotapi.NewInlineKeyboardButtonData("😷 анонимно", ButtonData{ActionType: callbackActionTypeConfirmUploadAnon}.String()),
			tgbotapi.NewInlineKeyboardButtonData("❌ отмена", ButtonData{ActionType: callbackActionTypeDeclineUpload}.String()),
		),
	)
	if text != "" {
		kb.InlineKeyboard = append(kb.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ убрать текст", ButtonData{ActionType: callbackActionTypeRemoveCaption}.String()),
		))
	}
	return kb
}

func createConfirmedPostKeyboard(anon bool) tgbotapi.InlineKeyboardMarkup {
	if anon {
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🟢😷 принято анонимно", ButtonData{ActionType: callbackActionTypeOther}.String()),
			),
		)
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟢 принято", ButtonData{ActionType: callbackActionTypeOther}.String()),
		),
	)
}

func createDeclinedPostKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔴 отменено", ButtonData{ActionType: callbackActionTypeOther}.String()),
		),
	)
}

func createLoadingKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(getText(TextTypeLoading), ButtonData{ActionType: callbackActionTypeOther}.String()),
		),
	)
}
