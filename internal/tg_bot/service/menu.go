package service

import (
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/constant"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"time"
)

// sendIntroMessageWithDelay sends an introductory message after a specified delay.
func (b *TgBotServices) sendIntroMessageWithDelay(delayInSec uint8, text string) {
	time.Sleep(time.Duration(delayInSec) * time.Second)
	if err := b.sendMessage(b.ChatID, text, 0, nil); err != nil {
		logrus.WithError(err).Error("Error sending intro message")
	}
}

// getKeyboardRow creates a single-row inline keyboard with one button.
func (b *TgBotServices) getKeyboardRow(buttonText, buttonCode string) []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(buttonText, buttonCode))
}

// printIntro sends a sequence of introductory messages with delays to the current chat.
func (b *TgBotServices) printIntro() {
	b.sendIntroMessageWithDelay(1, "Привет, пока что я небольшой bot-проект")
	b.sendIntroMessageWithDelay(2, "Но мои возможности регулярно растут")
	b.sendIntroMessageWithDelay(1, constant.EMOJI_BICEPS)
}

// askToPrintIntro prompts the user to choose whether to view the introductory messages.
func (b *TgBotServices) askToPrintIntro() error {
	markup := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_PRINT_INTRO),
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_SKIP_INTRO),
		),
	)
	return b.sendMessage(b.ChatID, "Это приветственное вступление, в нем описываются возможности бота, но ты можешь пропустить его. Что ты выберешь?", 0, markup)
}

// sendSorryMsg sends an apologetic message in response to an unsupported action.
func (b *TgBotServices) sendSorryMsg(update *tgbotapi.Update) error {
	return b.sendMessage(b.ChatID, "Я пока этого не умею, но я учусь", update.Message.MessageID, nil)
}

// showBarMenu displays the main keyboard menu.
func (b *TgBotServices) showBarMenu() error {
	markup := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_WHAT_TO_DO),
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_WHITCH_MOVIE_TO_WATCH),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_TRANSLATE),
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_YANDEX_DDIALOGS),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_GENERATIVE_MODEL),
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_GENERATIVE_MENU),
		),
	)
	markup.ResizeKeyboard = true
	markup.OneTimeKeyboard = true

	return b.sendMessage(b.ChatID, "Меню ↓", 0, markup)
}

// showHeadMenu displays the main inline menu with bot capabilities.
func (b *TgBotServices) showHeadMenu() error {
	markup := tgbotapi.NewInlineKeyboardMarkup(
		b.getKeyboardRow(constant.BUTTON_TEXT_WHAT_TO_DO, constant.BUTTON_CODE_WHAT_TO_DO),
		b.getKeyboardRow(constant.BUTTON_TEXT_WHITCH_MOVIE_TO_WATCH, constant.BUTTON_CODE_WHITCH_MOVIE_TO_WATCH),
		b.getKeyboardRow(constant.BUTTON_TEXT_TRANSLATE, constant.BUTTON_CODE_TRANSLATE),
		b.getKeyboardRow(constant.BUTTON_TEXT_YANDEX_DDIALOGS, constant.BUTTON_CODE_YANDEX_DDIALOGS),
	)
	return b.sendMessage(b.ChatID, "Выберите способность:", 0, markup)
}

func (b *TgBotServices) getDefaultInlineResults() []interface{} {
	var results []interface{}

	activity := b.Boring.WhatToDo()
	activityResult := tgbotapi.NewInlineQueryResultArticleMarkdown(
		ActivityResultID,
		"Предложи чем мне заняться",
		activity,
	)
	results = append(results, activityResult)

	movieResult := tgbotapi.NewInlineQueryResultArticleMarkdown(
		MovieResultID,
		"Посоветуй подборку фильмов",
		"Нажми, чтобы перейти к подборке фильмов",
	)
	movieResult.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("Перейти", b.MoviesURL),
			),
		},
	}
	results = append(results, movieResult)
	return results
}

// SendActivityMsg sends a random activity suggestion to the current chat.
func (b *TgBotServices) SendActivityMsg() error {
	text := b.Boring.WhatToDo()
	return b.sendMessage(b.ChatID, text, 0, nil)
}

// SendMoviesLink sends a message with a link to a movie recommendation site.
func (b *TgBotServices) SendMoviesLink() error {
	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(constant.BUTTON_TEXT_WHITCH_MOVIE_TO_WATCH, b.MoviesURL),
		),
	)
	return b.sendMessage(b.ChatID, "Тут представлена подборка отличных фильмов по мнению Дениса!", 0, markup)
}

// showGenerativeMenu displays a menu for Generative models controls, restricted to the bot owner.
func (b *TgBotServices) showGenerativeMenu() error {
	if b.ChatID != b.OwnerID {
		if err := b.showBarMenu(); err != nil {
			logrus.WithError(err).Error("Ошибка отображения основного меню:")
		}
		return b.sendMessage(b.ChatID, "Извини, но доступ к этому меню есть только у моего Хозяина.", 0, nil)
	}

	rows := [][]tgbotapi.KeyboardButton{
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_CHANGE_MODEL),
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_PRINT_MENU),
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_CHANGE_HISTORY_SIZE),
		),
	}
	markup := tgbotapi.NewReplyKeyboard(rows...)
	markup.ResizeKeyboard = true
	markup.OneTimeKeyboard = true
	return b.sendMessage(b.ChatID, "Выберите пункт ↓", 0, markup)
}
