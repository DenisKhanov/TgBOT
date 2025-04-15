// Package repository provides a user state management system for a Telegram bot.
// It stores user states in memory and persists them to a file.
package repository

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"
)

// UsersState manages the state of Telegram bot users in memory and on disk.
type UsersState struct {
	BatchBuffer     map[int64]*models.UserState `json:"batchBuffer"` // In-memory store of user states by chat ID.
	storageFilePath string                      // File path for persisting user states.
	mu              *sync.RWMutex               // Protects BatchBuffer from concurrent access
}

// NewUsersStateMap creates a new UsersState instance with an empty memory buffer.
// Arguments:
//   - envStoragePath: file path where user states are persisted.
//
// Returns a pointer to a UsersState.
func NewUsersStateMap(envStoragePath string) *UsersState {
	return &UsersState{
		BatchBuffer:     make(map[int64]*models.UserState),
		storageFilePath: envStoragePath,
		mu:              &sync.RWMutex{},
	}
}

// GetTranslateState return user's translate bool status
func (m *UsersState) GetTranslateState(chatID int64) bool {
	return m.BatchBuffer[chatID].IsTranslating
}

// GetGenerativeState return user's translate bool status
func (m *UsersState) GetGenerativeState(chatID int64) bool {
	return m.BatchBuffer[chatID].IsGenerative
}

// ReadFileToMemoryURL reads user states from the storage file into the in-memory buffer.
// Returns an error if the file cannot be read or parsed.
func (m *UsersState) ReadFileToMemoryURL() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	file, err := os.Open(m.storageFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			logrus.Infof("Storage file %s does not exist, starting with empty buffer", m.storageFilePath)
			return nil
		}
		err = fmt.Errorf("failed to open storage file %s: %w", m.storageFilePath, err)
		logrus.WithError(err).Error("Error reading storage file")
		return err
	}

	defer func() {
		if err = file.Close(); err != nil {
			logrus.WithError(err).Errorf("Failed to close file: %v", err)
		}
	}()

	data, err := os.ReadFile(m.storageFilePath)
	if err != nil {
		err = fmt.Errorf("failed to read storage file %s: %w", m.storageFilePath, err)
		logrus.WithError(err).Error("Error reading storage file")
		return err
	}

	if len(data) == 0 {
		logrus.Infof("Storage file %s is empty, starting with empty buffer", m.storageFilePath)
		return nil
	}

	var buffer map[int64]*models.UserState
	if err = json.Unmarshal(data, &buffer); err != nil {
		err = fmt.Errorf("failed to unmarshal storage file %s: %w", m.storageFilePath, err)
		logrus.WithError(err).Error("Error parsing storage file")
		return err
	}

	m.BatchBuffer = buffer
	logrus.Infof("Loaded %d user states from %s", len(m.BatchBuffer), m.storageFilePath)
	return nil
}

// StoreUserState updates or creates a user state in the in-memory buffer.
// Arguments:
//   - chatID: Telegram chat ID of the user.
//   - currentStep: current step in the bot's conversation flow.
//   - lastUserMassage: last message sent by the user.
//   - callbackQueryData: data from the last callback query.
//   - isTranslating: whether the user is currently in translation mode.
func (m *UsersState) StoreUserState(chatID int64, currentStep, lastUserMassage, callbackQueryData string, isTranslating, isGenerative bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BatchBuffer[chatID] = &models.UserState{
		ChatID:            chatID,
		CurrentStep:       currentStep,
		LastUserMessages:  lastUserMassage,
		CallbackQueryData: callbackQueryData,
		IsTranslating:     isTranslating,
		IsGenerative:      isGenerative,
	}
}

// SaveUserYandexSmartHomeInfo stores Yandex Smart Home token and device info for a user.
// Arguments:
//   - chatID: Telegram chat ID of the user.
//   - token: Yandex Smart Home OAuth token.
//   - userDevices: map of device names to device details.
func (m *UsersState) SaveUserSmartHomeInfo(chatID int64, token string, userDevices map[string]*models.Device) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BatchBuffer[chatID] = &models.UserState{
		Token:   token,
		Devices: userDevices,
	}

}

// GetUserYandexSmartHomeToken retrieves the Yandex Smart Home token for a user.
// Arguments:
//   - chatID: Telegram chat ID of the user.
//
// Returns the token or an error if not found.
func (m *UsersState) GetUserSmartHomeToken(chatID int64) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.BatchBuffer[chatID]
	if !ok || state == nil || state.Token == "" {
		err := fmt.Errorf("no token found for chatID %d", chatID)
		logrus.WithError(err).Warn("Token retrieval failed")
		return "", err
	}
	return state.Token, nil
}

// GetUserYandexSmartHomeDevices retrieves the Yandex Smart Home devices for a user.
// Arguments:
//   - chatID: Telegram chat ID of the user.
//
// Return the device map or an error if no devices are found.
func (m *UsersState) GetUserSmartHomeDevices(chatID int64) (map[string]*models.Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.BatchBuffer[chatID]
	if !ok || state == nil || len(state.Devices) == 0 {
		err := fmt.Errorf("no devices found for chatID %d", chatID)
		logrus.WithError(err).Warn("Devices retrieval failed")
		return nil, err
	}
	return state.Devices, nil
}

// SaveBatchToFile persists the in-memory user state buffer to the storage file.
// Returns an error if the file cannot be written.
func (m *UsersState) SaveBatchToFile() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	startTime := time.Now() // Засекаем время начала операции

	// Write to a temporary file first
	tempPath := m.storageFilePath + ".tmp"
	file, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		err = fmt.Errorf("failed to open temp file %s: %w", tempPath, err)
		logrus.WithError(err).Error("Error saving batch to file")
		return err
	}
	defer func() {
		if err = file.Close(); err != nil {
			logrus.WithError(err).Errorf("Failed to close file: %v", err)
		}
	}()

	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)
	if err = encoder.Encode(m.BatchBuffer); err != nil {
		err = fmt.Errorf("failed to encode batch to temp file %s: %w", tempPath, err)
		logrus.WithError(err).Error("Error encoding batch")
		return err
	}
	if err = writer.Flush(); err != nil {
		err = fmt.Errorf("failed to flush temp file %s: %w", tempPath, err)
		logrus.WithError(err).Error("Error flushing batch")
		return err
	}

	// Atomically rename a temp file to final destination
	if err = os.Rename(tempPath, m.storageFilePath); err != nil {
		err = fmt.Errorf("failed to rename temp file %s to %s: %w", tempPath, m.storageFilePath, err)
		logrus.WithError(err).Error("Error finalizing batch save")
		return err
	}

	elapsedTime := time.Since(startTime)
	logrus.Infof("Saved %d user states to %s in %v", len(m.BatchBuffer), m.storageFilePath, elapsedTime)
	return nil
}
