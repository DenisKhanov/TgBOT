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

// Константы для идентификаторов результатов
const (
	ActivityResultID  = "activity"
	MovieResultID     = "movie"
	TranslateResultID = "translate"
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
	GenerateStreamTextMsg(text string, history []models.Message) <-chan string
	ChangeGenerativeModelName(modelName string) error
}

// The UsersChatStateRepository defines the interface for user state persistence.
type UsersChatStateRepository interface {
	ReadFileToMemoryURL() error
	SaveBatchToFile() error
	StoreUserState(chatID int64, currentStep, lastUserMassage, callbackQueryData string, isTranslating, isGenerative, isChangingGenModel, isChangingHistorySize bool)
	SaveUserSmartHomeInfo(chatID int64, token string, devices map[string]*models.Device)
	GetUserSmartHomeToken(chatID int64) (string, error)
	GetUserSmartHomeDevices(chatID int64) (map[string]*models.Device, error)
	GetTranslateState(chatID int64) bool
	GetGenerativeState(chatID int64) bool
	GetChangeModelState(chatID int64) bool
	GetChangeHistorySizeState(chatID int64) bool
}

type AIDialogHistoryRepository interface {
	LoadDialogFromFile() error
	SaveDialog(chatID int64, dialog []models.Message) error
	SaveMsgToDialog(chatID int64, msg models.Message) error
	GetDialogHistory(chatID int64) ([]models.Message, error)
	ClearHistory(chatID int64) error
	SaveBatchToFile() error
}

// Handler defines the interface for OAuth token handling.
type Handler interface {
	GetUserToken(chatID int64) (models.ResponseOAuth, error)
}

