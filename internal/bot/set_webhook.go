package bot

import (
	"net/http"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"
)

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
