package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// changeHistorySize updates the maximum size limit for the dialog history based on user input.
func (b *TgBotServices) changeHistorySize(update *tgbotapi.Update) error {
	msg := update.Message.Text
	if msg == "" {
		return b.sendMessage(b.ChatID, "Нужно ввести целое число! Например: 50", update.Message.MessageID, nil)
	}

	newSize, err := strconv.Atoi(msg)
	if err != nil {
		logrus.WithError(err).Error("Ошибка преобразования: ")
		return b.sendMessage(b.ChatID, "Нужно ввести именно целое число от 1 до 200! Например: 50", update.Message.MessageID, nil)
	}
	if newSize < 1 || newSize > 200 {
		return b.sendMessage(b.ChatID, "Нужно ввести именно целое число от 1 до 200! Например: 50", update.Message.MessageID, nil)
	}

	b.dialogHistorySize = newSize
	msg = fmt.Sprintf("Теперь размер памяти истории диалога с ИИ = %d", b.dialogHistorySize)
	return b.sendMessage(b.ChatID, msg, update.Message.MessageID, nil)
}

// checkSizeDialogHistory reports whether the current dialog exceeds the configured limit.
func (b *TgBotServices) checkSizeDialogHistory(history []models.Message) bool {
	return len(history) > b.dialogHistorySize
}

// generativeTextWithStream streams the AI answer into a single Telegram message.
func (b *TgBotServices) generativeTextWithStream(update *tgbotapi.Update) error {
	msg := tgbotapi.NewMessage(b.ChatID, "Я обрабатываю ваш запрос...")
	lastMsg, err := b.Bot.Send(msg)
	if err != nil {
		logrus.WithError(err).Error("Ошибка отправки сообщения")
	}

	history, err := b.AIDialogRepo.GetDialogHistory(b.ChatID)
	if err != nil {
		logrus.WithError(err).Error("Failed to load dialog history")
		history = []models.Message{}
	}

	if b.checkSizeDialogHistory(history) {
		if err = b.AIDialogRepo.ClearHistory(b.ChatID); err != nil {
			logrus.WithError(err).Error("Failed to clear dialog history")
		}

		msg = tgbotapi.NewMessage(b.ChatID, "Размер истории переписки с ИИ превышен и был очищен. Создан новый чат")
		if _, err = b.Bot.Send(msg); err != nil {
			logrus.WithError(err).Error("Ошибка отправки сообщения")
		}
	}

	userMsg := models.Message{
		Role:    "user",
		Content: update.Message.Text,
	}
	if err = b.AIDialogRepo.SaveMsgToDialog(b.ChatID, userMsg); err != nil {
		logrus.WithError(err).Error("Failed to save user message to dialog")
	}

	responseChan := b.Generative.GenerateStreamTextMsg(update.Message.Text, history)

	var fullResponse strings.Builder
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case chunk, ok := <-responseChan:
			if !ok {
				if fullResponse.Len() > 0 {
					finalMsg := tgbotapi.NewEditMessageText(b.ChatID, lastMsg.MessageID, fullResponse.String())
					if _, err = b.Bot.Send(finalMsg); err != nil {
						logrus.WithError(err).Error("Failed to send final message")
					}
				}

				if len(lastMsg.Text) > 0 {
					aiResponse := models.Message{
						Role:    "assistant",
						Content: fullResponse.String(),
					}
					if err = b.AIDialogRepo.SaveMsgToDialog(b.ChatID, aiResponse); err != nil {
						logrus.WithError(err).Error("Failed to save AI response to dialog")
					}
				}

				return err
			}

			fullResponse.WriteString(chunk)

		case <-ticker.C:
			if fullResponse.Len() > 0 {
				editedMsg := tgbotapi.NewEditMessageText(b.ChatID, lastMsg.MessageID, fullResponse.String())
				if _, err = b.Bot.Send(editedMsg); err != nil {
					logrus.WithError(err).Error("Failed to edit message")
				}
			}
		}
	}
}

// changeGenerativeModel switches the current model to the user-selected one.
func (b *TgBotServices) changeGenerativeModel(update *tgbotapi.Update) error {
	if err := b.Generative.ChangeGenerativeModelName(update.Message.Text); err != nil {
		logrus.WithError(err).Error("Change generative model failed")
		b.sendMessage(b.ChatID, "На данный момент сменить генеративную модель не удалось. "+
			"Попробуй проверить правильно ли ты указал название модели или есть ли к ней доступ у твоего аккаунта!", update.Message.MessageID, nil)
		return err
	}

	return b.sendMessage(b.ChatID, "Смена произошла успешно!", update.Message.MessageID, nil)
}
