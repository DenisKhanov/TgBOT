// Package repository provides a simple in-memory storage solution for managing user tokens.
// It implements basic CRUD operations for token pairs associated with user IDs.
package repository

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/server/models"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
)

// The Repository represents an in-memory storage for user tokens.
// It uses a map to associate user IDs with their respective token pairs.
type Repository struct {
	userToken map[int64]models.Tokens // Map storing user IDs and their tokens.
	filePath  string
	mu        sync.RWMutex
}

// NewRepository creates a new instance of Repository with an initialized token map.
// Returns a pointer to the Repository.
func NewRepository(filePath string) (*Repository, error) {
	repo := &Repository{
		userToken: make(map[int64]models.Tokens),
		filePath:  filePath,
	}
	if err := repo.loadFromFile(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *Repository) loadFromFile() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			logrus.Infof("Token storage file %s does not exist, starting with empty repository", r.filePath)
			return nil
		}
		return fmt.Errorf("failed to read token storage %s: %w", r.filePath, err)
	}

	if len(data) == 0 {
		logrus.Infof("Token storage file %s is empty, starting with empty repository", r.filePath)
		return nil
	}

	var tokens map[int64]models.Tokens
	if err = json.Unmarshal(data, &tokens); err != nil {
		return fmt.Errorf("failed to unmarshal token storage %s: %w", r.filePath, err)
	}

	r.userToken = tokens
	logrus.Infof("Loaded %d user tokens from %s", len(r.userToken), r.filePath)
	return nil
}

func (r *Repository) saveToFile() error {
	tempPath := r.filePath + ".tmp"
	file, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open temp token storage %s: %w", tempPath, err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			logrus.WithError(err).Error("failed to close token storage file")
		}
	}()

	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(r.userToken); err != nil {
		return fmt.Errorf("failed to encode token storage %s: %w", tempPath, err)
	}
	if err = writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush token storage %s: %w", tempPath, err)
	}
	if err = os.Rename(tempPath, r.filePath); err != nil {
		return fmt.Errorf("failed to rename token storage %s to %s: %w", tempPath, r.filePath, err)
	}
	return nil
}

// SaveUserToken saves a token pair for a given user ID in the repository.
// Arguments:
//   - userID: the ID of the user (int64).
//   - tokenPair: the token pair (models.Tokens) to be saved.
//
// Returns an error if the token cannot be saved.
func (r *Repository) SaveUserToken(userID int64, tokenPair models.Tokens) error {
	if tokenPair.AccessToken == "" || tokenPair.RefreshToken == "" {
		err := errors.New("invalid token pair: access or refresh token is empty")
		logrus.WithError(err).Error("failed to save user token")
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.userToken[userID] = tokenPair
	if err := r.saveToFile(); err != nil {
		logrus.WithError(err).Error("failed to persist user token")
		return err
	}
	logrus.Infof("successfully saved token for userID: %d", userID)
	return nil
}

// GetUserToken retrieves the token pair associated with a given user ID.
// Arguments:
//   - userID: the ID of the user (int64).
//
// Returns the token pair (models.Tokens) and an error if the user ID is not found.
func (r *Repository) GetUserToken(userID int64) (models.Tokens, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tokenPair, exist := r.userToken[userID]
	if !exist {
		err := errors.New("userID not found in repository")
		logrus.WithError(err).Info("token retrieval failed")
		return models.Tokens{}, err
	}
	return tokenPair, nil
}
