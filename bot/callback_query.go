package bot

import (
	"context"
	"encoding/json"

	mmc "github.com/cherya/memezis-bot/memezis_client"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

func (b *PublisherBot) callbackQuery(ctx context.Context, callback *tgbotapi.CallbackQuery) error {
	data := new(ButtonData)
	err := json.Unmarshal([]byte(callback.Data), &data)
	if err != nil {
		return errors.Wrap(err, "callbackQuery: can't unmarshal button data")
	}

	userID := userFromContext(ctx)

	switch data.ActionType {
	// voting buttons
	case callbackActionTypeUpVote, callbackActionTypeDownVote:
		resp, err := b.mc.VoteByPostID(data.PostID, userID, data.IsUp())
		if err != nil {
			return errors.Wrap(err, "callbackQuery: error voting post")
		}

		var markupUpdate tgbotapi.EditMessageReplyMarkupConfig

		switch resp.Status {
		case mmc.PublishStatusEnqueued, mmc.PublishStatusDeclined:
			markupUpdate = tgbotapi.NewEditMessageReplyMarkup(
				b.suggestionChannel, callback.Message.MessageID, createVoteEndKeyboard(data.PostID, resp.Up, resp.Down))
		default:
			// checking is vote changed
			if data.ActionType == callbackActionTypeUpVote && data.UpVotes == resp.Up {
				_, err = b.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, getAlreadyVotedCallbackTexts()))
				return nil
			}
			if data.ActionType == callbackActionTypeDownVote && data.DownVotes == resp.Down {
				_, err = b.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, getAlreadyVotedCallbackTexts()))
				return nil
			}
			markupUpdate = tgbotapi.NewEditMessageReplyMarkup(
				b.suggestionChannel, callback.Message.MessageID, createVotingKeyboard(data.PostID, resp.Up, resp.Down))
		}

		_, err = b.send(markupUpdate)
		if err != nil {
			return errors.Wrap(err, "callbackQuery: error updating markup post")
		}

		_, err = b.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, getVoteCallbackTexts()))
		if err != nil {
			return errors.Wrap(err, "callbackQuery: can't answer to callback")
		}
	//button "scheduled"
	case callbackActionTypeScheduled:
		_, err = b.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, queuedText))
		if err != nil {
			return errors.Wrap(err, "callbackQuery: can't answer to callback")
		}
	// button "declinec"
	case callbackActionTypeDeclined:
		_, err = b.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, declinedText))
		if err != nil {
			return errors.Wrap(err, "callbackQuery: can't answer to callback")
		}
	}

	return nil
}

// AnswerCallbackQuery sends a response to an inline query callback.
func (b *PublisherBot) AnswerCallbackQuery(config tgbotapi.CallbackConfig) (tgbotapi.APIResponse, error) {
	p := tgbotapi.Params{}

	p.AddNonEmpty("callback_query_id", config.CallbackQueryID)
	if config.Text != "" {
		p.AddNonEmpty("text", config.Text)
	}
	p.AddBool("show_alert", config.ShowAlert)
	if config.URL != "" {
		p.AddNonEmpty("url", config.URL)
	}
	p.AddNonZero("cache_time", config.CacheTime)

	return b.api.MakeRequest("answerCallbackQuery", p)
}
