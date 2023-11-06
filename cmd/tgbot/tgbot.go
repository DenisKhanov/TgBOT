package main

import (
	"GoProgects/PetProjects/cmd/api"
	"GoProgects/PetProjects/internal/app/config"
	"GoProgects/PetProjects/internal/app/constant"
	"GoProgects/PetProjects/internal/app/logcfg"
	"GoProgects/PetProjects/internal/app/repository"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	botChatID int64
	bot       *tgbotapi.BotAPI
)

func sendIntroMessageWithDelay(delayInSec uint8, text string) {
	msg := tgbotapi.NewMessage(botChatID, text)
	time.Sleep(time.Duration(delayInSec) * time.Second)
	bot.Send(msg)
}
func getKeyboardRow(buttonText, buttonCode string) []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(buttonText, buttonCode))
}

func printIntro() {
	sendIntroMessageWithDelay(1, "Привет, пока что я небольшой bot-проект")
	sendIntroMessageWithDelay(2, "Но мои возможности регулярно растут")
	sendIntroMessageWithDelay(1, constant.EMOJI_BICEPS)
}
func askToPrintIntro() {
	msg := tgbotapi.NewMessage(botChatID, "Это приветственное вступление, в нем описываются возможности бота, ты можешь пропустить его. Что ты выберешь?")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		getKeyboardRow(constant.BUTTON_TEXT_PRINT_INTRO, constant.BUTTON_CODE_PRINT_INTRO),
		getKeyboardRow(constant.BUTTON_TEXT_SKIP_INTRO, constant.BUTTON_CODE_SKIP_INTRO),
	)
	bot.Send(msg)
}
func showMenu() {
	msg := tgbotapi.NewMessage(botChatID, "Выберите способность:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(constant.BUTTON_TEXT_WHAT_TO_DO, constant.BUTTON_CODE_WHAT_TO_DO),
			tgbotapi.NewInlineKeyboardButtonData(constant.BUTTON_TEXT_TRANSLATE, constant.BUTTON_CODE_TRANSLATE)),
	)
	bot.Send(msg)
}

func generateActivityMsg() {
	text, errAPI := api.BoredAPI()
	var msg tgbotapi.MessageConfig
	if errAPI == nil {
		msg = tgbotapi.NewMessage(botChatID, text)
	} else {
		msg = tgbotapi.NewMessage(botChatID, "К сожалению в данный момент я не могу дотянуться до знаний")

	}
	bot.Send(msg)
}
func sendSorryMsg(update *tgbotapi.Update) {
	msg := tgbotapi.NewMessage(botChatID, "Я пока этого не умею, но я учусь")
	msg.ReplyToMessageID = update.Message.MessageID
	bot.Send(msg)
}
func translateText(update *tgbotapi.Update) {
	translatedText, err := api.TranslateAPI(update.Message.Text)
	if err != nil {
		logrus.Error(err)
	}
	msg := tgbotapi.NewMessage(botChatID, translatedText)
	msg.ReplyToMessageID = update.Message.MessageID
	bot.Send(msg)
}
func handleInlineQuery(bot *tgbotapi.BotAPI, query *tgbotapi.InlineQuery) {

	text, err := api.BoredAPI()
	if err != nil {
		logrus.Error(err)
	}
	whatToDo := []interface{}{
		tgbotapi.NewInlineQueryResultArticleMarkdown("1", "Подскажи чем мне заняться?", text),
	}
	inlineConf := tgbotapi.InlineConfig{
		InlineQueryID: query.ID,
		Results:       whatToDo,
		CacheTime:     0, // Время кэширования в секундах
	}
	if _, err = bot.Send(inlineConf); err != nil {
		log.Println("Ошибка при отправке ответа на inline-запрос:", err)
	}
}

func updateProcessing(update *tgbotapi.Update, usersState *repository.UsersState) {
	var choiceCode string
	if update.CallbackQuery != nil && update.CallbackQuery.Data != "" {
		botChatID = update.CallbackQuery.Message.Chat.ID
		choiceCode = update.CallbackQuery.Data
		fmt.Println(choiceCode)

		logrus.Infof("[%T] %s", time.Now(), choiceCode)
		switch choiceCode {
		case constant.BUTTON_CODE_PRINT_INTRO:
			usersState.StoreUserState(botChatID, "приветствие", "", choiceCode, false)
			printIntro()
			showMenu()
		case constant.BUTTON_CODE_SKIP_INTRO:
			usersState.StoreUserState(botChatID, "пропустить", "", choiceCode, false)
			showMenu()
		case constant.BUTTON_CODE_PRINT_MENU:
			usersState.StoreUserState(botChatID, "меню", "", choiceCode, false)
			showMenu()
		case constant.BUTTON_CODE_WHAT_TO_DO:
			usersState.StoreUserState(botChatID, "чемЗаняться", "", choiceCode, false)
			generateActivityMsg()
			showMenu()
		case constant.BUTTON_CODE_TRANSLATE:
			usersState.StoreUserState(botChatID, "перевод", "", choiceCode, true) // Устанавливаем состояние перевода в true
			msg := tgbotapi.NewMessage(botChatID, "Вы в режиме перевода. \nВведите текст который хотите чтобы я перевел или отправьте /stop, чтобы выйти из режима перевода.")
			bot.Send(msg)
		}
	} else if update.Message != nil && update.Message.Text != "" {
		fmt.Println("Hello")
		botChatID = update.Message.Chat.ID
		value, ok := usersState.BatchBuffer[botChatID]
		if update.Message.Text == "/stop" {
			usersState.StoreUserState(botChatID, "стоп", update.Message.Text, "", false)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Режим перевода выключен.")
			bot.Send(msg)
			showMenu()
		} else if ok && value.IsTranslating {
			translateText(update)
		} else if update.Message.Text == "/start" {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			usersState.StoreUserState(botChatID, "старт", update.Message.Text, "", false)
			botChatID = update.Message.Chat.ID
			askToPrintIntro()
		} else {
			sendSorryMsg(update)
		}
	}
}

func main() {

	cfg := config.NewConfig()
	logrus.Infof("BOT started with configuration logs level: %v", cfg.EnvLogs)

	logcfg.RunLoggerConfig(cfg.EnvLogs)

	token, err := os.ReadFile("tokenBOT.txt")
	if err != nil {
		logrus.Error(err)
	}

	usersState := repository.NewUsersState(cfg.EnvStoragePath)
	err = usersState.ReadFileToMemoryURL()
	if err != nil {
		logrus.Error(err)
	}

	bot, err = tgbotapi.NewBotAPI(string(token))
	if err != nil {
		logrus.Panic(err)
	}
	bot.Debug = true
	logrus.Infof("Bot API created successfully for %s", bot.Self.UserName)
	for _, user := range usersState.BatchBuffer {
		fmt.Println(*user)

	}

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60 //seconds timeout

	ticker := time.NewTicker(time.Minute * 5) // Например, каждые 5 минут
	defer ticker.Stop()
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			select {
			case <-ticker.C:
				if err = usersState.SaveBatchToFile(); err != nil {
					logrus.Error("Error while saving state on ticker: ", err)
				}
			case sig := <-signalChan:
				logrus.Infof("Received %v signal, shutting down bot...", sig)
				if err = usersState.SaveBatchToFile(); err != nil {
					logrus.Error("Error while saving state on shutdown: ", err)
				}
				return
			}
		}
	}()

	for update := range bot.GetUpdatesChan(updateConfig) {
		if update.InlineQuery != nil {
			handleInlineQuery(bot, update.InlineQuery)
		} else {
			updateProcessing(&update, usersState)
		}
	}
}
