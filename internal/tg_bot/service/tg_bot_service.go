package service

import (
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/constant"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/repository"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"log"
	"strconv"
	"time"
)

type Boring interface {
	BoredAPI() string
}

type YandexTranslate interface {
	TranslateAPI(text string) (string, error)
	DetectLangAPI(text string) (string, error)
}

type YandexSmartHome interface {
	GetHomeInfo(token string) (map[string]*models.Device, error)
	TurnOnOffAction(token, id string, value bool) error
}

type YandexAuth interface {
	AuthAPI(accessCode string) (string, error)
}

type Repository interface {
	ReadFileToMemoryURL() error
	SaveBatchToFile() error
	StoreUserState(chatID int64, currentStep, lastUserMassage, callbackQueryData string, isTranslating bool)
	SaveUserYandexSmartHomeInfo(chatID int64, token string, devices map[string]*models.Device)
	GetUserYandexSmartHomeToken(chatID int64) (string, error)
	GetUserYandexSmartHomeDevices(chatID int64) (map[string]*models.Device, error)
}

type Handler interface {
	GetUserToken(chatID int64) (models.ResponseOAuth, error)
}

type TgBotServices struct {
	Boring          Boring
	YandexTranslate YandexTranslate
	YandexSmartHome YandexSmartHome
	Repository      Repository
	ChatID          int64
	Bot             *tgbotapi.BotAPI
	Handler         Handler
}

func NewTgBot(boring Boring, yandex YandexTranslate, yandexSmartHome YandexSmartHome, repository Repository, bot *tgbotapi.BotAPI, handler Handler) *TgBotServices {
	return &TgBotServices{
		Boring:          boring,
		YandexTranslate: yandex,
		YandexSmartHome: yandexSmartHome,
		Repository:      repository,
		Bot:             bot,
		Handler:         handler,
	}
}

func (b *TgBotServices) sendIntroMessageWithDelay(delayInSec uint8, text string) {
	msg := tgbotapi.NewMessage(b.ChatID, text)
	time.Sleep(time.Duration(delayInSec) * time.Second)
	if _, err := b.Bot.Send(msg); err != nil {
		logrus.WithError(err).Error("error send msg: ")
	}
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
	if _, err := b.Bot.Send(msg); err != nil {
		logrus.WithError(err).Error("error send msg: ")
	}
}

