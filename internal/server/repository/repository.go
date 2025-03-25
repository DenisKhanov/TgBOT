package repository

import (
	"errors"
	"github.com/DenisKhanov/TgBOT/internal/server/models"
	"github.com/sirupsen/logrus"
)

type Repo struct {
	usersTokens map[int]models.Tokens
}

func NewRepo() *Repo {
	return &Repo{
		usersTokens: make(map[int]models.Tokens),
	}
}

func (r *Repo) SaveTokenPair(userID int, tokenPair models.Tokens) error {

}

func (r *Repo) GetTokenPair(userID int) (models.Tokens, error) {
	accessPair, exist := r.usersTokens[userID]
	if !exist {
		err := errors.New("user not found")
		logrus.WithError(err)
		return models.Tokens{}, err
	}
	return accessPair, nil
}
