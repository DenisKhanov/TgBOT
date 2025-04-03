// Package repository provides a simple in-memory storage solution for managing user tokens.
// It implements basic CRUD operations for token pairs associated with user IDs.
package repository

import (
	"errors"
	"github.com/DenisKhanov/TgBOT/internal/server/models"
	"github.com/sirupsen/logrus"
)

// Repository represents an in-memory storage for user tokens.
// It uses a map to associate user IDs with their respective token pairs.
type Repository struct {
	userToken map[int64]models.Tokens // Map storing user IDs and their tokens.
}

// NewRepository creates a new instance of Repository with an initialized token map.
// Returns a pointer to the Repository.
func NewRepository() *Repository {
	return &Repository{
		userToken: make(map[int64]models.Tokens),
	}
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

	r.userToken[userID] = tokenPair
	logrus.Infof("successfully saved token for userID: %d", userID)
	return nil
}

// GetUserToken retrieves the token pair associated with a given user ID.
// Arguments:
//   - userID: the ID of the user (int64).
//
// Returns the token pair (models.Tokens) and an error if the user ID is not found.
func (r *Repository) GetUserToken(userID int64) (models.Tokens, error) {
	tokenPair, exist := r.userToken[userID]
	if !exist {
		err := errors.New("userID not found in repository")
		logrus.WithError(err).Info("token retrieval failed")
		return models.Tokens{}, err
	}
	return tokenPair, nil
}
