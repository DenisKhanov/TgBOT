package repository

import (
	"bufio"
	"encoding/json"
	"errors"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

type UsersState struct {
	BatchBuffer     map[int64]*models.UserState `json:"batchBuffer"`
	storageFilePath string
}

func NewUsersStateMap(envStoragePath string) *UsersState {
	return &UsersState{
		BatchBuffer:     make(map[int64]*models.UserState),
		storageFilePath: envStoragePath,
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
	var bufferFromJSON models.UserState
	for scanner.Scan() {
		buffer = scanner.Bytes()
		err = json.Unmarshal(buffer, &bufferFromJSON)
		if err != nil {
			logrus.Error(err)
			return err
		}
		m.BatchBuffer[bufferFromJSON.ChatID] = &bufferFromJSON
	}
	if err = scanner.Err(); err != nil {
		logrus.Error(err)
		return err
	}
	return nil
}

//TODO реализовать сохранение всей переписки, а не только последнего сообщения

// StoreUserState saving user state in UsersState.BatchBuffer
func (m *UsersState) StoreUserState(chatID int64, currentStep, lastUserMassage, callbackQueryData string, isTranslating bool) {
	_, ok := m.BatchBuffer[chatID]
	if !ok {
		m.BatchBuffer[chatID] = &models.UserState{}
	}
	m.BatchBuffer[chatID].ChatID = chatID
	m.BatchBuffer[chatID].CurrentStep = currentStep
	m.BatchBuffer[chatID].LastUserMessages = lastUserMassage
	m.BatchBuffer[chatID].CallbackQueryData = callbackQueryData
	m.BatchBuffer[chatID].IsTranslating = isTranslating
}

func (m *UsersState) SaveUserYandexSmartHomeInfo(chatID int64, token string, userDevices map[string]*models.Device) {
	m.BatchBuffer[chatID].Token = token
	m.BatchBuffer[chatID].Devices = userDevices
}
func (m *UsersState) GetUserYandexSmartHomeToken(chatID int64) (string, error) {
	token := m.BatchBuffer[chatID].Token
	if token == "" {
		return "", errors.New("token not found")
	}
	return token, nil
}
func (m *UsersState) GetUserYandexSmartHomeDevices(chatID int64) (map[string]*models.Device, error) {
	devices := m.BatchBuffer[chatID].Devices
	if len(devices) == 0 {
		return nil, errors.New("no one device found ")
	}
	return devices, nil
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
	logrus.Infof("%v saved in file %s in %v", m.BatchBuffer, m.storageFilePath, elapsedTime)
	return nil
}
