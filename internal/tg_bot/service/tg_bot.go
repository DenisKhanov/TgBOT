// Package service provides the core logic for a Telegram bot, integrating various services.
// It handles user interactions, inline queries, and Yandex Smart Home operations.
package service

import (
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/constant"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Boring defines the interface for suggesting activities.
type Boring interface {
	WhatToDo() string
}

// Translate defines the interface for translation operations.
type Translate interface {
	TranslateAPI(text string) (string, error)  // Retrieves smart home device info.
	DetectLangAPI(text string) (string, error) // Toggles a device on or off.
}

// SmartHome defines the interface for Yandex Smart Home operations.
type SmartHome interface {
	GetHomeInfo(token string) (map[string]*models.Device, error)
	TurnOnOffAction(token, id string, value bool) error
}

type GenerativeModel interface {
	GenerateTextMsg(text string) (string, error)
	ChangeGenerativeModelName(modelName string) error
}

// The Repository defines the interface for user state persistence.
type Repository interface {
	ReadFileToMemoryURL() error
	SaveBatchToFile() error
	StoreUserState(chatID int64, currentStep, lastUserMassage, callbackQueryData string, isTranslating, isGenerative, IsChangingGenModel bool)
	SaveUserSmartHomeInfo(chatID int64, token string, devices map[string]*models.Device)
	GetUserSmartHomeToken(chatID int64) (string, error)
	GetUserSmartHomeDevices(chatID int64) (map[string]*models.Device, error)
	GetTranslateState(chatID int64) bool
	GetGenerativeState(chatID int64) bool
	GetChangeModelState(chatID int64) bool
}

// Handler defines the interface for OAuth token handling.
type Handler interface {
	GetUserToken(chatID int64) (models.ResponseOAuth, error)
}

// TgBotServices is the main service struct for the Telegram bot, integrating all dependencies.
type TgBotServices struct {
	Boring         Boring    // Activity suggestion service.
	Translate      Translate // Translation service.
	SmartHome      SmartHome // Smart home service.
	Generative     GenerativeModel
	Repository     Repository            // User state repository.
	ChatID         int64                 // Current chat ID.
	Bot            *tgbotapi.BotAPI      // Telegram Bot API instance.
	Handler        Handler               // OAuth handler.
	OAuthURL       string                // URL for OAuth authentication.
	OwnerID        int64                 // Owner's chatID for access to Yandex smart home menu button
	debounceTimers map[int64]*time.Timer // Per-chat debounce timers
	lastQueries    map[int64]string
	mu             *sync.Mutex // Protects debounceTimers
}

// NewTgBot creates a new TgBotServices instance with the specified dependencies.
// Arguments:
//   - boring: activity suggestion service.
//   - yandex: translation service.
//   - yandexSmartHome: smart home service.
//   - repository: user state repository.
//   - bot: Telegram Bot API instance.
//   - handler: OAuth handler.
//   - URL: OAuth URL.
//
// Returns a pointer to a TgBotServices.
func NewTgBot(boring Boring, translate Translate, smartHome SmartHome, generative GenerativeModel, repository Repository, bot *tgbotapi.BotAPI, handler Handler, URL string, ownerID int64) *TgBotServices {
	return &TgBotServices{
		Boring:         boring,
		Translate:      translate,
		SmartHome:      smartHome,
		Generative:     generative,
		Repository:     repository,
		Bot:            bot,
		Handler:        handler,
		OAuthURL:       URL,
		OwnerID:        ownerID,
		debounceTimers: make(map[int64]*time.Timer),
		lastQueries:    make(map[int64]string),
		mu:             &sync.Mutex{},
	}
}

// sendMessage sends a message to the specified chat with optional reply and markup.
// Arguments:
//   - chatID: the ID of the chat to send the message to.
//   - text: the text content of the message.
//   - replyToID: the ID of the message to reply to (0 if no reply).
//   - markup: an optional keyboard or inline markup (nil if none).
//
// Returns an error if the message fails to send.
func (b *TgBotServices) sendMessage(chatID int64, text string, replyToID int, markup interface{}) error {
	msg := tgbotapi.NewMessage(chatID, text)
	if replyToID != 0 {
		msg.ReplyToMessageID = replyToID
	}
	if markup != nil {
		msg.ReplyMarkup = markup
	}
	_, err := b.Bot.Send(msg)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to send message to chat %d: %s", chatID, text)
	}
	return err
}

