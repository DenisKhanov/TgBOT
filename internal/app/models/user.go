package models

// TODO как лучше хранить и передавать токен пользователя через методы?
type YandexUser struct {
	Token string
	Login string
	Email string
}