// TgBotServices is the main service struct for the Telegram bot, integrating all dependencies.
type TgBotServices struct {
	Boring            Boring    // Activity suggestion service.
	Translate         Translate // Translation service.
	SmartHome         SmartHome // Smart home service.
	Generative        GenerativeModel
	StateRepo         UsersChatStateRepository  // User state repository.
	AIDialogRepo      AIDialogHistoryRepository // User's & AI dialog history
	dialogHistorySize int                       // Max count messages in dialog history for one user
	ChatID            int64                     // Current chat ID.
	Bot               *tgbotapi.BotAPI          // Telegram Bot API instance.
	Handler           Handler                   // OAuth handler.
	OAuthURL          string                    // URL for OAuth authentication.
	OwnerID           int64                     // Owner's chatID for access to Yandex smart home menu button
	debounceTimers    map[int64]*time.Timer     // Per-chat debounce timers
	lastQueries       map[int64]string
	pendingReplies    map[string]struct {
		ChatID    int64
		MessageID int
	}
	mu *sync.Mutex // Protects debounceTimers
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
func NewTgBot(boring Boring, translate Translate, smartHome SmartHome, generative GenerativeModel, stateRepository UsersChatStateRepository, aiDialogRepository AIDialogHistoryRepository, bot *tgbotapi.BotAPI, handler Handler, URL string, ownerID int64) *TgBotServices {
	return &TgBotServices{
		Boring:         boring,
		Translate:      translate,
		SmartHome:      smartHome,
		Generative:     generative,
		StateRepo:      stateRepository,
		AIDialogRepo:   aiDialogRepository,
		Bot:            bot,
		Handler:        handler,
		OAuthURL:       URL,
		OwnerID:        ownerID,
		debounceTimers: make(map[int64]*time.Timer),
		lastQueries:    make(map[int64]string),
		pendingReplies: make(map[string]struct {
			ChatID    int64
			MessageID int
		}),
		mu: &sync.Mutex{},
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
			tgbotapi.NewKeyboardButton(constant.BUTTON_TEXT_STREAM_GENERATIVE_MODEL),
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

// HandleInlineQuery processes inline queries for translation or activity suggestions with debouncing.
// Arguments:
//   - bot: Telegram Bot API instance.
//   - query: the inline query from the user.
func (b *TgBotServices) HandleInlineQuery(bot *tgbotapi.BotAPI, query *tgbotapi.InlineQuery) {
	chatID := query.From.ID
	currentInput := query.Query

	b.mu.Lock()
	b.lastQueries[chatID] = currentInput

	// Останавливаем предыдущий таймер, если он существует
	if timer, exists := b.debounceTimers[chatID]; exists {
		timer.Stop()
	}

	var results []interface{}
	if currentInput == "" {
		results = b.getDefaultInlineResults()
		// Конфигурация ответа inline-запроса
		inlineConf := tgbotapi.InlineConfig{
			InlineQueryID:     query.ID,
			Results:           results,
			CacheTime:         0,
			IsPersonal:        true,
			SwitchPMText:      "Задать вопрос ИИ",
			SwitchPMParameter: "ask_ai",
		}

		if _, err := bot.Request(inlineConf); err != nil {
			logrus.WithError(err).Error("Failed to send inline query response")
		}
	} else {
		b.debounceTimers[chatID] = time.AfterFunc(1500*time.Millisecond, func() {
			results = b.handleTranslation(query.ID, chatID, currentInput)
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
	}
	b.mu.Unlock()
}

func (b *TgBotServices) getDefaultInlineResults() []interface{} {
	var results []interface{}

	// Если ввода нет, показываем два варианта
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
				tgbotapi.NewInlineKeyboardButtonURL("Перейти", "http://176.108.251.250:8444/"),
			),
		},
	}
	results = append(results, movieResult)
	return results
}

func (b *TgBotServices) handleTranslation(queryID string, chatID int64, input string) []interface{} {
	// Получаем блокировку для обновления состояния
	b.mu.Lock()
	lastQuery := b.lastQueries[chatID]
	// Удаляем таймер после выполнения
	delete(b.debounceTimers, chatID)
	b.mu.Unlock()

	var results []interface{}
	// Если текст запроса изменился за время ожидания, игнорируем
	if lastQuery != input {
		logrus.WithField("chatID", chatID).Info("Запрос устарел, пропускаем")
		return nil
	}

	// Выполняем перевод
	translatedText, err := b.Translate.TranslateAPI(input)
	if err != nil {
		logrus.WithError(err).
			WithField("input", input).
			Error("Inline query translate text failed")
		return nil
	}

	// Создаем результат с переводом
	result := tgbotapi.NewInlineQueryResultArticleMarkdown(
		TranslateResultID,
		"Перевести введенный текст",
		translatedText,
	)
	results = append(results, result)
	return results
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

	b.StateRepo.SaveUserSmartHomeInfo(b.ChatID, tokenData.AccessToken, userDevices)
	return b.sendMessage(b.ChatID, "Авторизация прошла успешно", 0, nil)
}

// showSmartHomeInfo sends information about the user's Smart Home devices.
// Returns an error if token retrieval, device info fetching, or message sending fails.
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
// Arguments:
//   - deviceName: the name of the device to toggle.
//
// Returns an error if token retrieval, device lookup, state change, or message sending fails.
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
	// Дополнительные настройки клавиатуры
	markup.ResizeKeyboard = true  // Подгоняет размер клавиатуры под экран
	markup.OneTimeKeyboard = true // Скрывает клавиатуру после выбора (опционально)
	return b.sendMessage(b.ChatID, "Выберите пункт ↓", 0, markup)
}

// changeHistorySize updates the maximum size limit for the dialog history based on user input.
//
// It extracts the new size from the user's message text and attempts to parse it as an integer.
// The new size must be between 1 and 200 (inclusive). If the input is invalid (e.g., not an integer
// or outside the allowed range), it sends an error message to the user. On success, it updates the
// dialog history size limit and sends a confirmation message to the user.
//
// Parameters:
//   - update: A pointer to tgbotapi.Update containing the user's message with the new history size.
//
// Returns:
//   - error: An error if sending the response message fails; nil otherwise.
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
	nesMsg := fmt.Sprintf("Теперь размер памяти истории диалога с ИИ = %d", b.dialogHistorySize)
	return b.sendMessage(b.ChatID, nesMsg, update.Message.MessageID, nil)

}