// sendIntroMessageWithDelay sends an introductory message after a specified delay.
// Arguments:
//   - delayInSec: the delay in seconds before sending the message.
//   - text: the text content of the message.
func (b *TgBotServices) sendIntroMessageWithDelay(delayInSec uint8, text string) {
	time.Sleep(time.Duration(delayInSec) * time.Second)
	if err := b.sendMessage(b.ChatID, text, 0, nil); err != nil {
		logrus.WithError(err).Error("Error sending intro message")
	}
}

// getKeyboardRow creates a single-row inline keyboard with one button.
// Arguments:
//   - buttonText: the text displayed on the button.
//   - buttonCode: the callback data associated with the button.
//
// Returns a slice of InlineKeyboardButton representing the row.
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
// Returns an error if the prompt message fails to send.
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
// Arguments:
//   - update: the Telegram update containing the message to reply to.
//
// Returns an error if the message fails to send.
func (b *TgBotServices) sendSorryMsg(update *tgbotapi.Update) error {
	return b.sendMessage(b.ChatID, "Я пока этого не умею, но я учусь", update.Message.MessageID, nil)
}

// showHeadMenu displays the main menu with bot capabilities as inline buttons.
// Returns an error if the menu message fails to send.
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
	// Дополнительные настройки клавиатуры
	markup.ResizeKeyboard = true  // Подгоняет размер клавиатуры под экран
	markup.OneTimeKeyboard = true // Скрывает клавиатуру после выбора (опционально)

	return b.sendMessage(b.ChatID, "Меню ↓", 0, markup)
}

// showHeadMenu displays the main menu with bot capabilities as inline buttons.
// Returns an error if the menu message fails to send.
func (b *TgBotServices) showHeadMenu() error {
	markup := tgbotapi.NewInlineKeyboardMarkup(
		b.getKeyboardRow(constant.BUTTON_TEXT_WHAT_TO_DO, constant.BUTTON_CODE_WHAT_TO_DO),
		b.getKeyboardRow(constant.BUTTON_TEXT_WHITCH_MOVIE_TO_WATCH, constant.BUTTON_CODE_WHITCH_MOVIE_TO_WATCH),
		b.getKeyboardRow(constant.BUTTON_TEXT_TRANSLATE, constant.BUTTON_CODE_TRANSLATE),
		b.getKeyboardRow(constant.BUTTON_TEXT_YANDEX_DDIALOGS, constant.BUTTON_CODE_YANDEX_DDIALOGS),
	)
	return b.sendMessage(b.ChatID, "Выберите способность:", 0, markup)
}

//TODO постоянно в inline режиме выдается одно сообщение с предложением чем заняться

