package main

import (
	"GoProgects/PetProjects/internal/app/api"
	"GoProgects/PetProjects/internal/app/config"
	"GoProgects/PetProjects/internal/app/handlers"
	"GoProgects/PetProjects/internal/app/logcfg"
	"GoProgects/PetProjects/internal/app/repository"
	"GoProgects/PetProjects/internal/app/services"
	"fmt"
	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	//Инициализируем переменные окружения или флаги CLI
	cfg := config.NewConfig()

	logcfg.RunLoggerConfig(cfg.EnvLogsLevel)
	logrus.Infof("BOT started with configuration logs level: %v", cfg.EnvLogsLevel)

	botAPI, err := tgbotapi.NewBotAPI(cfg.EnvBotToken)
	if err != nil {
		logrus.Fatalf("[ERROR] can't make telegram bot, %v", err)
	}
	botAPI.Debug = true

	usersState := repository.NewUsersStateMap(cfg.EnvStoragePath)
	myYandexTranslate := api.NewYandexAPI("https://translate.api.cloud.yandex.net/translate/v2/translate",
		"https://translate.api.cloud.yandex.net/translate/v2/detect", cfg.EnvYandexToken)
	myBoringAPI := api.NewBoringAPI("http://www.boredapi.com/api/activity/")
	myYandexAuth := api.NewYandexAuthAPI("https://oauth.yandex.ru/token")
	myYandexSmart := api.NewYandexSmartHomeAPI("https://api.iot.yandex.net")
	myBot := services.NewTgBot(myBoringAPI, myYandexTranslate, myYandexAuth, myYandexSmart, usersState, botAPI)
	myHandlers := handlers.NewHandlers(myBot)

	if err = myBot.Repository.ReadFileToMemoryURL(); err != nil {
		logrus.Error(err)
	}
	logrus.Infof("Bot API created successfully for %s", botAPI.Self.UserName)

	// Запускаем сервер для получения get запросов OAuth
	router := gin.Default()
	router.GET("/callback", myHandlers.LogIn)
	go func() {
		if err = router.Run(":8080"); err != nil {
			fmt.Println("Failed to start server:", err)
		}
	}()
	fmt.Println("Server started on :8080")

	ticker := time.NewTicker(time.Minute * 5) // Тикер для сохранения состояния пользователя в файл каждые 5 минут
	defer ticker.Stop()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60 //seconds timeout
	updates := botAPI.GetUpdatesChan(updateConfig)

	for {
		select {
		case sig := <-signalChan: //Ожидание сигнала на завершение
			logrus.Infof("Received %v signal, shutting down botAPI...", sig)
			if err = myBot.Repository.SaveBatchToFile(); err != nil {
				logrus.Error("Error while saving state on shutdown: ", err)
			}
			logrus.Info("Shutting down main loop...")
			os.Exit(1)

		case <-ticker.C: // Событие от тикера
			if err = myBot.Repository.SaveBatchToFile(); err != nil {
				logrus.Error("Error while saving state on ticker: ", err)
			}
		case update, ok := <-updates: // получение обновлений от телеграмм
			if !ok {
				logrus.Errorf("telegram update chan closed")
			}
			if update.InlineQuery != nil {
				myBot.HandleInlineQuery(botAPI, update.InlineQuery)
			} else {
				myBot.UpdateProcessing(&update, usersState)
			}
		}
	}
}
