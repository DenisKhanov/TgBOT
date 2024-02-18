package services

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type YandexTranslate interface {
	TranslateAPI(text string) (string, error)
	DetectLangAPI(text string) (string, error)
}

func (b *TgBotServices) translateText(update *tgbotapi.Update) {
	translatedText, err := b.YandexTranslate.TranslateAPI(update.Message.Text)
	if err != nil {
		logrus.Error(err)
		return
	}
	msg := tgbotapi.NewMessage(b.ChatID, translatedText)
	msg.ReplyToMessageID = update.Message.MessageID
	b.Bot.Send(msg)
}
