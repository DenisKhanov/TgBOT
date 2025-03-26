package repository

import (
	"errors"
	"github.com/DenisKhanov/TgBOT/internal/server/models"
	"github.com/sirupsen/logrus"
)

type Repository struct {
	userToken map[int64]models.Tokens
}

func NewRepository() *Repository {
	return &Repository{
		userToken: make(map[int64]models.Tokens),
	}
}

func (r *Repository) SaveUserToken(userID int64, tokenPair models.Tokens) error {
	r.userToken[userID] = tokenPair
	_, exist := r.userToken[userID]
	if !exist {
		err := errors.New("user's token can't save")
		logrus.WithError(err)
		return err
	}
	return nil
}

func (r *Repository) GetUserToken(userID int64) (models.Tokens, error) {
	tokenPair, exist := r.userToken[userID]
	if !exist {
		err := errors.New("userID not found")
		logrus.WithError(err)
		return models.Tokens{}, err
	}
	return tokenPair, nil
}
