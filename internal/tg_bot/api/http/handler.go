package http

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Handler struct {
	client      *http.Client // HTTP-клиент с настроенным TLS
	srvEndpoint string       //URL адрес сервера
}

func NewHandler(srvEndpoint, cert, key, ca string) *Handler {
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
	}
}

func (h *Handler) GetYaSmartHomeToken(accessCode, clientID, clientSecret string) (models.ResponseOAuth, error) {
	fmt.Printf("Sending request to: %s\n", h.srvEndpoint) // Отладка
	// Данные для отправки
	data := map[string]string{
		"access_code":   accessCode,
		"client_id":     clientID,
		"client_secret": clientSecret,
	}

	// Кодируем данные в JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return models.ResponseOAuth{}, fmt.Errorf("error marshal JSON: %w", err)
	}
	// URL вашего сервера
	url := h.srvEndpoint
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return models.ResponseOAuth{}, fmt.Errorf("error creating request: %w", err)
	}

	fmt.Println(req.Header)
	fmt.Println(req.Body)
	fmt.Println(req.Method)
	fmt.Println(req)

	req.Header.Set("Content-Type", "application/json") // Добавляем заголовок

	resp, err := h.client.Do(req)
	if err != nil {
		return models.ResponseOAuth{}, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.ResponseOAuth{}, fmt.Errorf("error reading response: %w", err)
	}

	fmt.Printf("Response body: %s\n", string(body)) // Отладка

	var resToken models.ResponseOAuth

	if err = json.Unmarshal(body, &resToken); err != nil {
		return models.ResponseOAuth{}, fmt.Errorf("error unmarshal response: %w", err)
	}

	return resToken, nil
}
