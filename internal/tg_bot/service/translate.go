package service

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// translateText translates the text from the provided update and sends it as a reply.
func (b *TgBotServices) translateText(update *tgbotapi.Update) error {
	translatedText, err := b.Translate.TranslateAPI(update.Message.Text)
	if err != nil {
		logrus.WithError(err).Error("Translation failed")
		return err
	}
	return b.sendMessage(b.ChatID, translatedText, update.Message.MessageID, nil)
}
