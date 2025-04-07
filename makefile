# Переменные
SERVER_DIR = cmd/server
BOT_DIR = cmd/tgbot
SERVER_BIN = server
BOT_BIN = tgbot
GO = go
GOFLAGS = -v
SERVER_ENV = server.env
BOT_ENV = bot.env
SERVER_CERT_GEN = pkg/tls_config/cert/server
BOT_CERT_GEN = pkg/tls_config/cert/client

# Цель по умолчанию
.PHONY: all
all: build

# Сборка обоих приложений
.PHONY: build
build: deps build-server build-bot


# Сборка сервера
.PHONY: build-server
build-server:
	$(GO) build $(GOFLAGS) -o $(SERVER_BIN) ./$(SERVER_DIR)

# Сборка бота
.PHONY: build-bot
build-bot:
	$(GO) build $(GOFLAGS) -o $(BOT_BIN) ./$(BOT_DIR)

# Генерация сертификатов
.PHONY: gen-certs
gen-certs:
	 @echo "Generating certificates for server in $(SERVER_CERT_GEN)..."
	cd $(SERVER_CERT_GEN) && chmod +x gen.sh && ./gen.sh

	@echo "Generating certificates for bot in $(BOT_CERT_GEN)..."
	cd $(BOT_CERT_GEN) && chmod +x gen.sh && ./gen.sh


# Запуск сервера и бота
.PHONY: run
run: gen-certs build
	@echo "Starting server and bot..."
	@./$(SERVER_BIN) > server.log 2>&1 || (echo "Server failed to start. Check server.log for errors."; exit 1) & \
        ./$(BOT_BIN) > bot.log 2>&1 || (echo "Bot failed to start. Check bot.log for errors."; exit 1) & \
    (sleep 2 && rm -f $(SERVER_ENV) $(BOT_ENV) && echo "Environment files removed")

# Остановка запущенных процессов
.PHONY: stop
stop:
	-pkill -f $(SERVER_BIN)
	-pkill -f $(BOT_BIN)

# Очистка бинарных файлов
.PHONY: clean
clean:
	rm -f $(SERVER_BIN) $(BOT_BIN)
	rm -f server.log bot.log

# Установка зависимостей
.PHONY: deps
deps:
	$(GO) mod tidy

# Форматирование кода
.PHONY: fmt
fmt:
	$(GO) fmt ./...

# Тестирование
.PHONY: test
test:
	$(GO) test ./... -v

# Запуск через systemd
.PHONY: start-systemd
start-systemd: gen-certs build
	@echo "Copying environment files to systemd..."
	@sudo cp $(SERVER_ENV) /etc/systemd/system/server.env
	@sudo cp $(BOT_ENV) /etc/systemd/system/tgbot.env
	@sudo systemctl start server
	@sudo systemctl start tgbot
	@sleep 2 && sudo rm -f /etc/systemd/system/server.env /etc/systemd/system/tgbot.env
	@echo "Environment files removed from systemd directory"

# Остановка через systemd
.PHONY: stop-systemd
stop-systemd:
	@sudo systemctl stop server
	@sudo systemctl stop tgbot

# Перезапуск через systemd
.PHONY: restart-systemd
restart-systemd: stop-systemd start-systemd