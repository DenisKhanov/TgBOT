package main

import (
	"GoProgects/PetProjects/internal/app/config"
	"GoProgects/PetProjects/internal/app/logcfg"
	"GoProgects/PetProjects/internal/app/repository"
	"GoProgects/PetProjects/internal/app/services"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	cfg := config.NewConfig()
	logrus.Infof("BOT started with configuration logs level: %v", cfg.EnvLogs)

	logcfg.RunLoggerConfig(cfg.EnvLogs)
	token, err := os.ReadFile("tokenBOT.txt")
	if err != nil {
		logrus.Error(err)
	}

	bot, err := tgbotapi.NewBotAPI(string(token))
	if err != nil {
		logrus.Panic(err)
	}
	bot.Debug = true
	usersState := repository.NewUsersStateMap(cfg.EnvStoragePath)
	myBot := services.NewTgBot(usersState, bot)
	if err = myBot.Repository.ReadFileToMemoryURL(); err != nil {
		logrus.Error(err)
	}
	logrus.Infof("Bot API created successfully for %s", bot.Self.UserName)

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
				if err = myBot.Repository.SaveBatchToFile(); err != nil {
					logrus.Error("Error while saving state on ticker: ", err)
				}
			case sig := <-signalChan:
				logrus.Infof("Received %v signal, shutting down bot...", sig)
				if err = myBot.Repository.SaveBatchToFile(); err != nil {
					logrus.Error("Error while saving state on shutdown: ", err)
				}
				return
			}
		}
	}()

	for update := range bot.GetUpdatesChan(updateConfig) {
		if update.InlineQuery != nil {
			myBot.HandleInlineQuery(bot, update.InlineQuery)
		} else {
			myBot.UpdateProcessing(&update, usersState)
		}
	}
}