// HandleInlineQuery processes inline queries for translation or activity suggestions with debouncing.
// Arguments:
//   - bot: Telegram Bot API instance.
//   - query: the inline query from the user.
func (b *TgBotServices) HandleInlineQuery(bot *tgbotapi.BotAPI, query *tgbotapi.InlineQuery) {
	chatID := query.From.ID
	currentInput := query.Query

	// Потокобезопасно обновляем последний запрос и таймер
	b.mu.Lock()
	b.lastQueries[chatID] = currentInput

	// Останавливаем предыдущий таймер, если он существует
	if timer, exists := b.debounceTimers[chatID]; exists {
		timer.Stop()
	}

	// Запускаем новый таймер с debounce на 1.5 секунды
	b.debounceTimers[chatID] = time.AfterFunc(1500*time.Millisecond, func() {
		// После задержки проверяем, актуален ли запрос
		b.mu.Lock()
		lastQuery := b.lastQueries[chatID]
		// Удаляем таймер после выполнения
		delete(b.debounceTimers, chatID)
		b.mu.Unlock()

		// Если текст запроса изменился за время ожидания, игнорируем
		if lastQuery != currentInput {
			logrus.WithField("chatID", chatID).Info("Запрос устарел, пропускаем")
			return
		}

		var results []interface{}

		// Если есть ввод, показываем перевод
		if currentInput != "" {
			textMsg, err := b.Generative.GenerateTextMsg(currentInput)
			if err != nil {
				logrus.WithError(err).Error("Inline query generative text dialog failed")
				return
			}
			result := tgbotapi.NewInlineQueryResultArticleMarkdown(
				"1",
				"Спросить у ИИ",
				textMsg,
			)
			results = append(results, result)
		} else {
			// Если ввода нет, показываем два варианта
			activity := b.Boring.WhatToDo()
			activityResult := tgbotapi.NewInlineQueryResultArticleMarkdown(
				"1",
				"Предложи чем мне заняться",
				activity,
			)
			results = append(results, activityResult)

			movieResult := tgbotapi.NewInlineQueryResultArticle(
				"2",
				"Посоветуй подборку фильмов",
				"Нажми, чтобы перейти к подборке фильмов",
			)
			movieResult.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonURL("Перейти", "http://176.108.251.250:8444/"),
					),
				},
			}
			results = append(results, movieResult)
		}

		// Конфигурация ответа inline-запроса
		inlineConf := tgbotapi.InlineConfig{
			InlineQueryID: query.ID,
			Results:       results,
			CacheTime:     0,
			IsPersonal:    true,
		}

		if _, err := bot.Request(inlineConf); err != nil {
			logrus.WithError(err).Error("Failed to send inline query response")
		}
	})
	b.mu.Unlock()
}

// SendActivityMsg sends a random activity suggestion to the current chat.
// Returns an error if the message fails to send.
func (b *TgBotServices) SendActivityMsg() error {
	text := b.Boring.WhatToDo()
	return b.sendMessage(b.ChatID, text, 0, nil)
}

// SendMoviesLink sends a message with a link to a movie recommendation site.
// Returns an error if the message fails to send.
func (b *TgBotServices) SendMoviesLink() error {
	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(constant.BUTTON_TEXT_WHITCH_MOVIE_TO_WATCH, "http://176.108.251.250:8444/"),
		),
	)
	return b.sendMessage(b.ChatID, "Тут представлена подборка отличных фильмов по мнению Дениса!", 0, markup)
}

// showSmartMenu displays a menu for Smart Home controls, restricted to the bot owner.
// Returns an error if the user is not the owner, authentication fails, or the message fails to send it.
func (b *TgBotServices) showSmartMenu() error {
	if b.ChatID != b.OwnerID {
		return b.sendMessage(b.ChatID, "Извини, но доступ к этому меню есть только у моего Хозяина.", 0, nil)
	}
	if _, err := b.Repository.GetUserSmartHomeToken(b.ChatID); err != nil {
		if err = b.getSmartHomeToken(b.ChatID); err != nil {
			return b.showOAuthButton()
		}
	}
	devices, err := b.Repository.GetUserSmartHomeDevices(b.ChatID)
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
	// Дополнительные настройки клавиатуры
	markup.ResizeKeyboard = true  // Подгоняет размер клавиатуры под экран
	markup.OneTimeKeyboard = true // Скрывает клавиатуру после выбора (опционально)
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
// Arguments:
//   - chatID: the ID of the chat to authenticate.
//
// Returns an error if token retrieval or device info fetching fails.
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

	b.Repository.SaveUserSmartHomeInfo(b.ChatID, tokenData.AccessToken, userDevices)
	return b.sendMessage(b.ChatID, "Авторизация прошла успешно", 0, nil)
}

