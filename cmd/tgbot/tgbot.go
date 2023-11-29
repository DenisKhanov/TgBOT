package main

import (
	"GoProgects/PetProjects/cmd/api"
	"GoProgects/PetProjects/internal/app/config"
	"GoProgects/PetProjects/internal/app/custom"
	"GoProgects/PetProjects/internal/app/logcfg"
	"GoProgects/PetProjects/internal/app/repository"
	"GoProgects/PetProjects/internal/app/services"
	"context"
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

	bot, err := tgbotapi.NewBotAPI(cfg.EnvBotToken)
	if err != nil {
		logrus.Panic(err)
	}
	bot.Debug = true
	customBot := &custom.BotAPICustom{BotAPI: bot}
	usersState := repository.NewUsersStateMap(cfg.EnvStoragePath)
	myBoringAPI := api.NewBoringAPI("http://www.boredapi.com/api/activity/")
	myYandexAPI := api.NewYandexAPI("https://translate.api.cloud.yandex.net/translate/v2/translate",
		"https://translate.api.cloud.yandex.net/translate/v2/detect", cfg.EnvYandexToken)
	myBot := services.NewTgBot(myBoringAPI, myYandexAPI, usersState, bot)

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Горутина для обработки сигналов остановки и тикера
	go func() {
		for {
			select {
			case <-ticker.C: // Событие от тикера
				if err = myBot.Repository.SaveBatchToFile(); err != nil {
					logrus.Error("Error while saving state on ticker: ", err)
				}
			case sig := <-signalChan: // При получении сигнала остановки
				logrus.Infof("Received %v signal, shutting down bot...", sig)
				if err = myBot.Repository.SaveBatchToFile(); err != nil {
					logrus.Error("Error while saving state on shutdown: ", err)
				}
				cancel() // Отправляем сигнал об остановке в основной цикл
				return
			}
		}
	}()

	// Основной цикл обработки обновлений
	for update := range customBot.GetUpdatesChan(ctx, updateConfig) { // Получение обновлений
		if update.InlineQuery != nil {
			myBot.HandleInlineQuery(bot, update.InlineQuery)
		} else {
			myBot.UpdateProcessing(&update, usersState)
		} // Когда получен сигнал об остановке
	}
	logrus.Info("Shutting down main loop...")

}
