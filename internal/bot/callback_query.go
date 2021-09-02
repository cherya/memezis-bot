package bot

import (
	"context"
	"encoding/json"

	"github.com/cherya/memezis/pkg/memezis"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	PublishStatusEnqueued = "enqueued"
	PublishStatusDeclined = "declined"
)

func (b *MemezisBot) callbackQuery(ctx context.Context, callback *tgbotapi.CallbackQuery) error {
	data := new(ButtonData)
	err := json.Unmarshal([]byte(callback.Data), &data)
	if err != nil {
		return errors.Wrap(err, "callbackQuery: can't unmarshal button data")
	}

	userID := userIDFromContext(ctx)

	switch data.ActionType {
	// voting buttons
	case callbackActionTypeUpVote, callbackActionTypeDownVote:
		var resp *memezis.Vote
		if data.ActionType == callbackActionTypeUpVote {
			resp, err = b.mc.UpVote(ctx, &memezis.VoteRequest{
				UserID: userID,
				PostID: data.PostID,
			})
		} else {
			resp, err = b.mc.DownVote(ctx, &memezis.VoteRequest{
				UserID: userID,
				PostID: data.PostID,
			})
		}
		if err != nil {
			return errors.Wrap(err, "callbackQuery: error voting post")
		}
		if !resp.Accepted {
			_, _ = b.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, getText(TextTypeOwnPostVote)))
			return nil
		}

		var markupUpdate tgbotapi.EditMessageReplyMarkupConfig

		switch resp.Status {
		case PublishStatusEnqueued, PublishStatusDeclined:
			markupUpdate = tgbotapi.NewEditMessageReplyMarkup(
				b.adminChannel, callback.Message.MessageID, createVoteEndKeyboard(data.PostID, resp.Up, resp.Down))
		default:
			// checking is vote changed
			if data.ActionType == callbackActionTypeUpVote && data.UpVotes == resp.Up {
				_, _ = b.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, getText(TextTypeVotedAlready)))
				return nil
			}
			if data.ActionType == callbackActionTypeDownVote && data.DownVotes == resp.Down {
				_, _ = b.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, getText(TextTypeVotedAlready)))
				return nil
			}
			markupUpdate = tgbotapi.NewEditMessageReplyMarkup(
				b.adminChannel, callback.Message.MessageID, createVotingKeyboard(data.PostID, resp.Up, resp.Down))
		}

		_, err = b.send(markupUpdate)
		if err != nil {
			return errors.Wrap(err, "callbackQuery: error updating markup post")
		}

		_, err = b.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, getText(TextTypeVoteCallback)))
		return errors.Wrap(err, "callbackQuery: can't answer to callback")
	//button "scheduled"
	case callbackActionTypeScheduled:
		_, err = b.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, getText(TextTypeQueued)))
		return errors.Wrap(err, "callbackQuery: can't answer to scheduled callback")
	// button "decline"
	case callbackActionTypeDeclined:
		_, err = b.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, getText(TextTypeDeclined)))
		return errors.Wrap(err, "callbackQuery: can't answer to declined callback")
	case callbackActionTypeConfirmUpload:
		markupUpdate := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, createLoadingKeyboard())
		_, err = b.send(markupUpdate)
		if err != nil {
			log.Error(errors.Wrap(err, "callbackQuery: can't update loadnubg markup"))
		}
		postID, err := b.uploadNewPost(ctx, callback.Message, callback.From)
		if err != nil {
			return errors.Wrap(err, "callbackQuery: can't answer to confirm callback")
		}
		err = b.uc.Set(postID, getUserFullName(callback.From), callback.From.ID)
		if err != nil {
			log.Error(errors.Wrap(err, "callbackQuery: can't save user to cache"))
		}
		markupUpdate = tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, createConfirmedPostKeyboard(false))
		_, err = b.send(markupUpdate)
		if err != nil {
			return errors.Wrap(err, "callbackQuery: can't update confirmed markup")
		}
		_, err = b.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, getText(TextTypeSucessUpload)))
		return errors.Wrap(err, "callbackQuery: can't answer to confirm callback")
	case callbackActionTypeConfirmUploadAnon:
		markupUpdate := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, createLoadingKeyboard())
		_, err = b.send(markupUpdate)
		if err != nil {
			log.Error(errors.Wrap(err, "callbackQuery: can't update loadnubg markup"))
		}
		_, err = b.uploadNewPost(ctx, callback.Message, callback.From)
		if err != nil {
			return errors.Wrap(err, "callbackQuery: can't answer to confirm callback")
		}
		markupUpdate = tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, createConfirmedPostKeyboard(true))
		_, err = b.send(markupUpdate)
		if err != nil {
			return errors.Wrap(err, "callbackQuery: can't update confirmed anon markup")
		}
		_, err = b.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, getText(TextTypeSucessUpload)))
		return errors.Wrap(err, "callbackQuery: can't answer to confirmed anon callback")
	case callbackActionTypeDeclineUpload:
		markupUpdate := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, createDeclinedPostKeyboard())
		_, err = b.send(markupUpdate)
		if err != nil {
			return errors.Wrap(err, "callbackQuery: can't update declined markup")
		}
		_, err = b.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return errors.Wrap(err, "callbackQuery: can't answer to declined callback")
	case callbackActionTypeRemoveCaption:
		editMessage := tgbotapi.NewEditMessageCaption(callback.Message.Chat.ID, callback.Message.MessageID, "")
		keyboard := createConfirmationKeyboard("")
		editMessage.ReplyMarkup = &keyboard
		_, err = b.send(editMessage)
		if err != nil {
			return errors.Wrap(err, "callbackQuery: can't remove caption")
		}
		_, err = b.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))
		return errors.Wrap(err, "callbackQuery: can't answer to remove caption callback")
	}

	return nil
}

// AnswerCallbackQuery sends a response to an inline query callback.
func (b *MemezisBot) AnswerCallbackQuery(config tgbotapi.CallbackConfig) (tgbotapi.APIResponse, error) {
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
