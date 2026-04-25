# TgBOT

![Telegram Bot](https://img.shields.io/badge/Telegram-Bot-blue?logo=telegram)
![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)

*Read this in [English](#english)*

## Русский

### Что это

`TgBOT` - это проект из двух Go-приложений:

- `cmd/tgbot` - Telegram-бот
- `cmd/server` - локальный HTTPS OAuth-сервер для авторизации в Яндекс Умном доме

Бот умеет:

- управлять устройствами Яндекс Умного дома
- переводить текст через Yandex Translate API
- отвечать через generative providers: `gemini`, `deepseek`, `openrouter`
- показывать ссылку на внешний каталог фильмов
- хранить состояние пользователей и историю AI-диалогов в JSON

Сервер и бот общаются по mutual TLS. Сертификаты лежат в `pkg/tls_config/cert/`.

### Требования

- Go `1.25+` для локальной сборки
- `make`
- Docker и Docker Compose, если нужен контейнерный запуск
- Telegram bot token
- Yandex OAuth client id / secret
- Yandex Translate API key
- generative provider API key

### Быстрый старт

```bash
git clone https://github.com/DenisKhanov/TgBOT.git
cd TgBOT
cp bot.env.example bot.env
cp server.env.example server.env
```

После этого заполни реальные значения в `bot.env` и `server.env`.

`san.cnf` руками править не нужно. `make gen-certs`, `make run` и `make deploy-container`
автоматически создают `pkg/tls_config/cert/server/san.cnf` из
`pkg/tls_config/cert/server/san.cnf.example` и синхронизируют `CN` / `SAN` с
`SERVER_ENDPOINT` и `HTTPS_SERVER`.

### Конфигурация

Основные переменные в `bot.env`:

- `TOKEN_BOT` - токен Telegram-бота
- `SERVER_ENDPOINT` - HTTPS endpoint OAuth-сервера, например `https://example.com:9443`
- `CLIENT_ID` - Yandex OAuth client id
- `OWNER_ID` - Telegram user id владельца
- `TRANSLATE_API_KEY` - ключ Yandex Translate
- `GENERATIVE_NAME` - `gemini`, `deepseek` или `openrouter`
- `GENERATIVE_API_KEY` - API key выбранного провайдера
- `GENERATIVE_MODEL` - имя модели провайдера
- `MOVIES_URL` - внешняя ссылка на каталог фильмов
- `CLIENT_CERT_FILE`, `CLIENT_KEY_FILE`, `CLIENT_CA_FILE` - пути к mTLS сертификатам
- `API_KEY` - общий ключ для запросов к локальному серверу

Основные переменные в `server.env`:

- `HTTPS_SERVER` - адрес HTTPS-сервера, обычно `0.0.0.0:9443`
- `OAUTH_ENDPOINT` - endpoint обмена OAuth code на token
- `CLIENT_ID`, `CLIENT_SECRET` - данные Yandex OAuth приложения
- `SERVER_CERT_FILE`, `SERVER_KEY_FILE`, `SERVER_CA_FILE` - TLS-файлы сервера
- `API_KEY` - общий ключ, который использует бот

Готовые шаблоны:

- `bot.env.example`
- `server.env.example`

### Команды Makefile

```bash
make help
```

Основные команды:

- `make build` - собрать оба бинаря
- `make build-server` - собрать только OAuth-сервер
- `make build-bot` - собрать только Telegram-бота
- `make gen-certs` - сгенерировать TLS-сертификаты
- `make run` - сгенерировать сертификаты, собрать и запустить оба процесса
- `make run-server` - сгенерировать сертификаты, собрать и запустить только сервер
- `make run-bot` - собрать и запустить только бота
- `make stop` - остановить оба процесса
- `make deploy-container` - собрать и поднять проект в контейнерах
- `make logs-container` - смотреть логи контейнеров
- `make stop-container` - остановить контейнеры
- `make start-systemd` - старт через systemd
- `make stop-systemd` - остановка systemd-сервисов
- `make restart-systemd` - рестарт systemd-сервисов

### Локальный запуск

```bash
make gen-certs
make run-server
make run-bot
```

Если нужен единый запуск одной командой:

```bash
make run
```

Замечание: `make run` после старта удаляет `bot.env` и `server.env`. Для VPS и долгоживущего
запуска обычно удобнее `make run-server` / `make run-bot` или контейнеры.

### Контейнерный запуск

```bash
make deploy-container
make logs-container
make stop-container
```

Контейнерный запуск использует `network_mode: host`, поэтому он рассчитан на Linux-host.

### OAuth для Яндекс

Для рабочего сценария авторизации проверь:

- `SERVER_ENDPOINT` в `bot.env` совпадает с внешним адресом сервера
- `HTTPS_SERVER` в `server.env` использует тот же host или `0.0.0.0:9443`
- в Yandex OAuth `redirect_uri` равен `https://<your-host>:9443/callback`

### Хранимые runtime-файлы

Проект может создавать и обновлять:

- `keep_chat.json`
- `dialog_ai.json`
- `server_tokens.json`
- `Bot.log`
- `Server.log`
- `pkg/tls_config/cert/server/san.cnf`

### Тесты и форматирование

```bash
make fmt
make test
```

### Структура проекта

```text
TgBOT/
├── cmd/
│   ├── server/
│   └── tgbot/
├── internal/
│   ├── app/
│   ├── logcfg/
│   ├── server/
│   └── tg_bot/
├── pkg/
│   └── tls_config/
├── bot.env.example
├── server.env.example
├── Dockerfile
├── docker-compose.yml
└── makefile
```

### Контакты

- Telegram: [Denis Khanov](https://t.me/DenKhan)
- Repository: [github.com/DenisKhanov/TgBOT](https://github.com/DenisKhanov/TgBOT)

---

## English

### Overview

`TgBOT` consists of two Go applications:

- `cmd/tgbot` - Telegram bot
- `cmd/server` - local HTTPS OAuth server used for Yandex Smart Home authorization

The bot supports:

- Yandex Smart Home device control
- text translation via Yandex Translate API
- generative replies through `gemini`, `deepseek`, or `openrouter`
- external movies catalog link
- JSON-backed user state and AI dialog history

The bot and server communicate over mutual TLS. Certificates live in `pkg/tls_config/cert/`.

### Requirements

- Go `1.25+` for local builds
- `make`
- Docker and Docker Compose for container deployment
- Telegram bot token
- Yandex OAuth client id / secret
- Yandex Translate API key
- generative provider API key

### Quick start

```bash
git clone https://github.com/DenisKhanov/TgBOT.git
cd TgBOT
cp bot.env.example bot.env
cp server.env.example server.env
```

Fill in real values in `bot.env` and `server.env`.

You do not need to edit `san.cnf` manually. `make gen-certs`, `make run`, and
`make deploy-container` automatically create `pkg/tls_config/cert/server/san.cnf` from
`pkg/tls_config/cert/server/san.cnf.example` and sync `CN` / `SAN` with
`SERVER_ENDPOINT` and `HTTPS_SERVER`.

### Configuration

Important variables in `bot.env`:

- `TOKEN_BOT` - Telegram bot token
- `SERVER_ENDPOINT` - OAuth server endpoint, for example `https://example.com:9443`
- `CLIENT_ID` - Yandex OAuth client id
- `OWNER_ID` - Telegram user id for owner-only actions
- `TRANSLATE_API_KEY` - Yandex Translate key
- `GENERATIVE_NAME` - `gemini`, `deepseek`, or `openrouter`
- `GENERATIVE_API_KEY` - API key for the selected provider
- `GENERATIVE_MODEL` - provider model name
- `MOVIES_URL` - external movies catalog URL
- `CLIENT_CERT_FILE`, `CLIENT_KEY_FILE`, `CLIENT_CA_FILE` - mTLS certificate paths
- `API_KEY` - shared key used when the bot talks to the local server

Important variables in `server.env`:

- `HTTPS_SERVER` - HTTPS bind address, usually `0.0.0.0:9443`
- `OAUTH_ENDPOINT` - token exchange endpoint
- `CLIENT_ID`, `CLIENT_SECRET` - Yandex OAuth application credentials
- `SERVER_CERT_FILE`, `SERVER_KEY_FILE`, `SERVER_CA_FILE` - TLS file paths
- `API_KEY` - shared key required by the bot

Reference files:

- `bot.env.example`
- `server.env.example`

### Make targets

```bash
make help
```

Common commands:

- `make build`
- `make build-server`
- `make build-bot`
- `make gen-certs`
- `make run`
- `make run-server`
- `make run-bot`
- `make stop`
- `make deploy-container`
- `make logs-container`
- `make stop-container`
- `make start-systemd`
- `make stop-systemd`
- `make restart-systemd`

### Local run

```bash
make gen-certs
make run-server
make run-bot
```

For one-command startup:

```bash
make run
```

Note: `make run` deletes `bot.env` and `server.env` after startup. For VPS deployment,
`make run-server` / `make run-bot` or containers are usually more practical.

### Container deployment

```bash
make deploy-container
make logs-container
make stop-container
```

Container deployment uses `network_mode: host`, so it is intended for Linux hosts.

### Yandex OAuth checklist

For the OAuth flow to work correctly:

- `SERVER_ENDPOINT` in `bot.env` must match the public server address
- `HTTPS_SERVER` in `server.env` should use the same host or `0.0.0.0:9443`
- the Yandex OAuth `redirect_uri` must be `https://<your-host>:9443/callback`

### Runtime files

The project may create and update:

- `keep_chat.json`
- `dialog_ai.json`
- `server_tokens.json`
- `Bot.log`
- `Server.log`
- `pkg/tls_config/cert/server/san.cnf`

### Tests and formatting

```bash
make fmt
make test
```

### Project layout

```text
TgBOT/
├── cmd/
│   ├── server/
│   └── tgbot/
├── internal/
│   ├── app/
│   ├── logcfg/
│   ├── server/
│   └── tg_bot/
├── pkg/
│   └── tls_config/
├── bot.env.example
├── server.env.example
├── Dockerfile
├── docker-compose.yml
└── makefile
```

### Contacts

- Telegram: [Denis Khanov](https://t.me/DenKhan)
- Repository: [github.com/DenisKhanov/TgBOT](https://github.com/DenisKhanov/TgBOT)
