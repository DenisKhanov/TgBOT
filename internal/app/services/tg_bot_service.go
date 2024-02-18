package services

import (
	"GoProgects/PetProjects/internal/app/constant"
	"GoProgects/PetProjects/internal/app/repository"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"log"
	"time"
)

type Repository interface {
	ReadFileToMemoryURL() error
	SaveBatchToFile() error
	StoreUserState(chatID int64, currentStep, lastUserMassage, callbackQueryData string, isTranslating bool)
	SaveUserYandexSmartHomeToken(chatID int64, token string)
	GetUserYandexSmartHomeToken(chatID int64) (string, error)
}

type TgBotServices struct {
	Boring          Boring
	YandexTranslate YandexTranslate
	YandexAuth      YandexAuth
	YandexSmartHome YandexSmartHome
	Repository      Repository
	ChatID          int64
	Bot             *tgbotapi.BotAPI
}

func NewTgBot(boring Boring, yandex YandexTranslate, yandexAuth YandexAuth, yandexSmartHome YandexSmartHome, repository Repository, bot *tgbotapi.BotAPI) *TgBotServices {
	return &TgBotServices{
		Boring:          boring,
		YandexTranslate: yandex,
		YandexAuth:      yandexAuth,
		YandexSmartHome: yandexSmartHome,
		Repository:      repository,
		Bot:             bot,
	}
}

func (b *TgBotServices) sendIntroMessageWithDelay(delayInSec uint8, text string) {
	msg := tgbotapi.NewMessage(b.ChatID, text)
	time.Sleep(time.Duration(delayInSec) * time.Second)
	b.Bot.Send(msg)
}
func (b *TgBotServices) getKeyboardRow(buttonText, buttonCode string) []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(buttonText, buttonCode))
}

func (b *TgBotServices) printIntro() {
	b.sendIntroMessageWithDelay(1, "Привет, пока что я небольшой bot-проект")
	b.sendIntroMessageWithDelay(2, "Но мои возможности регулярно растут")
	b.sendIntroMessageWithDelay(1, constant.EMOJI_BICEPS)
}
func (b *TgBotServices) askToPrintIntro() {
	msg := tgbotapi.NewMessage(b.ChatID, "Это приветственное вступление, в нем описываются возможности бота, ты можешь пропустить его. Что ты выберешь?")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		b.getKeyboardRow(constant.BUTTON_TEXT_PRINT_INTRO, constant.BUTTON_CODE_PRINT_INTRO),
		b.getKeyboardRow(constant.BUTTON_TEXT_SKIP_INTRO, constant.BUTTON_CODE_SKIP_INTRO),
	)
	b.Bot.Send(msg)
}

func (b *TgBotServices) sendSorryMsg(update *tgbotapi.Update) {
	msg := tgbotapi.NewMessage(b.ChatID, "Я пока этого не умею, но я учусь")
	msg.ReplyToMessageID = update.Message.MessageID
	b.Bot.Send(msg)
}

func (b *TgBotServices) showHeadMenu() {
	msg := tgbotapi.NewMessage(b.ChatID, "Выберите способность:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(constant.BUTTON_TEXT_WHAT_TO_DO, constant.BUTTON_CODE_WHAT_TO_DO)),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(constant.BUTTON_TEXT_TRANSLATE, constant.BUTTON_CODE_TRANSLATE)),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(constant.BUTTON_TEXT_YANDEX_DDIALOGS, constant.BUTTON_CODE_YANDEX_DDIALOGS)),
	)
	b.Bot.Send(msg)
}

var debounceTimer *time.Timer

