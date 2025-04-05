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
	BoredAPI() string
}

// YandexTranslate defines the interface for translation operations.
type YandexTranslate interface {
	TranslateAPI(text string) (string, error)  // Retrieves smart home device info.
	DetectLangAPI(text string) (string, error) // Toggles a device on or off.
}

// YandexSmartHome defines the interface for Yandex Smart Home operations.
type YandexSmartHome interface {
	GetHomeInfo(token string) (map[string]*models.Device, error)
	TurnOnOffAction(token, id string, value bool) error
}

// YandexAuth defines the interface for Yandex authentication (not implemented in this code).
type YandexAuth interface {
	AuthAPI(accessCode string) (string, error)
}

// The Repository defines the interface for user state persistence.
type Repository interface {
	ReadFileToMemoryURL() error
	SaveBatchToFile() error
	StoreUserState(chatID int64, currentStep, lastUserMassage, callbackQueryData string, isTranslating bool)
	SaveUserYandexSmartHomeInfo(chatID int64, token string, devices map[string]*models.Device)
	GetUserYandexSmartHomeToken(chatID int64) (string, error)
	GetUserYandexSmartHomeDevices(chatID int64) (map[string]*models.Device, error)
	GetTranslateState(chatID int64) bool
}

// Handler defines the interface for OAuth token handling.
type Handler interface {
	GetUserToken(chatID int64) (models.ResponseOAuth, error)
}

