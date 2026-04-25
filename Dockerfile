FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server && \
	CGO_ENABLED=0 GOOS=linux go build -o /out/tgbot ./cmd/tgbot

FROM alpine:3.21 AS base

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY success.html ./success.html
COPY --from=builder /out/server ./server
COPY --from=builder /out/tgbot ./tgbot

RUN mkdir -p /app/pkg/tls_config/cert/server /app/pkg/tls_config/cert/client

FROM base AS server

CMD ["./server"]

FROM base AS bot

CMD ["./tgbot"]
