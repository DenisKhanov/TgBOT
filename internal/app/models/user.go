package models

type UserState struct {
	ChatID            int64              `json:"chatID"`            // Идентификатор чата
	CurrentStep       string             `json:"currentStep"`       // Текущий этап диалога с пользователем
	LastUserMessages  string             `json:"lastUserMessages"`  // Данные, введённые пользователем, ключ - название данных
	CallbackQueryData string             `json:"callbackQueryData"` // Данные из callback-запросов, если они используются
	IsTranslating     bool               `json:"isTranslating"`     // Флаг состояния перевода для пользователя
	Token             string             `json:"token"`             // Токен сервиса яндекс smarthome
	Devices           map[string]*Device `json:"devices"`           // Карта устройств пользователя
}

type Device struct {
	Name  string
	ID    string
	State bool
}