// TgBotServices is the main service struct for the Telegram bot, integrating all dependencies.
type TgBotServices struct {
	Boring          Boring                // Activity suggestion service.
	YandexTranslate YandexTranslate       // Translation service.
	YandexSmartHome YandexSmartHome       // Smart home service.
	Repository      Repository            // User state repository.
	ChatID          int64                 // Current chat ID.
	Bot             *tgbotapi.BotAPI      // Telegram Bot API instance.
	Handler         Handler               // OAuth handler.
	OAuthURL        string                // URL for OAuth authentication.
	OwnerID         int64                 // Owner's chatID for access to Yandex smart home menu button
	debounceTimers  map[int64]*time.Timer // Per-chat debounce timers
	mu              *sync.Mutex           // Protects debounceTimers
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
func NewTgBot(boring Boring, yandex YandexTranslate, yandexSmartHome YandexSmartHome, repository Repository, bot *tgbotapi.BotAPI, handler Handler, URL string, ownerID int64) *TgBotServices {
	return &TgBotServices{
		Boring:          boring,
		YandexTranslate: yandex,
		YandexSmartHome: yandexSmartHome,
		Repository:      repository,
		Bot:             bot,
		Handler:         handler,
		OAuthURL:        URL,
		OwnerID:         ownerID,
		debounceTimers:  make(map[int64]*time.Timer),
		mu:              &sync.Mutex{},
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
	b.mu.Lock()
	defer b.mu.Unlock()

	chatID := query.From.ID // Используем ID пользователя как ключ для debounce
	currentInput := query.Query

	// Останавливаем предыдущий таймер, если он существует
	if timer, exists := b.debounceTimers[chatID]; exists {
		timer.Stop()
	}

	// Запускаем новый таймер с debounce на 1.5 секунды
	b.debounceTimers[chatID] = time.AfterFunc(1500*time.Millisecond, func() {
		var results []interface{}

		// Если есть ввод, показываем перевод
		if currentInput != "" {
			translatedText, err := b.YandexTranslate.TranslateAPI(currentInput)
			if err != nil {
				logrus.WithError(err).Error("Inline query translation failed")
				return
			}
			result := tgbotapi.NewInlineQueryResultArticleMarkdown(
				"1", // Уникальный ID результата
				"Перевести введенный текст", // Заголовок
				translatedText, // Текст результата
			)
			results = append(results, result)
		} else {
			// Если ввода нет, показываем два варианта
			// 1. Предложение активности
			activity := b.Boring.BoredAPI()
			activityResult := tgbotapi.NewInlineQueryResultArticleMarkdown(
				"1",
				"Предложи чем мне заняться",
				activity,
			)
			results = append(results, activityResult)

			// 2. Подборка фильмов со ссылкой
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

		fmt.Println("Обновляю")
		// Конфигурация ответа inline-запроса
		inlineConf := tgbotapi.InlineConfig{
			InlineQueryID: query.ID,
			Results:       results,
			CacheTime:     0, // Отключаем кэширование для свежести результатов
			IsPersonal:    true,
		}

		// Используем AnswerInlineQuery вместо Send
		if _, err := bot.Request(inlineConf); err != nil {
			logrus.WithError(err).Error("Failed to send inline query response")
		}
	})
}

// SendActivityMsg sends a random activity suggestion to the current chat.
// Returns an error if the message fails to send.
func (b *TgBotServices) SendActivityMsg() error {
	text := b.Boring.BoredAPI()
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

// showYandexSmartMenu displays a menu for Yandex Smart Home controls, restricted to the bot owner.
// Returns an error if the user is not the owner, authentication fails, or the message fails to send it.
func (b *TgBotServices) showYandexSmartMenu() error {
	if b.ChatID != b.OwnerID {
		return b.sendMessage(b.ChatID, "Извини, но доступ к этому меню есть только у моего Хозяина.", 0, nil)
	}
	if _, err := b.Repository.GetUserYandexSmartHomeToken(b.ChatID); err != nil {
		if err = b.GetYandexSmartHomeToken(b.ChatID); err != nil {
			return b.showYandexOAuthButton()
		}
	}
	devices, err := b.Repository.GetUserYandexSmartHomeDevices(b.ChatID)
	if err != nil {
		return b.sendMessage(b.ChatID, "Не удалось загрузить устройства", 0, nil)
	}

	//return b.sendMessage(b.ChatID, "Мы советует эти заведения из раздела Бар:", 0, markup)
	rows := [][]tgbotapi.KeyboardButton{
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_PRINT_MENU),
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_YANDEX_GET_HOME_INFO),
		),
	}
	//TODO: реализовать вкл или выкл в зависимости от состояния устройства

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

// showYandexOAuthButton prompts the user to authenticate with Yandex for Smart Home access.
// Returns an error if the message fails to send.
func (b *TgBotServices) showYandexOAuthButton() error {
	strChatID := strconv.FormatInt(b.ChatID, 10)
	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(constant.BUTTON_TEXT_YANDEX_SEND_CODE, b.OAuthURL+strChatID),
		),
	)
	return b.sendMessage(b.ChatID, "Нужно пройти аутентификацию ↓", 0, markup)
}

// GetYandexSmartHomeToken retrieves and stores a Yandex Smart Home token for the specified chat.
// Arguments:
//   - chatID: the ID of the chat to authenticate.
//
// Returns an error if token retrieval or device info fetching fails.
func (b *TgBotServices) GetYandexSmartHomeToken(chatID int64) error {
	tokenData, err := b.Handler.GetUserToken(chatID)
	if err != nil {
		logrus.WithError(err).Error("Failed to get Yandex token")
		return err
	}

	userDevices, err := b.YandexSmartHome.GetHomeInfo(tokenData.AccessToken)
	if err != nil {
		b.sendMessage(b.ChatID, "Произошла ошибка, не удалось получить информацию об устройствах", 0, nil)
		return fmt.Errorf("failed to get home info: %w", err)
	}

	b.Repository.SaveUserYandexSmartHomeInfo(b.ChatID, tokenData.AccessToken, userDevices)
	return b.sendMessage(b.ChatID, "Авторизация прошла успешно", 0, nil)
}

