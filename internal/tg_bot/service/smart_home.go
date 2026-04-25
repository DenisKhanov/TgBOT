package service

import (
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/constant"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"strconv"
)

// showSmartMenu displays a menu for Smart Home controls, restricted to the bot owner.
// Returns an error if the user is not the owner, authentication fails, or the message fails to send it.
func (b *TgBotServices) showSmartMenu() error {
	if b.ChatID != b.OwnerID {
		if err := b.showBarMenu(); err != nil {
			logrus.WithError(err).Error("Ошибка отображения основного меню:")
		}
		return b.sendMessage(b.ChatID, "Извини, но доступ к этому меню есть только у моего Хозяина.", 0, nil)
	}
	if _, err := b.StateRepo.GetUserSmartHomeToken(b.ChatID); err != nil {
		if err = b.getSmartHomeToken(b.ChatID); err != nil {
			return b.showOAuthButton()
		}
	}
	devices, err := b.StateRepo.GetUserSmartHomeDevices(b.ChatID)
	if err != nil {
		return b.sendMessage(b.ChatID, "Не удалось загрузить устройства", 0, nil)
	}

	rows := [][]tgbotapi.KeyboardButton{
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_PRINT_MENU),
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_YANDEX_GET_HOME_INFO),
		),
	}

	for name, device := range devices {
		state := "Включить"
		if device.ActualState {
			state = "Выключить"
		}
		buttonText := fmt.Sprintf("%s: %s", state, name)
		rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(buttonText)))
	}
	markup := tgbotapi.NewReplyKeyboard(rows...)
	markup.ResizeKeyboard = true
	markup.OneTimeKeyboard = true
	return b.sendMessage(b.ChatID, "Выберите пункт ↓", 0, markup)
}

// showOAuthButton prompts the user to authenticate with Yandex for Smart Home access.
// Returns an error if the message fails to send.
func (b *TgBotServices) showOAuthButton() error {
	strChatID := strconv.FormatInt(b.ChatID, 10)
	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(constant.BUTTON_TEXT_YANDEX_SEND_CODE, b.OAuthURL+strChatID),
		),
	)
	return b.sendMessage(b.ChatID, "Нужно пройти аутентификацию ↓", 0, markup)
}

// getSmartHomeToken retrieves and stores a Smart Home token for the specified chat.
func (b *TgBotServices) getSmartHomeToken(chatID int64) error {
	tokenData, err := b.Handler.GetUserToken(chatID)
	if err != nil {
		logrus.WithError(err).Error("Failed to get Yandex token")
		return err
	}

	userDevices, err := b.SmartHome.GetHomeInfo(tokenData.AccessToken)
	if err != nil {
		b.sendMessage(b.ChatID, "Произошла ошибка, не удалось получить информацию об устройствах", 0, nil)
		return fmt.Errorf("failed to get home info: %w", err)
	}

	b.StateRepo.SaveUserSmartHomeInfo(chatID, tokenData.AccessToken, userDevices)
	return b.sendMessage(chatID, "Авторизация прошла успешно", 0, nil)
}

// showSmartHomeInfo sends information about the user's Smart Home devices.
func (b *TgBotServices) showSmartHomeInfo() error {
	token, err := b.StateRepo.GetUserSmartHomeToken(b.ChatID)
	if err != nil {
		return b.sendMessage(b.ChatID, "Произошла ошибка, похоже вы не прошли авторизацию", 0, nil)
	}

	userHomeInfoData, err := b.SmartHome.GetHomeInfo(token)
	if err != nil {
		return b.sendMessage(b.ChatID, "Произошла ошибка, не удалось получить информацию от сервера", 0, nil)
	}

	var text string
	for name, device := range userHomeInfoData {
		state := "выключено"
		if device.ActualState {
			state = "включено"
		}
		text += fmt.Sprintf("%s: %s (ID: %s)\n", name, state, device.ID)
	}
	return b.sendMessage(b.ChatID, text, 0, nil)
}

// setDeviceTurnOnOffStatus toggles the state of a specified Smart Home device.
func (b *TgBotServices) setDeviceTurnOnOffStatus(deviceName string) error {
	token, err := b.StateRepo.GetUserSmartHomeToken(b.ChatID)
	if err != nil {
		return b.sendMessage(b.ChatID, "Произошла ошибка, похоже вы не прошли авторизацию", 0, nil)
	}

	devices, err := b.StateRepo.GetUserSmartHomeDevices(b.ChatID)
	if err != nil {
		return b.sendMessage(b.ChatID, "Произошла ошибка, устройства не найдены", 0, nil)
	}
	device, ok := devices[deviceName]
	if !ok {
		return b.sendMessage(b.ChatID, fmt.Sprintf("Устройство %s не найдено", deviceName), 0, nil)
	}

	if err = b.SmartHome.TurnOnOffAction(token, device.ID, device.ActualState); err != nil {
		return b.sendMessage(b.ChatID, "Не удалось подключиться к устройству", 0, nil)
	}

	device.ActualState = !device.ActualState
	text := "Выключил: " + deviceName
	if device.ActualState {
		text = "Включил: " + deviceName
	}
	return b.sendMessage(b.ChatID, text, 0, nil)
}
