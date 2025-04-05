// Package service provides business logic for managing Yandex Smart Home tokens.
// It integrates with an OAuth client and a repository to retrieve and store user tokens.
package service

import (
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/server/models"
	"github.com/sirupsen/logrus"
)

// The Repository defines an interface for storing and retrieving user tokens.
// It abstracts the underlying storage mechanism.
type Repository interface {
	// SaveUserToken saves a token pair for a given user ID.
	// Arguments:
	//   - userID: the ID of the user (int64).
	//   - tokenPair: the token pair (models.Tokens) to save.
	// Returns an error if the save operation fails.
	SaveUserToken(userID int64, tokenPair models.Tokens) error
	// GetUserToken retrieves the token pair associated with a given user ID.
	// Arguments:
	//   - userID: the ID of the user (int64).
	// Returns the token pair (models.Tokens) and an error if the user ID is not found.
	GetUserToken(userID int64) (models.Tokens, error)
}

// Auth defines an interface for interacting with the Yandex OAuth API.
// This allows for mocking or swapping implementations in tests.
type Auth interface {
	// GetOAuthToken retrieves an OAuth token from Yandex using an access code.
	// Arguments:
	//   - accessCode: the authorization code received from Yandex.
	// Returns a models.ResponseAUTH containing token details or an error if the request fails.
	GetOAuthToken(accessCode string) (models.ResponseAUTH, error)
}

// Service represents the core logic for interacting with Yandex OAuth and token storage.
// It combines an OAuth client and a repository to manage token operations.
type Service struct {
	oauth      Auth       // Client for Yandex OAuth interactions.
	repository Repository // Storage for user tokens.
}

// NewService creates a new Service instance with the provided OAuth client and repository.
// Arguments:
//   - oauth: an instance of YandexAuth for OAuth operations.
//   - repository: an implementation of the Repository interface for token storage.
//
// Returns a pointer to a Service.
func NewService(oauth Auth, repository Repository) *Service {
	return &Service{
		oauth:      oauth,
		repository: repository,
	}
}

// GetYandexSmartHomeToken retrieves a token from Yandex Smart Home using an access code
// and saves it to the repository for the specified user.
// Arguments:
//   - accessCode: the authorization code received from Yandex.
//   - userID: the user's user ID (int64), used as the user identifier.
//
// Returns an error if the token retrieval or save operation fails.
func (s *Service) GetYandexSmartHomeToken(accessCode string, userID int64) error {
	if accessCode == "" {
		err := fmt.Errorf("access code cannot be empty")
		logrus.WithError(err).Error("invalid input for token retrieval")
		return err
	}

	if userID <= 0 {
		err := fmt.Errorf("userID must be a positive integer")
		logrus.WithError(err).Error("invalid userID")
		return err
	}

	res, err := s.oauth.GetOAuthToken(accessCode)
	if err != nil {
		logrus.WithError(err).Error("failed to retrieve OAuth token from Yandex")
		return fmt.Errorf("oauth token retrieval failed: %w", err)
	}

	if res.AccessToken == "" {
		err = fmt.Errorf("received empty access token from Yandex")
		logrus.WithError(err).Error("invalid OAuth response")
		return err
	}

	accessPair := models.Tokens{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		ExpiresIn:    res.ExpiresIn,
	}

	if err = s.repository.SaveUserToken(userID, accessPair); err != nil {
		logrus.WithError(err).Errorf("failed to save token for chatID: %d", userID)
		return fmt.Errorf("token save failed: %w", err)
	}

	logrus.Infof("successfully saved Yandex Smart Home token for userID: %d", userID)
	return nil
}

// GetUserToken retrieves the saved token pair for a given user ID from the repository.
// Arguments:
//   - userID: the ID of the user (int64).
//
// Returns the token pair (models.Tokens) and an error if the token is not found.
func (s *Service) GetUserToken(userID int64) (models.Tokens, error) {
	if userID <= 0 {
		err := fmt.Errorf("userID must be a positive integer")
		logrus.WithError(err).Error("invalid userID")
		return models.Tokens{}, err
	}

	tokenPair, err := s.repository.GetUserToken(userID)
	if err != nil {
		logrus.WithError(err).Infof("failed to retrieve token for userID: %d", userID)
		return models.Tokens{}, fmt.Errorf("token retrieval failed: %w", err)
	}
	logrus.Infof("successfully retrieved token for userID: %d", userID)
	return tokenPair, nil
}