// SendUserHomeInfo sends information about the user's Yandex Smart Home devices.
// Returns an error if token retrieval, device info fetching, or message sending fails.
func (b *TgBotServices) SendUserHomeInfo() error {
	token, err := b.Repository.GetUserYandexSmartHomeToken(b.ChatID)
	if err != nil {
		return b.sendMessage(b.ChatID, "Произошла ошибка, похоже вы не прошли авторизацию", 0, nil)
	}

	userHomeInfoData, err := b.YandexSmartHome.GetHomeInfo(token)
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

// YandexDeviceTurnOnOff toggles the state of a specified Yandex Smart Home device.
// Arguments:
//   - deviceName: the name of the device to toggle.
//
// Returns an error if token retrieval, device lookup, state change, or message sending fails.
func (b *TgBotServices) YandexDeviceTurnOnOff(deviceName string) error {
	token, err := b.Repository.GetUserYandexSmartHomeToken(b.ChatID)
	if err != nil {
		return b.sendMessage(b.ChatID, "Произошла ошибка, похоже вы не прошли авторизацию", 0, nil)
	}

	devices, err := b.Repository.GetUserYandexSmartHomeDevices(b.ChatID)
	if err != nil {
		return b.sendMessage(b.ChatID, "Произошла ошибка, устройства не найдены", 0, nil)
	}
	device, ok := devices[deviceName]
	if !ok {
		return b.sendMessage(b.ChatID, fmt.Sprintf("Устройство %s не найдено", deviceName), 0, nil)

	}

	if err = b.YandexSmartHome.TurnOnOffAction(token, device.ID, device.ActualState); err != nil {
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
	translatedText, err := b.YandexTranslate.TranslateAPI(update.Message.Text)
	if err != nil {
		logrus.WithError(err).Error("Translation failed")
		return err
	}
	return b.sendMessage(b.ChatID, translatedText, update.Message.MessageID, nil)
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
			errOne = b.showYandexSmartMenu()
		case text == constant.BUTTON_TEXT_YANDEX_LOGIN:
			errOne = b.showYandexOAuthButton()
		case text == constant.BUTTON_TEXT_YANDEX_GET_HOME_INFO:
			errOne = b.SendUserHomeInfo()
			errTwo = b.showYandexSmartMenu()
		case text == constant.BUTTON_TEXT_TRANSLATE:
			b.Repository.StoreUserState(b.ChatID, "перевод", "", choiceCode, true) // Устанавливаем состояние перевода в true
			errOne = b.sendMessage(b.ChatID, "Вы в режиме перевода.\nВведите текст для перевода или /stop для выхода.", 0, nil)
		case text == "/start":
			logrus.Infof("Message [%s] from %s (chat %d)", update.Message.Text, update.Message.From.UserName, b.ChatID)
			b.Repository.StoreUserState(b.ChatID, "старт", update.Message.Text, "", false)
			errOne = b.askToPrintIntro()
		case text == "/stop":
			b.Repository.StoreUserState(b.ChatID, "стоп", update.Message.Text, "", false)
			errOne = b.sendMessage(b.ChatID, "Режим перевода выключен.", 0, nil)
			errTwo = b.showBarMenu()
		case b.Repository.GetTranslateState(b.ChatID):
			errOne = b.translateText(update)
		case text == "Включить: "+strings.TrimPrefix(text, "Включить: "), text == "Выключить: "+strings.TrimPrefix(text, "Выключить: "): // Dynamic device buttons
			deviceName := strings.Fields(text)[1]
			fmt.Println(deviceName)
			errOne = b.YandexDeviceTurnOnOff(deviceName)
			errTwo = b.showYandexSmartMenu()
		default:
			b.Repository.StoreUserState(b.ChatID, "i can't do it now", update.Message.Text, "", false)
			errOne = b.sendSorryMsg(update)
		}
		if errOne != nil || errTwo != nil {
			logrus.Error("errOne: ", errOne, "\n", "errTwo: ", errTwo)
		}

	}
}