// showSmartHomeInfo sends information about the user's Smart Home devices.
// Returns an error if token retrieval, device info fetching, or message sending fails.
func (b *TgBotServices) showSmartHomeInfo() error {
	token, err := b.Repository.GetUserSmartHomeToken(b.ChatID)
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
// Arguments:
//   - deviceName: the name of the device to toggle.
//
// Returns an error if token retrieval, device lookup, state change, or message sending fails.
func (b *TgBotServices) setDeviceTurnOnOffStatus(deviceName string) error {
	token, err := b.Repository.GetUserSmartHomeToken(b.ChatID)
	if err != nil {
		return b.sendMessage(b.ChatID, "Произошла ошибка, похоже вы не прошли авторизацию", 0, nil)
	}

	devices, err := b.Repository.GetUserSmartHomeDevices(b.ChatID)
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
	text := "Включил: " + deviceName
	if device.ActualState {
		text = "Выключил: " + deviceName
	}
	return b.sendMessage(b.ChatID, text, 0, nil)
}

// translateText translates the text from the provided update and sends it as a reply.
// Arguments:
//   - update: the Telegram update containing the text to translate.
//
// Returns an error if translation or message sending fails.
func (b *TgBotServices) translateText(update *tgbotapi.Update) error {
	translatedText, err := b.Translate.TranslateAPI(update.Message.Text)
	if err != nil {
		logrus.WithError(err).Error("Translation failed")
		return err
	}
	return b.sendMessage(b.ChatID, translatedText, update.Message.MessageID, nil)
}

// showGenerativeMenu displays a menu for Generative models controls, restricted to the bot owner.
// Returns an error if the user is not the owner, authentication fails, or the message fails to send it.
func (b *TgBotServices) showGenerativeMenu() error {
	if b.ChatID != b.OwnerID {
		return b.sendMessage(b.ChatID, "Извини, но доступ к этому меню есть только у моего Хозяина.", 0, nil)
	}

	rows := [][]tgbotapi.KeyboardButton{
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_CHANGE_MODEL),
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_PRINT_MENU),
		),
	}
	markup := tgbotapi.NewReplyKeyboard(rows...)
	// Дополнительные настройки клавиатуры
	markup.ResizeKeyboard = true  // Подгоняет размер клавиатуры под экран
	markup.OneTimeKeyboard = true // Скрывает клавиатуру после выбора (опционально)
	return b.sendMessage(b.ChatID, "Выберите пункт ↓", 0, markup)
}

// translateText translates the text from the provided update and sends it as a reply.
// Arguments:
//   - update: the Telegram update containing the text to translate.
//
// Returns an error if translation or message sending fails.
func (b *TgBotServices) generativeText(update *tgbotapi.Update) error {
	b.sendMessage(b.ChatID, "Я обрабатываю ваш запрос...", update.Message.MessageID, nil)
	aiResponse, err := b.Generative.GenerateTextMsg(update.Message.Text)
	if err != nil {
		logrus.WithError(err).Error("Request to generative model failed")
		b.sendMessage(b.ChatID, "На данный момент ИИ не доступен((", update.Message.MessageID, nil)
		return err
	}
	return b.sendMessage(b.ChatID, aiResponse, update.Message.MessageID, nil)
}

// changeGenerativeModel
func (b *TgBotServices) changeGenerativeModel(update *tgbotapi.Update) error {
	if err := b.Generative.ChangeGenerativeModelName(update.Message.Text); err != nil {
		logrus.WithError(err).Error("Change generative model failed")
		b.sendMessage(b.ChatID, "На данный момент сменить генеративную модель не удалось. Попробуй проверить правильно ли ты указал название модели или есть ли к ней доступ у твоего аккаунта!", update.Message.MessageID, nil)
		return err
	}

	return b.sendMessage(b.ChatID, "Смена произошла успешно!", update.Message.MessageID, nil)
}

