package repository

import (
	"encoding/json"
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"
)

// AiDialogHistory manages the storage and retrieval of dialog history for AI conversations.
//
// It maintains a thread-safe in-memory map of dialog histories, where each entry is associated with a chat ID.
// The dialog history can be persisted to and loaded from a file in JSON format. The struct ensures thread safety
// using a read-write mutex and prevents external modification by returning copies of the dialog history.
type AiDialogHistory struct {
	dialogHistory   map[int64][]models.Message // In-memory map of chat ID to dialog history
	mu              sync.RWMutex               // Mutex for thread-safe access
	storageFilePath string                     // Path to the file where dialog history is persisted
}

// NewAiDialogHistory creates a new instance of AiDialogHistory with the specified storage file path.
//
// It initializes an empty in-memory map for storing dialog histories and sets the file path for persistence.
//
// Parameters:
//   - storageFilePath: The file path where the dialog history will be persisted in JSON format.
//
// Returns:
//   - *AiDialogHistory: A pointer to the initialized AiDialogHistory instance.
func NewAiDialogHistory(storageFilePath string) *AiDialogHistory {
	return &AiDialogHistory{
		dialogHistory:   make(map[int64][]models.Message),
		mu:              sync.RWMutex{},
		storageFilePath: storageFilePath,
	}
}

// LoadDialogFromFile loads the dialog history from the configured storage file.
//
// It reads the file specified by storageFilePath and unmarshals its contents into the in-memory dialog history map.
// If the file does not exist, it logs a message and returns nil. If there are errors during reading or unmarshaling,
// it returns an error with details.
//
// Returns:
//   - error: An error if reading or unmarshaling the file fails; nil if the file does not exist or the operation succeeds.
func (d *AiDialogHistory) LoadDialogFromFile() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Читаем файл dialogs.json
	data, err := os.ReadFile(d.storageFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			logrus.Infof("File %s was not found", d.storageFilePath)
			return nil
		}
		return fmt.Errorf("failed to read dialog history from file %s: %w", d.storageFilePath, err)
	}

	if err = json.Unmarshal(data, &d.dialogHistory); err != nil {
		logrus.WithError(err).Error("failed to unmarshal dialog history:")
		return fmt.Errorf("failed to unmarshal dialog history: %w", err)
	}
	logrus.Infof("File %s successfully loaded", d.storageFilePath)
	return nil
}

// SaveDialog saves a dialog history for the specified chat ID.
//
// It creates a copy of the provided dialog to prevent external modifications and stores it in the in-memory map
// under the given chat ID. The operation is thread-safe due to the use of a mutex.
//
// Parameters:
//   - chatID: The ID of the chat associated with the dialog history.
//   - dialog: A slice of models.Message representing the dialog history to save.
//
// Returns:
//   - error: Always nil, as this operation does not currently fail.
func (d *AiDialogHistory) SaveDialog(chatID int64, dialog []models.Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Сохраняем копию диалога, чтобы избежать изменения внешнего среза
	dialogCopy := make([]models.Message, len(dialog))
	copy(dialogCopy, dialog)

	d.dialogHistory[chatID] = dialogCopy
	return nil
}

// SaveMsgToDialog appends a single message to the dialog history for the specified chat ID.
//
// It retrieves the current dialog history for the chat ID, appends the new message, and updates the in-memory map.
// If no history exists for the chat ID, it creates a new history. The operation is thread-safe due to the use of a mutex.
//
// Parameters:
//   - chatID: The ID of the chat associated with the dialog history.
//   - msg: The models.Message to append to the dialog history.
//
// Returns:
//   - error: Always nil, as this operation does not currently fail.
func (d *AiDialogHistory) SaveMsgToDialog(chatID int64, msg models.Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	history, exists := d.dialogHistory[chatID]
	if !exists {
		history = []models.Message{}
	}

	history = append(history, msg)
	d.dialogHistory[chatID] = history
	return nil
}

// GetDialogHistory retrieves the dialog history for the specified chat ID.
//
// It returns a copy of the dialog history to prevent external modifications. If no history exists for the chat ID,
// an empty slice is returned. The operation is thread-safe due to the use of a read-only mutex.
//
// Parameters:
//   - chatID: The ID of the chat whose dialog history is to be retrieved.
//
// Returns:
//   - []models.Message: A copy of the dialog history for the chat ID, or an empty slice if none exists.
//   - error: Always nil, as this operation does not currently fail.
func (d *AiDialogHistory) GetDialogHistory(chatID int64) ([]models.Message, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	history, exists := d.dialogHistory[chatID]
	if !exists {
		return []models.Message{}, nil
	}

	// Возвращаем копию истории, чтобы избежать изменения оригинала
	historyCopy := make([]models.Message, len(history))
	copy(historyCopy, history)
	return historyCopy, nil
}

// ClearHistory removes the dialog history for the specified chat ID.
//
// It deletes the dialog history entry associated with the chat ID from the in-memory map. The operation is
// thread-safe due to the use of a mutex.
//
// Parameters:
//   - chatID: The ID of the chat whose dialog history is to be cleared.
//
// Returns:
//   - error: Always nil, as this operation does not currently fail.
func (d *AiDialogHistory) ClearHistory(chatID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.dialogHistory, chatID)
	return nil
}

// SaveBatchToFile persists the entire dialog history to the configured storage file.
//
// It marshals the in-memory dialog history map to JSON format and writes it to the file specified by
// storageFilePath. The operation is thread-safe due to the use of a read-only mutex. It logs the time taken
// to complete the operation, and the number of user states saved.
//
// Returns:
//   - error: An error if marshaling or writing to the file fails; nil on success.
func (d *AiDialogHistory) SaveBatchToFile() error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	startTime := time.Now()

	data, err := json.MarshalIndent(d.dialogHistory, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dialog history: %w", err)
	}

	// Записываем в файл
	if err := os.WriteFile(d.storageFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write dialog history to file %s: %w", d.storageFilePath, err)
	}

	elapsedTime := time.Since(startTime)
	logrus.Infof("Saved %d user states to %s in %v", len(d.dialogHistory), d.storageFilePath, elapsedTime)
	return nil
}
