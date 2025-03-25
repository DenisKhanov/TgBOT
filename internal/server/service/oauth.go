package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/server/models"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type YandexAuth struct {
	endpoint     string
	clientID     string
	clientSecret string
}

func NewYandexAuthAPI(endpoint, clientID, clientSecret string) YandexAuth {
	return YandexAuth{
		endpoint:     endpoint,
		clientID:     clientID,
		clientSecret: clientSecret}
}

func (a *YandexAuth) GetOAuthToken(accessCode string) (models.ResponseAUTH, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	params := url.Values{}
	params.Add("grant_type", "authorization_code")
	params.Add("code", accessCode)
	params.Add("client_id", a.clientID)
	params.Add("client_secret", a.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, strings.NewReader(params.Encode()))
	if err != nil {
		err = fmt.Errorf("failed to create request with ctx: %w", err)
		return models.ResponseAUTH{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Кодируем строку client_id:client_secret методом base64
	auth := base64.StdEncoding.EncodeToString([]byte(params.Get("client_id") + ":" + params.Get("client_secret")))
	// Добавляем заголовок Authorization
	req.Header.Set("Authorization", "Basic "+auth)
	req.Form = params

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logrus.Error(err)
		return models.ResponseAUTH{}, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return models.ResponseAUTH{}, err
	}
	var response models.ResponseAUTH
	err = json.Unmarshal(data, &response)
	if err != nil {
		return models.ResponseAUTH{}, err
	}

	return response, nil
}
