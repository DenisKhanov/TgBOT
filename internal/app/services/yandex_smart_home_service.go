package services

import (
	"GoProgects/PetProjects/internal/app/constant"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type YandexAuth interface {
	AuthAPI(accessCode string) (string, error)
}
type YandexSmartHome interface {
	GetHomeInfo(token string) ([]byte, error)
	TurnOnOffAction(token, id string, value bool) error
}

var deviceIDNightLight = "a2e6c788-cbae-4846-af92-f7bfcae79fd5"
var deviceIDSpeaker = "41128f2a-e954-4af7-a81a-2f807ea8baa6"
var nightLightCondition = true
var speakerCondition = true

func (b *TgBotServices) showYandexMenu() {
	msg := tgbotapi.NewMessage(b.ChatID, "Выберите пункт:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(constant.BUTTON_TEXT_PRINT_MENU, constant.BUTTON_CODE_PRINT_MENU)),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(constant.BUTTON_TEXT_YANDEX_LOGIN, constant.BUTTON_CODE_YANDEX_LOGIN)),
	)
	b.Bot.Send(msg)
}

func (b *TgBotServices) showYandexSmartMenu() {
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
	b.Bot.Send(msg)
}

func (b *TgBotServices) GetYandexSmartHomeToken(accessCode string) {
	token, err := b.YandexAuth.AuthAPI(accessCode)
	if err != nil {
		logrus.Error(err)
	}
	b.Repository.SaveUserYandexSmartHomeToken(b.ChatID, token)
	msg := tgbotapi.NewMessage(b.ChatID, "Авторизация прошла успешно")
	b.Bot.Send(msg)
	b.showYandexSmartMenu()
}
func (b *TgBotServices) GetHomeInfo() {
	token, err := b.Repository.GetUserYandexSmartHomeToken(b.ChatID)
	if err != nil {
		msg := tgbotapi.NewMessage(b.ChatID, "Произошла ошибка, похоже вы  не прошли авторизацию")
		logrus.Error(err)
		b.Bot.Send(msg)
		return
	}
	homeInfoData, err := b.YandexSmartHome.GetHomeInfo(token)
	if err != nil {
		msg := tgbotapi.NewMessage(b.ChatID, "Произошла ошибка, не удалось получить от сервера информацию")
		logrus.Error(err)
		b.Bot.Send(msg)
		return
	}
	msg := tgbotapi.NewMessage(b.ChatID, string(homeInfoData))
	b.Bot.Send(msg)
}

// TODO ID устройства должно быть получено автоматически из информации об устройствах пользователя
func (b *TgBotServices) YandexDeviceTurnOnOff(deviceID string, deviceCondition *bool) {
	token, err := b.Repository.GetUserYandexSmartHomeToken(b.ChatID)
	if err != nil {
		msg := tgbotapi.NewMessage(b.ChatID, "Произошла ошибка, похоже вы  не прошли авторизацию")
		logrus.Error(err)
		b.Bot.Send(msg)
		return
	}
	if err := b.YandexSmartHome.TurnOnOffAction(token, deviceID, *deviceCondition); err != nil {
		msg := tgbotapi.NewMessage(b.ChatID, "Не удалось подключиться к устройству")
		logrus.Error(err)
		b.Bot.Send(msg)
		return
	}
	if !*deviceCondition {
		*deviceCondition = true
	} else {
		*deviceCondition = false
	}
	msg := tgbotapi.NewMessage(b.ChatID, "Выполнено")
	b.Bot.Send(msg)
}
