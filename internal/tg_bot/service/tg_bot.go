// Package service provides the core logic for a Telegram bot, integrating various services.
// It handles user interactions, inline queries, and Yandex Smart Home operations.
package service

import (
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/constant"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
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
	MoviesURL         string                    // External URL with movie подборкой
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
func NewTgBot(boring Boring, translate Translate, smartHome SmartHome, generative GenerativeModel, stateRepository UsersChatStateRepository, aiDialogRepository AIDialogHistoryRepository, bot *tgbotapi.BotAPI, handler Handler, URL string, ownerID int64, moviesURL string) *TgBotServices {
	return &TgBotServices{
		Boring:            boring,
		Translate:         translate,
		SmartHome:         smartHome,
		Generative:        generative,
		StateRepo:         stateRepository,
		AIDialogRepo:      aiDialogRepository,
		dialogHistorySize: 50,
		Bot:               bot,
		Handler:           handler,
		OAuthURL:          URL,
		OwnerID:           ownerID,
		MoviesURL:         moviesURL,
		debounceTimers:    make(map[int64]*time.Timer),
		lastQueries:       make(map[int64]string),
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

func (b *TgBotServices) setModeState(currentStep string, isTranslating, isGenerative, isChangingGenModel, isChangingHistorySize bool) {
	b.StateRepo.StoreUserState(b.ChatID, currentStep, "", "", isTranslating, isGenerative, isChangingGenModel, isChangingHistorySize)
}

func (b *TgBotServices) handleModeInput(update *tgbotapi.Update) (error, error, bool) {
	switch {
	case b.StateRepo.GetChangeHistorySizeState(b.ChatID):
		return b.changeHistorySize(update), b.showBarMenu(), true
	case b.StateRepo.GetChangeModelState(b.ChatID):
		return b.changeGenerativeModel(update), b.showBarMenu(), true
	case b.StateRepo.GetTranslateState(b.ChatID):
		return b.translateText(update), nil, true
	case b.StateRepo.GetGenerativeState(b.ChatID):
		return b.generativeTextWithStream(update), nil, true
	default:
		return nil, nil, false
	}
}

func (b *TgBotServices) parseDeviceToggleCommand(text string) (string, bool) {
	if strings.HasPrefix(text, "Включить: ") {
		return strings.TrimPrefix(text, "Включить: "), true
	}
	if strings.HasPrefix(text, "Выключить: ") {
		return strings.TrimPrefix(text, "Выключить: "), true
	}
	return "", false
}

func (b *TgBotServices) handleTextCommand(update *tgbotapi.Update, text string) (error, error, bool) {
	switch text {
	case constant.BUTTON_TEXT_PRINT_INTRO:
		b.printIntro()
		return b.showBarMenu(), nil, true
	case constant.BUTTON_TEXT_SKIP_INTRO, constant.BUTTON_TEXT_PRINT_MENU:
		return b.showBarMenu(), nil, true
	case constant.BUTTON_TEXT_WHAT_TO_DO:
		return b.SendActivityMsg(), b.showBarMenu(), true
	case constant.BUTTON_TEXT_WHITCH_MOVIE_TO_WATCH:
		return b.SendMoviesLink(), b.showBarMenu(), true
	case constant.BUTTON_TEXT_YANDEX_DDIALOGS:
		return b.showSmartMenu(), nil, true
	case constant.BUTTON_TEXT_YANDEX_LOGIN:
		return b.showOAuthButton(), nil, true
	case constant.BUTTON_TEXT_YANDEX_GET_HOME_INFO:
		return b.showSmartHomeInfo(), b.showSmartMenu(), true
	case constant.BUTTON_TEXT_GENERATIVE_MENU:
		return b.showGenerativeMenu(), nil, true
	case constant.BUTTON_TEXT_CHANGE_HISTORY_SIZE:
		b.setModeState("смена памяти ИИ", false, false, false, true)
		return b.sendMessage(b.ChatID, "Ты в режиме смены размера памяти генеративной модели.\nВведи целое число от 1 до 200 или /stop для выхода.", 0, nil), nil, true
	case constant.BUTTON_TEXT_CHANGE_MODEL:
		b.setModeState("смена ИИ", false, false, true, false)
		return b.sendMessage(b.ChatID, "Ты в режиме смены генеративной модели.\nВведи название генеративной модели с сайта https://openrouter.ai/models. Например: deepseek/deepseek-chat-v3-0324:free или /stop для выхода.", 0, nil), nil, true
	case constant.BUTTON_TEXT_GENERATIVE_MODEL, constant.BUTTON_TEXT_STREAM_GENERATIVE_MODEL:
		b.setModeState("ИИ", false, true, false, false)
		return b.sendMessage(b.ChatID, "Вы в режиме общения с ИИ.\nВведите свой вопрос или /stop для выхода.", 0, nil), nil, true
	case constant.BUTTON_TEXT_TRANSLATE:
		b.setModeState("перевод", true, false, false, false)
		return b.sendMessage(b.ChatID, "Вы в режиме перевода.\nВведите текст для перевода или /stop для выхода.", 0, nil), nil, true
	case "/start":
		logrus.Infof("Message [%s] from %s (chat %d)", update.Message.Text, update.Message.From.UserName, b.ChatID)
		b.StateRepo.StoreUserState(b.ChatID, "старт", update.Message.Text, "", false, false, false, false)
		return b.askToPrintIntro(), nil, true
	case "/stop":
		b.StateRepo.StoreUserState(b.ChatID, "стоп", update.Message.Text, "", false, false, false, false)
		return b.sendMessage(b.ChatID, "Возврат в основное меню", 0, nil), b.showBarMenu(), true
	default:
		deviceName, ok := b.parseDeviceToggleCommand(text)
		if !ok {
			return nil, nil, false
		}
		return b.setDeviceTurnOnOffStatus(deviceName), b.showSmartMenu(), true
	}
}

// UpdateProcessing handles incoming Telegram updates (messages and callback queries).
// Arguments:
//   - update: the Telegram update to process.
//   - usersState: the user state repository instance.
func (b *TgBotServices) UpdateProcessing(update *tgbotapi.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	b.ChatID = update.Message.Chat.ID
	text := update.Message.Text

	errOne, errTwo, handled := b.handleTextCommand(update, text)
	if !handled {
		errOne, errTwo, handled = b.handleModeInput(update)
	}
	if !handled {
		b.StateRepo.StoreUserState(b.ChatID, "i can't do it now", update.Message.Text, "", false, false, false, false)
		errOne = b.sendSorryMsg(update)
	}
	if errOne != nil || errTwo != nil {
		logrus.Error("errOne: ", errOne, "\n", "errTwo: ", errTwo)
	}
}
