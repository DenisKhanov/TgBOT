package models

type UserState struct {
	ChatID            int64              `json:"chatID"`            // Идентификатор чата
	CurrentStep       string             `json:"currentStep"`       // Текущий этап диалога с пользователем
	LastUserMessages  string             `json:"lastUserMessages"`  // Данные, введённые пользователем, ключ - название данных
	CallbackQueryData string             `json:"callbackQueryData"` // Данные из callback-запросов, если они используются
	IsTranslating     bool               `json:"isTranslating"`     // Флаг состояния перевода для пользователя
	IsGenerative      bool               `json:"isGenerative"`      // Флаг состояния режима ИИ для пользователя
	Token             string             `json:"-"`                 // Токен сервиса яндекс smartphone. Не записывается в файл
	Devices           map[string]*Device `json:"devices"`           // Карта устройств пользователя
}

type Device struct {
	Name        string
	ID          string
	ActualState bool
}
