package http

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	client      *http.Client // HTTP-клиент с настроенным TLS
	srvEndpoint string       //URL адрес сервера
	apiKey      string
}

func NewHandler(srvEndpoint, cert, key, ca, apiKey string) *Handler {
	// Проверяем, что endpoint использует HTTPS
	if !strings.HasPrefix(srvEndpoint, "https://") {
		panic(fmt.Errorf("server endpoint must start with 'https://': %s", srvEndpoint))
	}
	wd, err := os.Getwd()
	if err != nil {
		panic(fmt.Errorf("could not get working directory: %w", err))
	}
	// Загружаем сертификат клиента и ключ
	clientCert, err := tls.LoadX509KeyPair(wd+cert, wd+key)
	if err != nil {
		panic(fmt.Errorf("error loading client certificate: %w", err))
	}

	// Загружаем корневой сертификат (CA)
	caCert, err := os.ReadFile(wd + ca)
	if err != nil {
		panic(fmt.Errorf("error loading CA certificate: %w", err))
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		panic(fmt.Errorf("failed to append CA certificate to pool"))
	}

	// Настраиваем TLS
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{clientCert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: false,
	}

	// Создаем транспорт и клиент
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	return &Handler{
		client:      client,
		srvEndpoint: srvEndpoint,
		apiKey:      apiKey,
	}
}

func (h *Handler) GetUserToken(chatID int64) (models.ResponseOAuth, error) {
	strChatID := strconv.Itoa(int(chatID))
	req, err := http.NewRequest("GET", h.srvEndpoint+"?state="+strChatID, nil)
	if err != nil {
		return models.ResponseOAuth{}, err
	}
	req.Header.Set("X-API-Key", h.apiKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return models.ResponseOAuth{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.ResponseOAuth{}, fmt.Errorf("server returned: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.ResponseOAuth{}, err
	}
	var tokenPair models.ResponseOAuth
	err = json.Unmarshal(data, &tokenPair)
	fmt.Println("TOKEN PAIR")
	fmt.Println(tokenPair)
	return tokenPair, nil
}
