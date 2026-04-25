SHELL := /bin/bash

SERVER_DIR := cmd/server
BOT_DIR := cmd/tgbot

SERVER_BIN := server
BOT_BIN := tgbot

GO := go
GOFLAGS := -v

SERVER_ENV := server.env
BOT_ENV := bot.env

SERVER_LOG := server.log
BOT_LOG := bot.log

SERVER_CERT_GEN := pkg/tls_config/cert/server
BOT_CERT_GEN := pkg/tls_config/cert/client
SAN_CNF := $(SERVER_CERT_GEN)/san.cnf
SAN_CNF_TEMPLATE := $(SERVER_CERT_GEN)/san.cnf.example
TOKEN_STORAGE := server_tokens.json

DOCKER_COMPOSE := docker compose
DOCKER_COMPOSE_FILE := docker-compose.yml

SYSTEMD_SERVER_ENV := /etc/systemd/system/server.env
SYSTEMD_BOT_ENV := /etc/systemd/system/tgbot.env

.DEFAULT_GOAL := help

.PHONY: help all build build-server build-bot deps fmt test clean \
	gen-certs ensure-san-cnf sync-san-cnf run run-server run-bot start-server start-bot \
	stop stop-server stop-bot prepare-runtime-files deploy-container \
	stop-container logs-container start-systemd stop-systemd restart-systemd

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*## "; printf "\nAvailable targets:\n\n"} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-18s %s\n", $$1, $$2} END {printf "\n"}' $(MAKEFILE_LIST)

all: build ## Build both binaries

build: build-server build-bot ## Build server and bot

build-server: ## Build OAuth server binary
	$(GO) build $(GOFLAGS) -o $(SERVER_BIN) ./$(SERVER_DIR)

build-bot: ## Build Telegram bot binary
	$(GO) build $(GOFLAGS) -o $(BOT_BIN) ./$(BOT_DIR)

deps: ## Run go mod tidy
	$(GO) mod tidy

fmt: ## Format Go code
	$(GO) fmt ./...

test: ## Run all tests
	$(GO) test ./... -v

clean: ## Remove binaries and logs
	rm -f $(SERVER_BIN) $(BOT_BIN) $(SERVER_LOG) $(BOT_LOG)

prepare-runtime-files: ## Create missing runtime state and log files
	@touch $(SERVER_LOG) $(BOT_LOG) $(TOKEN_STORAGE) keep_chat.json dialog_ai.json

ensure-san-cnf: ## Create san.cnf from template if it is missing
	@if [ ! -f $(SAN_CNF) ]; then \
		cp $(SAN_CNF_TEMPLATE) $(SAN_CNF); \
		echo "Created $(SAN_CNF) from $(SAN_CNF_TEMPLATE)"; \
	fi

sync-san-cnf: ensure-san-cnf ## Sync CN and SAN values in san.cnf from bot.env and server.env
	@set -eu; \
	bot_host=$$(awk -F= '/^SERVER_ENDPOINT=/{print $$2}' $(BOT_ENV) | sed -E 's#^https?://##; s#/.*##; s#:[0-9]+$$##'); \
	server_host=$$(awk -F= '/^HTTPS_SERVER=/{print $$2}' $(SERVER_ENV) | sed -E 's#/.*##; s#:[0-9]+$$##'); \
	if [ -z "$$bot_host" ]; then \
		echo "SERVER_ENDPOINT not found in $(BOT_ENV)"; \
		exit 1; \
	fi; \
	if [ -z "$$server_host" ]; then \
		echo "HTTPS_SERVER not found in $(SERVER_ENV)"; \
		exit 1; \
	fi; \
	cert_host="$$bot_host"; \
	case "$$server_host" in \
		0.0.0.0|127.0.0.1|localhost|::) ;; \
		*) if [ "$$bot_host" != "$$server_host" ]; then \
		echo "Host mismatch between $(BOT_ENV) ($$bot_host) and $(SERVER_ENV) ($$server_host)"; \
		exit 1; \
		fi ;; \
	esac; \
	if printf '%s' "$$cert_host" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		ip_entry="IP.1 = $$cert_host"; \
		dns_entry="DNS.1 = localhost"; \
	else \
		ip_entry="IP.1 = 127.0.0.1"; \
		dns_entry="DNS.1 = $$cert_host"; \
	fi; \
	sed -E -i \
		-e "s#^CN = .*#CN = $$cert_host#" \
		-e "s#^IP\\.1 = .*#$$ip_entry#" \
		-e "s#^DNS\\.1 = .*#$$dns_entry#" \
		$(SAN_CNF); \
	echo "Updated $(SAN_CNF) for host $$cert_host"

gen-certs: sync-san-cnf ## Generate TLS certificates for server and bot
	@echo "Generating certificates for server in $(SERVER_CERT_GEN)..."
	cd $(SERVER_CERT_GEN) && sh gen.sh
	@echo "Generating certificates for bot in $(BOT_CERT_GEN)..."
	cd $(BOT_CERT_GEN) && sh gen.sh

run: gen-certs build ## Generate certs, build and start server and bot
	@$(MAKE) --no-print-directory start-server
	@$(MAKE) --no-print-directory start-bot
	@sleep 2 && rm -f $(SERVER_ENV) $(BOT_ENV) && echo "Environment files removed"

run-server: gen-certs build-server ## Build and start only the OAuth server
	@$(MAKE) --no-print-directory start-server

run-bot: build-bot ## Build and start only the Telegram bot
	@$(MAKE) --no-print-directory start-bot

start-server:
	@echo "Starting server..."
	@./$(SERVER_BIN) > $(SERVER_LOG) 2>&1 || (echo "Server failed to start. Check $(SERVER_LOG) for errors."; exit 1) &

start-bot:
	@echo "Starting bot..."
	@./$(BOT_BIN) > $(BOT_LOG) 2>&1 || (echo "Bot failed to start. Check $(BOT_LOG) for errors."; exit 1) &

stop: stop-server stop-bot ## Stop server and bot processes

stop-server: ## Stop only the OAuth server
	-pkill -f $(SERVER_BIN)

stop-bot: ## Stop only the Telegram bot
	-pkill -f $(BOT_BIN)

deploy-container: gen-certs prepare-runtime-files ## Build and start project in Docker containers
	$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) up -d --build

stop-container: ## Stop Docker containers
	$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) down

logs-container: ## Tail Docker container logs
	$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) logs -f

start-systemd: gen-certs build ## Deploy through systemd
	@echo "Copying environment files to systemd..."
	@sudo cp $(SERVER_ENV) $(SYSTEMD_SERVER_ENV)
	@sudo cp $(BOT_ENV) $(SYSTEMD_BOT_ENV)
	@sudo systemctl start server
	@sudo systemctl start tgbot
	@sleep 2 && sudo rm -f $(SYSTEMD_SERVER_ENV) $(SYSTEMD_BOT_ENV)
	@echo "Environment files removed from systemd directory"

stop-systemd: ## Stop systemd services
	@sudo systemctl stop server
	@sudo systemctl stop tgbot

restart-systemd: stop-systemd start-systemd ## Restart systemd services
