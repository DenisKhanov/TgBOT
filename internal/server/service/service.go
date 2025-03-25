package service

import "github.com/DenisKhanov/TgBOT/internal/server/models"

type Repository interface {
	SaveTokenPair(userID int, tokenPair models.Tokens) error
	GetTokenPair(userID int) (models.Tokens, error)
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

func (s *Service) GetYandexSmartHomeToken(accessCode string, chatID int) error {

	res, err := s.oauth.GetOAuthToken(accessCode)
	if err != nil {
		return err
	}
	accessPair := models.Tokens{AccessToken: res.AccessToken, RefreshToken: res.RefreshToken}

	if err = s.repository.SaveTokenPair(chatID, accessPair); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetUserToken(userID int) (accessToken string, err error) {
	tokenPair, err := s.repository.GetTokenPair(userID)
	if err != nil {
		return "", err
	}
	return tokenPair.AccessToken, nil
}