func (b *TgBotServices) sendSorryMsg(update *tgbotapi.Update) {
	msg := tgbotapi.NewMessage(b.ChatID, "Я пока этого не умею, но я учусь")
	msg.ReplyToMessageID = update.Message.MessageID
	if _, err := b.Bot.Send(msg); err != nil {
		logrus.WithError(err).Error("error send msg: ")
	}
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
	if _, err := b.Bot.Send(msg); err != nil {
		logrus.WithError(err).Error("error send msg: ")
	}
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
			text = b.Boring.BoredAPI()
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

func (b *TgBotServices) SendActivityMsg() {
	text := b.Boring.BoredAPI()

	msg := tgbotapi.NewMessage(b.ChatID, text)

	if _, err := b.Bot.Send(msg); err != nil {
		logrus.WithError(err).Error("error send msg: ")
	}
}

func (b *TgBotServices) showYandexSmartMenu() {
	if _, err := b.Repository.GetUserYandexSmartHomeToken(b.ChatID); err != nil {
		if err = b.GetYandexSmartHomeToken(b.ChatID); err != nil {
			b.showYandexOAuthButton()
			return
		}
	}
	msg := tgbotapi.NewMessage(b.ChatID, "Выберите пункт:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(constant.BUTTON_TEXT_PRINT_MENU, constant.BUTTON_CODE_PRINT_MENU)),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(constant.BUTTON_TEXT_YANDEX_GET_HOME_INFO, constant.BUTTON_CODE_YANDEX_GET_HOME_INFO)),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(constant.BUTTON_TEXT_YANDEX_TURN_ON_NIGHT_LIGHT, constant.BUTTON_CODE_YANDEX_TURN_ON_NIGHT_LIGHT)),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(constant.BUTTON_TEXT_YANDEX_TURN_ON_SPEAKER, constant.BUTTON_CODE_YANDEX_TURN_ON_SPEAKER)),
	)
	if _, err := b.Bot.Send(msg); err != nil {
		logrus.WithError(err).Error("error send msg: ")
	}
}

func (b *TgBotServices) showYandexOAuthButton() {
	strChatID := strconv.Itoa(int(b.ChatID))
	msg := tgbotapi.NewMessage(b.ChatID, "Нужно пройти аутентификацию:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(constant.BUTTON_TEXT_PRINT_MENU, constant.BUTTON_CODE_PRINT_MENU)),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(constant.BUTTON_TEXT_YANDEX_SEND_CODE, constant.BUTTON_CODE_YANDEX_SEND_CODE+strChatID)),
	)
	if _, err := b.Bot.Send(msg); err != nil {
		logrus.WithError(err).Error("error send msg: ")
	}
}

func (b *TgBotServices) GetYandexSmartHomeToken(chatID int64) error {
	tokenData, err := b.Handler.GetUserToken(chatID)
	if err != nil {
		logrus.Error(err)
		return err
	}
	//TODO
	fmt.Println("TOKEN DATA")
	fmt.Println(tokenData)
	userDevices, err := b.YandexSmartHome.GetHomeInfo(tokenData.AccessToken)
	if err != nil {
		msg := tgbotapi.NewMessage(b.ChatID, "Произошла ошибка, не удалось получить от сервера информацию")
		logrus.Error(err)
		if _, err = b.Bot.Send(msg); err != nil {
			logrus.WithError(err).Error("error send msg: ")
			return err
		}
	}
	b.Repository.SaveUserYandexSmartHomeInfo(b.ChatID, tokenData.AccessToken, userDevices)
	msg := tgbotapi.NewMessage(b.ChatID, "Авторизация прошла успешно")
	if _, err = b.Bot.Send(msg); err != nil {
		logrus.WithError(err).Error("error send msg: ")
		return err
	}
	return nil
}

func (b *TgBotServices) SendUserHomeInfo() {
	//TODO реализовать вывод информации об умном доме пользователя
	//token, err := b.Repository.GetUserYandexSmartHomeToken(b.ChatID)
	//if err != nil {
	//	msg := tgbotapi.NewMessage(b.ChatID, "Произошла ошибка, похоже вы  не прошли авторизацию")
	//	logrus.Error(err)
	//	b.Bot.Send(msg)
	//	return
	//}
	//userHomeInfoData, err := b.YandexSmartHome.GetHomeInfo(token)
	//if err != nil {
	//	msg := tgbotapi.NewMessage(b.ChatID, "Произошла ошибка, не удалось получить от сервера информацию")
	//	logrus.Error(err)
	//	b.Bot.Send(msg)
	//	return
	//}
	//msg := tgbotapi.NewMessage(b.ChatID, userHomeInfoData)
	//b.Bot.Send(msg)
}

// TODO ID устройства должно быть получено автоматически из информации об устройствах пользователя

func (b *TgBotServices) YandexDeviceTurnOnOff(deviceName string) {
	token, err := b.Repository.GetUserYandexSmartHomeToken(b.ChatID)
	fmt.Println("TOKEN = " + token)
	if err != nil {
		msg := tgbotapi.NewMessage(b.ChatID, "Произошла ошибка, похоже вы  не прошли авторизацию")
		logrus.Error(err)
		if _, err = b.Bot.Send(msg); err != nil {
			logrus.WithError(err).Error("error send msg: ")
		}
		return
	}
	devices, err := b.Repository.GetUserYandexSmartHomeDevices(b.ChatID)
	if err != nil {
		msg := tgbotapi.NewMessage(b.ChatID, "Произошла ошибка, устройства не найдены")
		logrus.Error(err)
		if _, err = b.Bot.Send(msg); err != nil {
			logrus.WithError(err).Error("error send msg: ")
		}
		return
	}
	deviceID := devices[deviceName].ID
	deviceState := devices[deviceName].State

	if err := b.YandexSmartHome.TurnOnOffAction(token, deviceID, deviceState); err != nil {
		msg := tgbotapi.NewMessage(b.ChatID, "Не удалось подключиться к устройству")
		logrus.Error(err)
		if _, err = b.Bot.Send(msg); err != nil {
			logrus.WithError(err).Error("error send msg: ")
		}
		return
	}
	if !deviceState {
		devices[deviceName].State = true
	} else {
		devices[deviceName].State = false
	}
	msg := tgbotapi.NewMessage(b.ChatID, "Выполнено")
	if _, err = b.Bot.Send(msg); err != nil {
		logrus.WithError(err).Error("error send msg: ")
	}
}

func (b *TgBotServices) translateText(update *tgbotapi.Update) {
	translatedText, err := b.YandexTranslate.TranslateAPI(update.Message.Text)
	if err != nil {
		logrus.Error(err)
		return
	}
	msg := tgbotapi.NewMessage(b.ChatID, translatedText)
	msg.ReplyToMessageID = update.Message.MessageID
	if _, err = b.Bot.Send(msg); err != nil {
		logrus.WithError(err).Error("error send msg: ")
	}
}

func (b *TgBotServices) UpdateProcessing(update *tgbotapi.Update, usersState *repository.UsersState) {
	var choiceCode string
	if update.CallbackQuery != nil && update.CallbackQuery.Data != "" {
		b.ChatID = update.CallbackQuery.Message.Chat.ID
		choiceCode = update.CallbackQuery.Data
		b.Repository.StoreUserState(b.ChatID, "button", "", choiceCode, false)

		logrus.Infof("[%T] %s", time.Now(), choiceCode)
		switch choiceCode {
		case constant.BUTTON_CODE_PRINT_INTRO:
			b.printIntro()
			b.showHeadMenu()
		case constant.BUTTON_CODE_SKIP_INTRO:
			b.showHeadMenu()
			//TODO проверить почему при нажатии вывести яндекс меню снова приветствие напечаталось
		case constant.BUTTON_CODE_PRINT_MENU:
			b.showHeadMenu()
			//TODO исправить проблему выдачи одинаковых ответов в инлайн режиме
		case constant.BUTTON_CODE_WHAT_TO_DO:
			b.SendActivityMsg()
			b.showHeadMenu()
		case constant.BUTTON_CODE_YANDEX_DDIALOGS:
			b.showYandexSmartMenu()
		case constant.BUTTON_CODE_YANDEX_LOGIN:
			b.showYandexOAuthButton()
		case constant.BUTTON_CODE_YANDEX_GET_HOME_INFO:
			b.SendUserHomeInfo()
			b.showYandexSmartMenu()
			//TODO вывод кнопок с устройствами должен происходить динамически, в зависимости от их наличия
		case constant.BUTTON_CODE_YANDEX_TURN_ON_NIGHT_LIGHT:
			b.YandexDeviceTurnOnOff("Ночник")
			b.showYandexSmartMenu()
		case constant.BUTTON_CODE_YANDEX_TURN_ON_SPEAKER:
			b.YandexDeviceTurnOnOff("Колонки")
			b.showYandexSmartMenu()
		case constant.BUTTON_CODE_TRANSLATE:
			b.Repository.StoreUserState(b.ChatID, "перевод", "", choiceCode, true) // Устанавливаем состояние перевода в true
			msg := tgbotapi.NewMessage(b.ChatID, "Вы в режиме перевода. \nВведите текст который хотите чтобы я перевел или отправьте /stop, чтобы выйти из режима перевода.")
			if _, err := b.Bot.Send(msg); err != nil {
				logrus.WithError(err).Error("error send msg: ")
			}
		}
	} else if update.Message != nil && update.Message.Text != "" {
		b.ChatID = update.Message.Chat.ID
		value, ok := usersState.BatchBuffer[b.ChatID]
		if update.Message.Text == "/stop" {
			b.Repository.StoreUserState(b.ChatID, "стоп", update.Message.Text, "", false)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Режим перевода выключен.")
			if _, err := b.Bot.Send(msg); err != nil {
				logrus.WithError(err).Error("error send msg: ")
			}
			b.showHeadMenu()
		} else if ok && value.IsTranslating {
			b.translateText(update)
		} else if update.Message.Text == "/start" {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			b.Repository.StoreUserState(b.ChatID, "старт", update.Message.Text, "", false)
			b.ChatID = update.Message.Chat.ID
			b.askToPrintIntro()
		} else {
			b.Repository.StoreUserState(b.ChatID, "i can't do it now ", update.Message.Text, "", false)
			b.sendSorryMsg(update)
		}
	}
}
