package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"time"
)

type DeviceActionState struct {
	Instance string `json:"instance"`
	Value    bool   `json:"value"`
}

type DeviceAction struct {
	Type  string            `json:"type"`
	State DeviceActionState `json:"state"`
}

type DeviceInfo struct {
	ID      string         `json:"id"`
	Actions []DeviceAction `json:"actions"`
}

type DevicesInfoResponse struct {
	Devices []DeviceInfo `json:"devices"`
}

type ResponseSmart struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type YandexSmartHome struct {
	endpoint string
	DevicesInfoResponse
	ResponseSmart
}

func NewYandexSmartHomeAPI(endpoint string) *YandexSmartHome {
	return &YandexSmartHome{endpoint: endpoint}
}

func (sh *YandexSmartHome) GetHomeInfo(token string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	fmt.Println("Token!!!", token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sh.endpoint+"/v1.0/user/info", nil)
	if err != nil {
		err = fmt.Errorf("failed to create request with ctx: %w", err)
		logrus.Error(err)
		return nil, err
	}
	// Добавляем заголовок Authorization
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	//TODO пока не произвожу анмаршалинг тела ответа, но над этим стоит подумать
	//var response ResponseAUTH
	//err = json.Unmarshal(data, &response)
	//if err != nil {
	//	return "", err
	//}

	logrus.Infof("Статус-код: %s, Response body: %s ", res.Status, string(data))

	return data, nil
}

func (sh *YandexSmartHome) TurnOnOffAction(token, id string, value bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	reqBody := DevicesInfoResponse{
		Devices: []DeviceInfo{
			{ID: id,
				Actions: []DeviceAction{
					{Type: "devices.capabilities.on_off",
						State: DeviceActionState{
							Instance: "on",
							Value:    value}}}}},
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		err = fmt.Errorf("error marshalling JSON: %v", err)
		logrus.Error(err)
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sh.endpoint+"/v1.0/devices/actions", bytes.NewBuffer(jsonBody))
	if err != nil {
		err = fmt.Errorf("failed to create request with ctx: %w", err)
		logrus.Error(err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logrus.Error(err)
		return err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		logrus.Error(err)
		return err
	}

	logrus.Infof("Статус-код: %s, Response body: %s ", res.Status, string(data))
	if res.StatusCode == http.StatusOK {
		return nil
	}
	return fmt.Errorf("error:%s", res.Status)
}
