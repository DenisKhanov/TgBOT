package main

import (
	"GoProgects/PetProjects/cmd/api"
	"GoProgects/PetProjects/internal/app/config"
	"GoProgects/PetProjects/internal/app/constant"
	"GoProgects/PetProjects/internal/app/logcfg"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"
)

func sendMessage(bot *tgbotapi.BotAPI, msg tgbotapi.MessageConfig) bool {
	for i := 0; i < 5; i++ {
		_, err := bot.Send(msg)
		if err != nil {
			logrus.Error(err)
			time.Sleep(500 * time.Millisecond)
		} else {
			return true
		}
	}
	return false
}

var botChatID int64
var bot *tgbotapi.BotAPI

func sendIntroMassegeWithDelay(delayInSec uint8, text string) {
	msg := tgbotapi.NewMessage(botChatID, text)
	time.Sleep(time.Duration(delayInSec) * time.Second)
	sendMessage(bot, msg)
}

func printIntro() {
	sendIntroMassegeWithDelay(1, "Привет, пока что я небольшой bot-проект")
	sendIntroMassegeWithDelay(2, "Но мои возможности регулярно растут")
	sendIntroMassegeWithDelay(1, constant.EMOJI_BICEPS)
}

func main() {
	cfg := config.NewConfig()
	logrus.Infof("BOT started with configuration logs level: %v", cfg.EnvLogs)

	logcfg.RunLoggerConfig(cfg.EnvLogs)

	token, err := os.ReadFile("tokenBOT.txt")
	if err != nil {
		logrus.Error(err)
	}
	bot, err = tgbotapi.NewBotAPI(string(token))
	if err != nil {
		logrus.Panic(err)
	}
	bot.Debug = true
	logrus.Infof("Bot API created successfully for %s", bot.Self.UserName)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60 //seconds timeout
	for update := range bot.GetUpdatesChan(updateConfig) {
		if update.Message != nil {
			botChatID = update.Message.Chat.ID
			logrus.Infof("[%s]%s", update.Message.From.UserName, update.Message.Text)
			var msg tgbotapi.MessageConfig
			switch {
			case update.Message.Text == "/start":
				printIntro()
				keyboard := tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Чем мне заняться?"),
						tgbotapi.NewKeyboardButton("Кнопка 2"),
					))
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите опцию:")
				msg.ReplyMarkup = keyboard

			case strings.ToLower(update.Message.Text) == "чем мне заняться?":
				text, errAPI := api.BoredAPI()
				fmt.Println(text)
				fmt.Println(errAPI)
				if errAPI == nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, text)
					msg.ReplyToMessageID = update.Message.MessageID
				}
			default:
				msg = tgbotapi.NewMessage(botChatID, "Я пока этого не умею, но я учусь")
				msg.ReplyToMessageID = update.Message.MessageID
			}
			if !sendMessage(bot, msg) {
				logrus.Infof("Не удалось отправить сообщение")
			}

		}
	}
}
