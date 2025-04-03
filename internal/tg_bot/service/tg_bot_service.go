// Package service provides the core logic for a Telegram bot, integrating various services.
// It handles user interactions, inline queries, and Yandex Smart Home operations.
package service

import (
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/constant"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/repository"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"strconv"
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

func (b *TgBotServices) sendIntroMessageWithDelay(delayInSec uint8, text string) {
	time.Sleep(time.Duration(delayInSec) * time.Second)
	if err := b.sendMessage(b.ChatID, text, 0, nil); err != nil {
		logrus.WithError(err).Error("Error sending intro message")
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

func (b *TgBotServices) askToPrintIntro() error {
	markup := tgbotapi.NewInlineKeyboardMarkup(
		b.getKeyboardRow(constant.BUTTON_TEXT_PRINT_INTRO, constant.BUTTON_CODE_PRINT_INTRO),
		b.getKeyboardRow(constant.BUTTON_TEXT_SKIP_INTRO, constant.BUTTON_CODE_SKIP_INTRO),
	)
	return b.sendMessage(b.ChatID, "Это приветственное вступление, в нем описываются возможности бота, но ты можешь пропустить его. Что ты выберешь?", 0, markup)

}

func (b *TgBotServices) sendSorryMsg(update *tgbotapi.Update) error {
	return b.sendMessage(b.ChatID, "Я пока этого не умею, но я учусь", update.Message.MessageID, nil)
}

func (b *TgBotServices) showHeadMenu() error {
	markup := tgbotapi.NewInlineKeyboardMarkup(
		b.getKeyboardRow(constant.BUTTON_TEXT_WHAT_TO_DO, constant.BUTTON_CODE_WHAT_TO_DO),
		b.getKeyboardRow(constant.BUTTON_TEXT_WHITCH_MOVIE_TO_WATCH, constant.BUTTON_CODE_WHITCH_MOVIE_TO_WATCH),
		b.getKeyboardRow(constant.BUTTON_TEXT_TRANSLATE, constant.BUTTON_CODE_TRANSLATE),
		b.getKeyboardRow(constant.BUTTON_TEXT_YANDEX_DDIALOGS, constant.BUTTON_CODE_YANDEX_DDIALOGS),
	)
	return b.sendMessage(b.ChatID, "Выберите способность:", 0, markup)
}

//TODO постоянно в инлайн режиме выдается одно сообщение с предложением чем заняться

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

func (b *TgBotServices) SendActivityMsg() error {
	text := b.Boring.BoredAPI()
	return b.sendMessage(b.ChatID, text, 0, nil)
}
func (b *TgBotServices) SendMoviesLink() error {
	markup := tgbotapi.NewInlineKeyboardMarkup(
		b.getKeyboardRow(constant.BUTTON_TEXT_PRINT_MENU, constant.BUTTON_CODE_PRINT_MENU),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(constant.BUTTON_TEXT_WHITCH_MOVIE_TO_WATCH, "http://176.108.251.250:8444/"),
		),
	)
	return b.sendMessage(b.ChatID, "Тут представлена подборка отличных фильмов по мнению Дениса!", 0, markup)
}

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

	rows := [][]tgbotapi.InlineKeyboardButton{
		b.getKeyboardRow(constant.BUTTON_TEXT_PRINT_MENU, constant.BUTTON_CODE_PRINT_MENU),
		b.getKeyboardRow(constant.BUTTON_TEXT_YANDEX_GET_HOME_INFO, constant.BUTTON_CODE_YANDEX_GET_HOME_INFO),
	}
	for name := range devices {
		buttonText := fmt.Sprintf("Включить/Выключить %s", name)
		buttonCode := fmt.Sprintf("device:%s", name)
		rows = append(rows, b.getKeyboardRow(buttonText, buttonCode))
	}

	markup := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return b.sendMessage(b.ChatID, "Выберите пункт:", 0, markup)
}

func (b *TgBotServices) showYandexOAuthButton() error {
	strChatID := strconv.FormatInt(b.ChatID, 10)
	markup := tgbotapi.NewInlineKeyboardMarkup(
		b.getKeyboardRow(constant.BUTTON_TEXT_PRINT_MENU, constant.BUTTON_CODE_PRINT_MENU),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(constant.BUTTON_TEXT_YANDEX_SEND_CODE, b.OAuthURL+strChatID),
		),
	)
	return b.sendMessage(b.ChatID, "Нужно пройти аутентификацию:", 0, markup)
}

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
	return b.sendMessage(b.ChatID, "Выполнено", 0, nil)
}

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
func (b *TgBotServices) UpdateProcessing(update *tgbotapi.Update, usersState *repository.UsersState) {
	var choiceCode string
	var errOne, errTwo error
	if update.CallbackQuery != nil && update.CallbackQuery.Data != "" {
		b.ChatID = update.CallbackQuery.Message.Chat.ID
		choiceCode = update.CallbackQuery.Data
		b.Repository.StoreUserState(b.ChatID, "button", "", choiceCode, false)

		logrus.Infof("Callback query [%s] from chat %d", choiceCode, b.ChatID)
		switch choiceCode {
		case constant.BUTTON_CODE_PRINT_INTRO:
			b.printIntro()
			errOne = b.showHeadMenu()
		case constant.BUTTON_CODE_SKIP_INTRO:
			errOne = b.showHeadMenu()
		case constant.BUTTON_CODE_PRINT_MENU:
			errOne = b.showHeadMenu()
			//TODO исправить проблему выдачи одинаковых ответов в инлайн режиме
		case constant.BUTTON_CODE_WHAT_TO_DO:
			errOne = b.SendActivityMsg()
			errTwo = b.showHeadMenu()
		case constant.BUTTON_CODE_WHITCH_MOVIE_TO_WATCH:
			errOne = b.SendMoviesLink()
		case constant.BUTTON_CODE_YANDEX_DDIALOGS:
			errOne = b.showYandexSmartMenu()
		case constant.BUTTON_CODE_YANDEX_LOGIN:
			errOne = b.showYandexOAuthButton()
		case constant.BUTTON_CODE_YANDEX_GET_HOME_INFO:
			errOne = b.SendUserHomeInfo()
			errTwo = b.showYandexSmartMenu()
		case "device:" + choiceCode[7:]: // Dynamic device buttons
			deviceName := choiceCode[7:]
			errOne = b.YandexDeviceTurnOnOff(deviceName)
			errTwo = b.showYandexSmartMenu()
		case constant.BUTTON_CODE_TRANSLATE:
			b.Repository.StoreUserState(b.ChatID, "перевод", "", choiceCode, true) // Устанавливаем состояние перевода в true
			errOne = b.sendMessage(b.ChatID, "Вы в режиме перевода.\nВведите текст для перевода или /stop для выхода.", 0, nil)
		}
		if errOne != nil || errTwo != nil {
			logrus.Error("errOne: ", errOne, "\n", "errTwo: ", errTwo)
		}
	} else if update.Message != nil && update.Message.Text != "" {
		b.ChatID = update.Message.Chat.ID
		value, ok := b.Repository.(*repository.UsersState).BatchBuffer[b.ChatID]
		if update.Message.Text == "/stop" {
			b.Repository.StoreUserState(b.ChatID, "стоп", update.Message.Text, "", false)
			b.sendMessage(b.ChatID, "Режим перевода выключен.", 0, nil)
			b.showHeadMenu()
		} else if ok && value.IsTranslating {
			b.translateText(update)
		} else if update.Message.Text == "/start" {
			logrus.Infof("Message [%s] from %s (chat %d)", update.Message.Text, update.Message.From.UserName, b.ChatID)
			b.Repository.StoreUserState(b.ChatID, "старт", update.Message.Text, "", false)
			b.askToPrintIntro()
		} else {
			b.Repository.StoreUserState(b.ChatID, "i can't do it now", update.Message.Text, "", false)
			b.sendSorryMsg(update)
		}
	}
}
