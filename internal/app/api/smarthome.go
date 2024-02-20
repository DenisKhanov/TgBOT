package api

import (
	"GoProgects/PetProjects/internal/app/models"
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

type userDeviceInfo struct {
	Status    string `json:"status"`
	RequestId string `json:"request_id"`
	Rooms     []struct {
		Id          string   `json:"id"`
		Name        string   `json:"name"`
		HouseholdId string   `json:"household_id"`
		Devices     []string `json:"devices"`
	} `json:"rooms"`
	Groups  []interface{} `json:"groups"`
	Devices []struct {
		Id           string        `json:"id"`
		Name         string        `json:"name"`
		Aliases      []interface{} `json:"aliases"`
		Type         string        `json:"type"`
		ExternalId   string        `json:"external_id"`
		SkillId      string        `json:"skill_id"`
		HouseholdId  string        `json:"household_id"`
		Room         string        `json:"room"`
		Groups       []interface{} `json:"groups"`
		Capabilities []struct {
			Reportable  bool   `json:"reportable"`
			Retrievable bool   `json:"retrievable"`
			Type        string `json:"type"`
			Parameters  struct {
				Split bool `json:"split"`
			} `json:"parameters"`
			State struct {
				Instance string `json:"instance"`
				Value    bool   `json:"value"`
			} `json:"state"`
			LastUpdated float64 `json:"last_updated"`
		} `json:"capabilities"`
		Properties []interface{} `json:"properties"`
	} `json:"devices"`
	Scenarios  []interface{} `json:"scenarios"`
	Households []struct {
		Id   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"households"`
}

func NewYandexSmartHomeAPI(endpoint string) *YandexSmartHome {
	return &YandexSmartHome{endpoint: endpoint}
}

func (sh *YandexSmartHome) GetHomeInfo(token string) (map[string]*models.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
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

	//заполняем структуру с данными об умном доме пользователя
	var response userDeviceInfo
	err = json.Unmarshal(data, &response)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	//создаем карту с устройствами пользователя, для дальнейшего их использования
	userDevices := make(map[string]*models.Device)
	for _, v := range response.Devices {
		userDevices[v.Name] = &models.Device{
			Name:  v.Name,
			ID:    v.Id,
			State: v.Capabilities[0].State.Value,
		}
	}

	logrus.Infof("Статус-код: %s, Response body: %s ", res.Status, string(data))

	return userDevices, nil
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
							Value:    !value}}}}},
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