// checkSizeDialogHistory checks if the dialog history exceeds the allowed size limit.
//
// It compares the length of the provided history with the configured dialog history size limit
// stored in b.dialogHistorySize. This method is typically used to determine whether the history
// should be cleared to prevent excessive memory usage or to stay within token limits for the
// generative model.
//
// Parameters:
//   - history: A slice of models.Message representing the current dialog history.
//
// Returns:
//   - bool: True if the history size exceeds the limit; false otherwise.
func (b *TgBotServices) checkSizeDialogHistory(history []models.Message) bool {
	return len(history) > b.dialogHistorySize
}

// generativeTextWithStream generates a streaming text response from the AI model based on the user's input.
//
// This method sends an initial message to the user indicating that the request is being processed,
// retrieves the dialog history, and checks if the history exceeds the size limit. If the limit is exceeded,
// it clears the history and notifies the user. The user's message is then saved to the dialog history,
// and the generative model is invoked to produce a streaming response. The response is sent to the user
// in chunks, updating the initial message every 500 milliseconds. Once the streaming is complete,
// the final response is saved to the dialog history.
//
// Parameters:
//   - update: A pointer to tgbotapi.Update containing the user's message to process.
//
// Returns:
//   - error: An error if sending messages, saving to the dialog history, or retrieving the history fails;
//     nil otherwise. Note that errors during streaming are logged but do not interrupt the process.
func (b *TgBotServices) generativeTextWithStream(update *tgbotapi.Update) error {
	msg := tgbotapi.NewMessage(b.ChatID, "Я обрабатываю ваш запрос...")
	lastMsg, err := b.Bot.Send(msg)
	if err != nil {
		logrus.WithError(err).Error("Ошибка отправки сообщения")
	}
	history, err := b.AIDialogRepo.GetDialogHistory(b.ChatID)
	if err != nil {
		logrus.WithError(err).Error("Failed to load dialog history")
		// Продолжаем без истории, чтобы не прерывать работу
		history = []models.Message{}
	}
	if b.checkSizeDialogHistory(history) {
		if err = b.AIDialogRepo.ClearHistory(b.ChatID); err != nil {
			logrus.WithError(err).Error("Failed to clear dialog history")
		}
		msg = tgbotapi.NewMessage(b.ChatID, "Размер истории переписки с ИИ превышен и был очищен. Создан новый чат")
		_, err = b.Bot.Send(msg)
		if err != nil {
			logrus.WithError(err).Error("Ошибка отправки сообщения")
		}
	}
	// Добавляем текущее сообщение пользователя в историю
	userMsg := models.Message{
		Role:    "user",
		Content: update.Message.Text,
	}
	if err = b.AIDialogRepo.SaveMsgToDialog(b.ChatID, userMsg); err != nil {
		logrus.WithError(err).Error("Failed to save user message to dialog")
	}
	responseChan := b.Generative.GenerateStreamTextMsg(update.Message.Text, history)

	// Обрабатываем поток и обновляем сообщение
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

// changeGenerativeModel updates the generative model used by the bot based on the user's input.
//
// It attempts to change the generative model by calling ChangeGenerativeModelName with the text
// provided in the update.Message.Text. If the model change fails (e.g., due to an invalid model name
// or lack of access), it logs the error and sends a failure message to the user. On success, it sends
// a confirmation message to the user.
//
// Parameters:
//   - update: A pointer to tgbotapi.Update containing the user's message with the new model name.
//
// Returns:
//   - error: An error if the model change fails or if sending the response message fails; nil otherwise.
func (b *TgBotServices) changeGenerativeModel(update *tgbotapi.Update) error {
	if err := b.Generative.ChangeGenerativeModelName(update.Message.Text); err != nil {
		logrus.WithError(err).Error("Change generative model failed")
		b.sendMessage(b.ChatID, "На данный момент сменить генеративную модель не удалось. "+
			"Попробуй проверить правильно ли ты указал название модели или есть ли к ней доступ у твоего аккаунта!", update.Message.MessageID, nil)
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
		case text == constant.BUTTON_TEXT_CHANGE_HISTORY_SIZE:
			b.StateRepo.StoreUserState(b.ChatID, "смена памяти ИИ", "", choiceCode, false, false, false, true) // Устанавливаем состояние смены размера памяти ИИ в true
			errOne = b.sendMessage(b.ChatID, "Ты в режиме смены размера памяти генеративной модели.\nВведи целое число от 1 до 200 или /stop для выхода.", 0, nil)
		case text == constant.BUTTON_TEXT_CHANGE_MODEL:
			b.StateRepo.StoreUserState(b.ChatID, "смена ИИ", "", choiceCode, false, false, true, false) // Устанавливаем состояние смены генеративной модели в true
			errOne = b.sendMessage(b.ChatID, "Ты в режиме смены генеративной модели.\nВведи название генеративной модели с сайта https://openrouter.ai/models. Например: deepseek/deepseek-chat-v3-0324:free или /stop для выхода.", 0, nil)
		case text == constant.BUTTON_TEXT_STREAM_GENERATIVE_MODEL:
			b.StateRepo.StoreUserState(b.ChatID, "ИИ", "", choiceCode, false, true, false, false) // Устанавливаем состояние режима общения с ИИ в true
			errOne = b.sendMessage(b.ChatID, "Вы в режиме общения с ИИ.\nВведите свой вопрос или /stop для выхода.", 0, nil)
		case text == constant.BUTTON_TEXT_TRANSLATE:
			b.StateRepo.StoreUserState(b.ChatID, "перевод", "", choiceCode, true, false, false, false) // Устанавливаем состояние перевода в true
			errOne = b.sendMessage(b.ChatID, "Вы в режиме перевода.\nВведите текст для перевода или /stop для выхода.", 0, nil)
		case text == "/start":
			logrus.Infof("Message [%s] from %s (chat %d)", update.Message.Text, update.Message.From.UserName, b.ChatID)
			b.StateRepo.StoreUserState(b.ChatID, "старт", update.Message.Text, "", false, false, false, false)
			errOne = b.askToPrintIntro()
		case text == "/stop":
			b.StateRepo.StoreUserState(b.ChatID, "стоп", update.Message.Text, "", false, false, false, false)
			errOne = b.sendMessage(b.ChatID, "Возврат в основное меню", 0, nil)
			errTwo = b.showBarMenu()
		case b.StateRepo.GetChangeHistorySizeState(b.ChatID):
			errOne = b.changeHistorySize(update)
			errTwo = b.showBarMenu()
		case b.StateRepo.GetChangeModelState(b.ChatID):
			errOne = b.changeGenerativeModel(update)
			errTwo = b.showBarMenu()
		case b.StateRepo.GetTranslateState(b.ChatID):
			errOne = b.translateText(update)
		case b.StateRepo.GetGenerativeState(b.ChatID):
			errOne = b.generativeTextWithStream(update)
		case text == "Включить: "+strings.TrimPrefix(text, "Включить: "), text == "Выключить: "+strings.TrimPrefix(text, "Выключить: "): // Dynamic device buttons
			deviceName := strings.Fields(text)[1]
			fmt.Println(deviceName)
			errOne = b.setDeviceTurnOnOffStatus(deviceName)
			errTwo = b.showSmartMenu()
		default:
			b.StateRepo.StoreUserState(b.ChatID, "i can't do it now", update.Message.Text, "", false, false, false, false)
			errOne = b.sendSorryMsg(update)
		}
		if errOne != nil || errTwo != nil {
			logrus.Error("errOne: ", errOne, "\n", "errTwo: ", errTwo)
		}

	}
}