// UpdateProcessing handles incoming Telegram updates (messages and callback queries).
// Arguments:
//   - update: the Telegram update to process.
//   - usersState: the user state repository instance.
func (b *TgBotServices) UpdateProcessing(update *tgbotapi.Update) {
	var choiceCode string
	var errOne, errTwo error
	if update.Message != nil && update.Message.Text != "" {
		b.ChatID = update.Message.Chat.ID
		text := update.Message.Text
		switch {
		case text == constant.BUTTON_TEXT_PRINT_INTRO:
			b.printIntro()
			errOne = b.showBarMenu()
		case text == constant.BUTTON_TEXT_SKIP_INTRO:
			errOne = b.showBarMenu()
		case text == constant.BUTTON_TEXT_PRINT_MENU:
			errOne = b.showBarMenu()
			//TODO исправить проблему выдачи одинаковых ответов в inline режиме
		case text == constant.BUTTON_TEXT_WHAT_TO_DO:
			errOne = b.SendActivityMsg()
			errTwo = b.showBarMenu()
		case text == constant.BUTTON_TEXT_WHITCH_MOVIE_TO_WATCH:
			errOne = b.SendMoviesLink()
			errTwo = b.showBarMenu()
		case text == constant.BUTTON_TEXT_YANDEX_DDIALOGS:
			errOne = b.showSmartMenu()
		case text == constant.BUTTON_TEXT_YANDEX_LOGIN:
			errOne = b.showOAuthButton()
		case text == constant.BUTTON_TEXT_YANDEX_GET_HOME_INFO:
			errOne = b.showSmartHomeInfo()
			errTwo = b.showSmartMenu()
		case text == constant.BUTTON_TEXT_GENERATIVE_MENU:
			errOne = b.showGenerativeMenu()
		case text == constant.BUTTON_TEXT_CHANGE_MODEL:
			b.Repository.StoreUserState(b.ChatID, "смена ИИ", "", choiceCode, false, false, true) // Устанавливаем состояние перевода в true
			errOne = b.sendMessage(b.ChatID, "Вы в режиме смены генеративной модели.\nВведи название генеративной модели с сайта https://openrouter.ai/models. Например: deepseek/deepseek-chat-v3-0324:free или /stop для выхода.", 0, nil)
		case text == constant.BUTTON_TEXT_GENERATIVE_MODEL:
			b.Repository.StoreUserState(b.ChatID, "ИИ", "", choiceCode, false, true, false) // Устанавливаем состояние перевода в true
			errOne = b.sendMessage(b.ChatID, "Вы в режиме общения с ИИ.\nВведите свой вопрос или /stop для выхода.", 0, nil)
		case text == constant.BUTTON_TEXT_TRANSLATE:
			b.Repository.StoreUserState(b.ChatID, "перевод", "", choiceCode, true, false, false) // Устанавливаем состояние перевода в true
			errOne = b.sendMessage(b.ChatID, "Вы в режиме перевода.\nВведите текст для перевода или /stop для выхода.", 0, nil)
		case text == "/start":
			logrus.Infof("Message [%s] from %s (chat %d)", update.Message.Text, update.Message.From.UserName, b.ChatID)
			b.Repository.StoreUserState(b.ChatID, "старт", update.Message.Text, "", false, false, false)
			errOne = b.askToPrintIntro()
		case text == "/stop":
			b.Repository.StoreUserState(b.ChatID, "стоп", update.Message.Text, "", false, false, false)
			errOne = b.sendMessage(b.ChatID, "Возврат в основное меню", 0, nil)
			errTwo = b.showBarMenu()
		case b.Repository.GetChangeModelState(b.ChatID):
			errOne = b.changeGenerativeModel(update)
			errTwo = b.showBarMenu()
		case b.Repository.GetTranslateState(b.ChatID):
			errOne = b.translateText(update)
		case b.Repository.GetGenerativeState(b.ChatID):
			errOne = b.generativeText(update)
		case text == "Включить: "+strings.TrimPrefix(text, "Включить: "), text == "Выключить: "+strings.TrimPrefix(text, "Выключить: "): // Dynamic device buttons
			deviceName := strings.Fields(text)[1]
			fmt.Println(deviceName)
			errOne = b.setDeviceTurnOnOffStatus(deviceName)
			errTwo = b.showSmartMenu()
		default:
			b.Repository.StoreUserState(b.ChatID, "i can't do it now", update.Message.Text, "", false, false, false)
			errOne = b.sendSorryMsg(update)
		}
		if errOne != nil || errTwo != nil {
			logrus.Error("errOne: ", errOne, "\n", "errTwo: ", errTwo)
		}

	}
}
