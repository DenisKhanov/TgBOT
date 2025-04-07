# TgBOT - Многофункциональный Telegram бот

![Telegram Bot](https://img.shields.io/badge/Telegram-Bot-blue?logo=telegram)
![Go](https://img.shields.io/badge/Go-1.16+-00ADD8?logo=go)

*Read this in [English](#tgbot---multifunctional-telegram-bot)*

## 📝 Описание

TgBOT - это многофункциональный Telegram бот, написанный на языке Go, который предоставляет пользователям различные
возможности:

- 🏠 **Управление умным домом** - интеграция с Яндекс Умным домом для управления устройствами
- 🌐 **Перевод текста** - перевод сообщений с использованием Яндекс API
- 🎯 **Рекомендации по активностям** - предложения, чем заняться, когда скучно
- 🎯 **Показать ссылку на подборку отличных фильмов** - ссылка на мой сайт со списком фильмов по категориям.

Бот использует современную архитектуру с применением паттерна Dependency Injection для управления зависимостями и
чистого кода для лучшей поддержки и расширяемости.

P.s.- этот проект был написан мной для личного удобства использования моих умных устройств и может быть не сильно
универсален.

## 🚀 Возможности

- **Интеграция с Яндекс Умным домом**
    - Авторизация через OAuth
    - Получение списка устройств
    - Управление устройствами (включение/выключение)

- **Перевод текста**
    - Автоматическое определение языка
    - Перевод текста с использованием Яндекс API

- **Рекомендации по активностям**
    - Предложения интересных занятий
    - Разнообразные категории активностей

## 🛠️ Технологии

- **Go** - основной язык программирования
- **Telegram Bot API** - для взаимодействия с Telegram
- **Яндекс API** - для перевода и управления умным домом
- **TLS** - для защиты соединений
- **Logrus** - для логирования

## 📋 Требования

- Go 1.16 или выше
- Доступ к интернету
- Токен Telegram бота
- Токены Яндекс API (для перевода и умного дома)

## 📥 Установка

1. Клонируйте репозиторий:
   ```bash
   git clone https://github.com/DenisKhanov/TgBOT.git
   cd TgBOT
   ```

2. Настройте генерацию TLS сертификата:
   ```
   В файле /pkg/tls_config/cert/server/san.cnf 
   отредактируйте строку 
   ```
   `[alt_names]`

   `IP.1 = 176.108.251.250 #it's ip address your server`
   ```
   Впишите сюда внешний ip адрес своего сервера,
   чтобы сгенирировались верные сертификаты
   ```

## ⚙️ Конфигурация

Перед запуском необходимо настроить конфигурацию:

1. Создайте файл `server.env` со следующим содержимым:
   ```
   LOG_LEVEL=info (уровень_логирования)
   LOG_FILE_NAME=Server.log (имя_файла_для_сохранения_логов)
   OAUTH_ENDPOINT=https://oauth.yandex.ru/token (путь_для_получения_токена_умного_дома)
   HTTPS_SERVER=localhost:8080 (адрес_на_котором_запускается_сервер)
   SERVER_CERT_FILE=/pkg/tls_config/cert/server/server.crt (путь_TLS_к_ключам_и_сертификатам)
   SERVER_KEY_FILE=/pkg/tls_config/cert/server/server.key (путь_TLS_к_ключам_и_сертификатам)
   SERVER_CA_FILE=/pkg/tls_config/cert/server/ca.crt (путь_TLS_к_ключам_и_сертификатам)
   CLIENT_ID=_______________________ (ID_внешнего_приложения_умного_дома https://oauth.yandex.ru/client) 
   CLIENT_SECRET=________________________ (secret_внешнего_приложения_умного_дома https://oauth.yandex.ru/client) 
   API_KEY=__________________________ (ключ_для_упрощенной_блокировки_несанкционированного_доступа_к_серверу)
   ```

2. Создайте файл `bot.env` со следующим содержимым:
   ```
   LOG_LEVEL=info (уровень_логирования)
   LOG_FILE_NAME=Bot.log (имя_файла_для_сохранения_логов)
   FILE_STORAGE_PATH=./keep_chat.json (файл_куда_сохраняется_история_состояния_чата)
   TOKEN_BOT=________:_____________________ (токен_ТГ_бота)
   TOKEN_YANDEX=Api-Key ____________________________ (токен_яндекс_переводчика)
   SERVER_ENDPOINT="https://localhost:8080" (эндпоинт_для_запроса_токена_умного_дома_у_сервера)
   CLIENT_ID=_______________________ (ID_внешнего_приложения_умного_дома https://oauth.yandex.ru/client)
   OWNER_ID=_______________________ (ID_владельца_бота_для_доступа_к_умному_дому)
   CLIENT_CERT_FILE=/pkg/tls_config/cert/client/client.crt (путь_TLS_к_ключам_и_сертификатам)
   CLIENT_KEY_FILE=/pkg/tls_config/cert/client/client.key (путь_TLS_к_ключам_и_сертификатам)
   CLIENT_CA_FILE=/pkg/tls_config/cert/server/ca.crt (путь_TLS_к_ключам_и_сертификатам)
   API_KEY=___________________________ (ключ_для_упрощенной_блокировки_несанкционированного_доступа_к_серверу)
   ```

## 🚀 Запуск

### Обычный запуск

```bash
make run
```

### Запуск через systemd

```bash
make start-systemd
```

### Остановка

```bash
make stop
```

или

```bash
make stop-systemd
```

## 🧪 Тестирование

```bash
make test
```

## 📁 Структура проекта

```
TgBOT/
├── cmd/                      # Точки входа в приложение
│   ├── server/               # Сервер для OAuth авторизации
│   └── tgbot/                # Telegram бот
├── internal/                 # Внутренний код приложения
│   ├── app/                  # Инициализация и запуск приложений
│   ├── logcfg/               # Конфигурация логирования
│   ├── server/               # Компоненты сервера
│   └── tg_bot/               # Компоненты Telegram бота
├── pkg/                      # Публичные пакеты
│   └── tls_config/           # Конфигурация TLS
└── makefile                  # Файл сборки проекта
```

## 🤝 Вклад в проект

Вклады приветствуются! Пожалуйста, следуйте этим шагам:

1. Форкните репозиторий
2. Создайте ветку для вашей функции (`git checkout -b feature/amazing-feature`)
3. Зафиксируйте ваши изменения (`git commit -m 'Add some amazing feature'`)
4. Отправьте изменения в ветку (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

## 📄 Лицензия

## 📞 Контакты

Имя - [Denis Khanov](https://t.me/DenKhan)

Ссылка на проект: [https://github.com/DenisKhanov/TgBOT](https://github.com/DenisKhanov/TgBOT)

---

# TgBOT - Multifunctional Telegram Bot

![Telegram Bot](https://img.shields.io/badge/Telegram-Bot-blue?logo=telegram)
![Go](https://img.shields.io/badge/Go-1.16+-00ADD8?logo=go)

*Читать на [русском языке](#tgbot---многофункциональный-telegram-бот)*

## 📝 Description

TgBOT2 is a multifunctional Telegram bot written in Go that provides users with various capabilities:

- 🏠 **Smart Home Control** - integration with Yandex Smart Home to control devices
- 🌐 **Text Translation** - message translation using Yandex API
- 🎯 **Activity Recommendations** - suggestions on what to do when bored
- 🎯 **Show a link to a selection of great movies** - a link to my website with a list of movies by category.

The bot uses a modern architecture with the Dependency Injection pattern for managing dependencies and clean code for
better maintainability and extensibility.

## 🚀 Features

- **Yandex Smart Home Integration**
    - OAuth authorization
    - Device listing
    - Device control (on/off)

- **Text Translation**
    - Automatic language detection
    - Text translation using Yandex API

- **Activity Recommendations**
    - Interesting activity suggestions
    - Various activity categories

## 🛠️ Technologies

- **Go** - main programming language
- **Telegram Bot API** - for Telegram interaction
- **Yandex API** - for translation and smart home control
- **TLS** - for secure connections
- **Logrus** - for logging

## 📋 Requirements

- Go 1.16 or higher
- Internet access
- Telegram bot token
- Yandex API tokens (for translation and smart home)

## 📥 Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/DenisKhanov/TgBOT.git
   cd TgBOT
   ```

2. Configure TLS certificate generation:
   ```
    In the /pkg/tls_config/cert/server/san.cnf file 
   edit the line
   ```
   `[alt_names]`

   `IP.1 = 176.108.251.250 #it's ip address your server`
   ```
   Enter the external ip address of your server here,
   to generate the correct certificates
   ```

## ⚙️ Configuration

Before running, you need to configure:

1. Create a `server.env` file with the following content:
   ```
   LOG_LEVEL=info (уровень_логирования)
   LOG_FILE_NAME=Server.log (имя_файла_для_сохранения_логов)
   OAUTH_ENDPOINT=https://oauth.yandex.ru/token (путь_для_получения_токена_умного_дома)
   HTTPS_SERVER=localhost:8080 (адрес_на_котором_запускается_сервер)
   SERVER_CERT_FILE=/pkg/tls_config/cert/server/server.crt (путь_TLS_к_ключам_и_сертификатам)
   SERVER_KEY_FILE=/pkg/tls_config/cert/server/server.key (путь_TLS_к_ключам_и_сертификатам)
   SERVER_CA_FILE=/pkg/tls_config/cert/server/ca.crt (путь_TLS_к_ключам_и_сертификатам)
   CLIENT_ID=_______________________ (ID_внешнего_приложения_умного_дома https://oauth.yandex.ru/client) 
   CLIENT_SECRET=________________________ (secret_внешнего_приложения_умного_дома https://oauth.yandex.ru/client) 
   API_KEY=__________________________ (ключ_для_упрощенной_блокировки_несанкционированного_доступа_к_серверу)
   ```

2. Create a `bot.env` file with the following content:
   ```
   LOG_LEVEL=info (уровень_логирования)
   LOG_FILE_NAME=Bot.log (имя_файла_для_сохранения_логов)
   FILE_STORAGE_PATH=./keep_chat.json (файл_куда_сохраняется_история_состояния_чата)
   TOKEN_BOT=________:_____________________ (токен_ТГ_бота)
   TOKEN_YANDEX=Api-Key ____________________________ (токен_яндекс_переводчика)
   SERVER_ENDPOINT="https://localhost:8080" (эндпоинт_для_запроса_токена_умного_дома_у_сервера)
   CLIENT_ID=_______________________ (ID_внешнего_приложения_умного_дома https://oauth.yandex.ru/client)
   OWNER_ID=_______________________ (ID_владельца_бота_для_доступа_к_умному_дому)
   CLIENT_CERT_FILE=/pkg/tls_config/cert/client/client.crt (путь_TLS_к_ключам_и_сертификатам)
   CLIENT_KEY_FILE=/pkg/tls_config/cert/client/client.key (путь_TLS_к_ключам_и_сертификатам)
   CLIENT_CA_FILE=/pkg/tls_config/cert/server/ca.crt (путь_TLS_к_ключам_и_сертификатам)
   API_KEY=___________________________ (ключ_для_упрощенной_блокировки_несанкционированного_доступа_к_серверу)
   ```

## 🚀 Running

### Normal Run

```bash
make run
```

### Run via systemd

```bash
make start-systemd
```

### Stopping

```bash
make stop
```

or

```bash
make stop-systemd
```

## 🧪 Testing

```bash
make test
```

## 📁 Project Structure

```
TgBOT/
├── cmd/                      # Application entry points
│   ├── server/               # Server for OAuth authorization
│   └── tgbot/                # Telegram bot
├── internal/                 # Internal application code
│   ├── app/                  # Application initialization and running
│   ├── logcfg/               # Logging configuration
│   ├── server/               # Server components
│   └── tg_bot/               # Telegram bot components
├── pkg/                      # Public packages
│   └── tls_config/           # TLS configuration
└── makefile                  # Project build file
```

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

## 📞 Contact

Your Name - [Denis Khanov](https://t.me/DenKhan)

Project Link: [https://github.com/DenisKhanov/TgBOT](https://github.com/DenisKhanov/TgBOT)
