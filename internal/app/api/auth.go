package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ResponseAUTH struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}
type RequestAUTH struct {
	ClientID     string `json:"client_id"`     // идентификатор приложения
	ClientSecret string `json:"client_secret"` // секретный ключ приложения
	GrantType    string `json:"grant_type"`
	Code         string `json:"code"`
}
type YandexAuth struct {
	endpoint string
	RequestAUTH
	ResponseAUTH
}

func NewYandexAuthAPI(endpoint string) *YandexAuth {
	return &YandexAuth{endpoint: endpoint}
}

func (a *YandexAuth) AuthAPI(accessCode string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	params := url.Values{}
	params.Add("grant_type", "authorization_code")
	params.Add("code", accessCode)
	params.Add("client_id", "f78d9fab1f2b49ca9c729ec0c72964a8")
	params.Add("client_secret", "557fe2992efd46cc8d766a10072774cb")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, strings.NewReader(params.Encode()))
	if err != nil {
		err = fmt.Errorf("failed to create request with ctx: %w", err)
		logrus.Error(err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Кодируем строку client_id:client_secret методом base64
	auth := base64.StdEncoding.EncodeToString([]byte(params.Get("client_id") + ":" + params.Get("client_secret")))
	fmt.Println(auth)
	// Добавляем заголовок Authorization
	req.Header.Set("Authorization", "Basic "+auth)
	req.Form = params

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logrus.Error(err)
		return "", err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		logrus.Error(err)
		return "", err
	}
	var response ResponseAUTH
	err = json.Unmarshal(data, &response)
	if err != nil {
		return "", err
	}

	logrus.Infof("Статус-код: %s, Response body: %s ", res.Status, string(data))

	return response.AccessToken, nil
}
