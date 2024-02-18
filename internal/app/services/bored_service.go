package services

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Boring interface {
	BoredAPI() (string, error)
}

func (b *TgBotServices) GenerateActivityMsg() {
	text, err := b.Boring.BoredAPI()
	ruText, err := b.YandexTranslate.TranslateAPI(text)
	var msg tgbotapi.MessageConfig
	if err == nil {
		msg = tgbotapi.NewMessage(b.ChatID, ruText)
	} else {
		msg = tgbotapi.NewMessage(b.ChatID, "К сожалению в данный момент я не могу дотянуться до знаний")

	}
	b.Bot.Send(msg)
}