// HandleInlineQuery запросы в режиме inline которые переводят текст и генерируют "Чем заняться".
// Пока что в режиме реального времени, скидывая счетчик задержки когда пользователь печатает и переводя, когда он остановился на 1,5 сек
func (b *TgBotServices) HandleInlineQuery(bot *tgbotapi.BotAPI, query *tgbotapi.InlineQuery) {
	currentInput := query.Query

	if debounceTimer != nil {
		debounceTimer.Stop()
	}

	debounceTimer = time.AfterFunc(1500*time.Millisecond, func() {
		var text string
		var err error
		var name string

		if currentInput != "" {
			text, err = b.YandexTranslate.TranslateAPI(currentInput)
			name = "Перевести введенный текст"
		} else if currentInput == "" {
			text, err = b.Boring.BoredAPI()
			text, err = b.YandexTranslate.TranslateAPI(text)
			name = "Предложи чем мне заняться"
		} else {
			// Если нет ввода, прерываем выполнение функции
			return
		}

		if err != nil {
			logrus.Error(err)
			return
		}

		whatToDo := []interface{}{
			tgbotapi.NewInlineQueryResultArticleMarkdown("1", name, text),
		}
		inlineConf := tgbotapi.InlineConfig{
			InlineQueryID: query.ID,
			Results:       whatToDo,
			CacheTime:     0,
		}
		if _, err := bot.Send(inlineConf); err != nil {
			log.Println("Ошибка при отправке ответа на inline-запрос:", err)
		}
	})
}

func (b *TgBotServices) UpdateProcessing(update *tgbotapi.Update, usersState *repository.UsersState) {
	var choiceCode string
	if update.CallbackQuery != nil && update.CallbackQuery.Data != "" {
		b.ChatID = update.CallbackQuery.Message.Chat.ID
		choiceCode = update.CallbackQuery.Data

		logrus.Infof("[%T] %s", time.Now(), choiceCode)
		switch choiceCode {
		case constant.BUTTON_CODE_PRINT_INTRO:
			b.printIntro()
			b.showHeadMenu()
		case constant.BUTTON_CODE_SKIP_INTRO:
			b.showHeadMenu()
		case constant.BUTTON_CODE_PRINT_MENU:
			b.showHeadMenu()
		case constant.BUTTON_CODE_WHAT_TO_DO:
			b.GenerateActivityMsg()
			b.showHeadMenu()
		case constant.BUTTON_CODE_YANDEX_DDIALOGS:
			b.showYandexMenu()
		case constant.BUTTON_CODE_YANDEX_GET_HOME_INFO:
			b.GetHomeInfo()
			b.showYandexSmartMenu()
		case constant.BUTTON_CODE_YANDEX_TURN_ON_NIGHT_LIGHT:
			b.YandexDeviceTurnOnOff(deviceIDNightLight, &nightLightCondition)
			b.showYandexSmartMenu()
		case constant.BUTTON_CODE_YANDEX_TURN_ON_SPEAKER:
			b.YandexDeviceTurnOnOff(deviceIDSpeaker, &speakerCondition)
			b.showYandexSmartMenu()
		case constant.BUTTON_CODE_TRANSLATE:
			b.Repository.StoreUserState(b.ChatID, "перевод", "", choiceCode, true) // Устанавливаем состояние перевода в true
			msg := tgbotapi.NewMessage(b.ChatID, "Вы в режиме перевода. \nВведите текст который хотите чтобы я перевел или отправьте /stop, чтобы выйти из режима перевода.")
			b.Bot.Send(msg)
		}
	} else if update.Message != nil && update.Message.Text != "" {
		fmt.Println("Hello")
		b.ChatID = update.Message.Chat.ID
		value, ok := usersState.BatchBuffer[b.ChatID]
		if update.Message.Text == "/stop" {
			b.Repository.StoreUserState(b.ChatID, "стоп", update.Message.Text, "", false)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Режим перевода выключен.")
			b.Bot.Send(msg)
			b.showHeadMenu()
		} else if ok && value.IsTranslating {
			b.translateText(update)
		} else if update.Message.Text == "/start" {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			b.Repository.StoreUserState(b.ChatID, "старт", update.Message.Text, "", false)
			b.ChatID = update.Message.Chat.ID
			b.askToPrintIntro()
		} else {
			b.sendSorryMsg(update)
		}
	}
}
