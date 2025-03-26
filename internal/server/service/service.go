package service

import (
	"github.com/DenisKhanov/TgBOT/internal/server/models"
)

type Repository interface {
	SaveUserToken(userID int64, tokenPair models.Tokens) error
	GetUserToken(userID int64) (models.Tokens, error)
}
type Service struct {
	oauth      YandexAuth
	repository Repository
}

func NewService(oauth YandexAuth, repository Repository) *Service {
	return &Service{
		oauth:      oauth,
		repository: repository,
	}
}

func (s *Service) GetYandexSmartHomeToken(accessCode string, chatID int64) error {

	res, err := s.oauth.GetOAuthToken(accessCode)
	if err != nil {
		return err
	}
	accessPair := models.Tokens{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		ExpiresIn:    res.ExpiresIn,
	}

	if err = s.repository.SaveUserToken(chatID, accessPair); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetUserToken(userID int64) (models.Tokens, error) {
	tokenPair, err := s.repository.GetUserToken(userID)
	if err != nil {
		return models.Tokens{}, err
	}
	return tokenPair, nil
}
