package repository

import (
	"bufio"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

type UserState struct {
	ChatID            int64  `json:"chatID"`            // Идентификатор чата
	CurrentStep       string `json:"currentStep"`       // Текущий этап диалога с пользователем
	LastUserMessages  string `json:"lastUserMessages"`  // Данные, введённые пользователем, ключ - название данных
	CallbackQueryData string `json:"callbackQueryData"` // Данные из callback-запросов, если они используются
	IsTranslating     bool   `json:"isTranslating"`     // Флаг состояния перевода для пользователя
}
type UsersState struct {
	BatchBuffer     map[int64]*UserState `json:"batchBuffer"`
	storageFilePath string
}

func NewUsersState(envStoragePath string) *UsersState {
	return &UsersState{
		BatchBuffer:     make(map[int64]*UserState),
		storageFilePath: envStoragePath,
	}
}
func newUserState(chatID int64) *UserState {
	return &UserState{
		ChatID:            chatID,
		CurrentStep:       "/start",
		LastUserMessages:  "",
		CallbackQueryData: "",
		IsTranslating:     false,
	}
}

func (m *UsersState) ReadFileToMemoryURL() error {
	file, err := os.Open(m.storageFilePath)
	if err != nil {
		logrus.Error(err)
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var buffer []byte
	var bufferJSON UserState
	for scanner.Scan() {
		buffer = scanner.Bytes()
		err = json.Unmarshal(buffer, &bufferJSON)
		if err != nil {
			logrus.Error(err)
			return err
		}
		m.BatchBuffer[bufferJSON.ChatID] = &bufferJSON
	}
	if err = scanner.Err(); err != nil {
		logrus.Error(err)
		return err
	}
	return nil
}

// StoreUserState saving user state in UsersState.BatchBuffer
func (m *UsersState) StoreUserState(chatID int64, currentStep, lastUserMassage, callbackQueryData string, isTranslating bool) {
	m.BatchBuffer[chatID] = newUserState(chatID)
	m.BatchBuffer[chatID].CurrentStep = currentStep
	m.BatchBuffer[chatID].LastUserMessages = lastUserMassage
	m.BatchBuffer[chatID].CallbackQueryData = callbackQueryData
	m.BatchBuffer[chatID].IsTranslating = isTranslating
}

func (m *UsersState) SaveBatchToFile() error {
	startTime := time.Now() // Засекаем время начала операции
	file, err := os.OpenFile(m.storageFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		logrus.Error(err)
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)
	for _, v := range m.BatchBuffer {
		err = encoder.Encode(v)
		if err != nil {
			return err
		}
	}
	err = writer.Flush() // Запись оставшихся данных из буфера в файл
	if err != nil {
		return err
	}

	elapsedTime := time.Since(startTime) // Вычисляем затраченное время
	logrus.Infof("%d URL saved in %v", m.BatchBuffer, elapsedTime)
	m.BatchBuffer = make(map[int64]*UserState)
	return nil
}
