// Package api provides an implementation for interacting with the Yandex Smart Home API.
// It supports retrieving device information and performing actions like turning devices on or off.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"time"
)

// Constants for HTTP headers and API paths
const (
	UserInfoPath       = "/v1.0/user/info"
	DevicesActionsPath = "/v1.0/devices/actions"
)

// DevicesInfoResponse represents the response structure containing information about devices.
type DevicesInfoResponse struct {
	Devices []DeviceInfo `json:"devices"` // List of devices
}

// DeviceInfo describes a single device with its available actions.
type DeviceInfo struct {
	ID      string         `json:"id"`      // Unique device identifier
	Actions []DeviceAction `json:"actions"` // List of actions that can be performed
}

// DeviceAction describes a single device action.
type DeviceAction struct {
	Type  string            `json:"type"`  // Action type (e.g., "on_off", "brightness")
	State DeviceActionState `json:"state"` // Action state details
}

// DeviceActionState contains the details of an action's state.
type DeviceActionState struct {
	Instance string `json:"instance"` // Action instance (e.g., "power", "thermostat")
	Value    bool   `json:"value"`    // State value (true/false)
}

// ResponseSmart represents the authentication response from the smart home API.
type ResponseSmart struct {
	TokenType    string `json:"token_type"`    // Type of token (e.g., "Bearer").
	AccessToken  string `json:"access_token"`  // Access token for API requests
	ExpiresIn    int    `json:"expires_in"`    // Token expiration time in seconds
	RefreshToken string `json:"refresh_token"` // Refresh token for renewing access.
	Scope        string `json:"scope"`         // Scope of the token.
}

// userDeviceInfo represents the structure of user smart home data returned by the API.
type userDeviceInfo struct {
	Status    string `json:"status"`     // Request status
	RequestId string `json:"request_id"` // Unique request identifier
	Rooms     []struct {
		Id          string   `json:"id"`           // Room ID
		Name        string   `json:"name"`         // Room name
		HouseholdId string   `json:"household_id"` // Household ID
		Devices     []string `json:"devices"`      // List of device IDs in the room
	} `json:"rooms"`
	Groups  []interface{} `json:"groups"` // List of device groups
	Devices []struct {
		Id           string        `json:"id"`           // Device ID
		Name         string        `json:"name"`         // Device name
		Aliases      []interface{} `json:"aliases"`      // Device aliases
		Type         string        `json:"type"`         // Device type
		ExternalId   string        `json:"external_id"`  // External device ID
		SkillId      string        `json:"skill_id"`     // Skill ID
		HouseholdId  string        `json:"household_id"` // Household ID
		Room         string        `json:"room"`         // Room name
		Groups       []interface{} `json:"groups"`       // Device groups
		Capabilities []struct {
			Reportable  bool   `json:"reportable"`  // Whether state can be reported
			Retrievable bool   `json:"retrievable"` // Whether state can be retrieved
			Type        string `json:"type"`        // Capability type
			Parameters  struct {
				Split bool `json:"split"` // Split parameter
			} `json:"parameters"`
			State struct {
				Instance string `json:"instance"` // Capability instance
				Value    bool   `json:"value"`    // Capability state value
			} `json:"state"`
			LastUpdated float64 `json:"last_updated"` // Last update timestamp
		} `json:"capabilities"`
		Properties []interface{} `json:"properties"` // Device properties
	} `json:"devices"`
	Scenarios  []interface{} `json:"scenarios"` // List of scenarios
	Households []struct {
		Id   string `json:"id"`   // Household ID
		Name string `json:"name"` // Household name
		Type string `json:"type"` // Household type
	} `json:"households"`
}

// YandexSmartHome represents a client for interacting with the Yandex Smart Home API.
type YandexSmartHome struct {
	endpoint string // API endpoint URL.
	client   *http.Client
}

// NewYandexSmartHomeAPI creates a new instance of YandexSmartHome with the specified endpoint.
// Arguments:
//   - endpoint: the base URL of the Yandex Smart Home API.
//
// Returns a pointer to a YandexSmartHome.
func NewYandexSmartHomeAPI(endpoint string) *YandexSmartHome {
	return &YandexSmartHome{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 15 * time.Second, // Configurable in a real app
		},
	}
}

// GetHomeInfo retrieves information about the user's smart home devices.
// Arguments:
//   - token: OAuth token for authentication.
//
// Returns a map of device names to their details or an error if the request fails.
func (sh *YandexSmartHome) GetHomeInfo(token string) (map[string]*models.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sh.endpoint+UserInfoPath, nil)
	if err != nil {
		err = fmt.Errorf("failed to create request: %w", err)
		logrus.WithError(err).Error("Error creating GetHomeInfo request")
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to fetch home info from %s", req.URL)
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err = res.Body.Close(); err != nil {
			logrus.WithError(err).Errorf("Failed to close response body: %v", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected status code: %d", res.StatusCode)
		logrus.WithError(err).Errorf("GetHomeInfo failed with status: %s", res.Status)
		return nil, err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		logrus.WithError(err).Error("Failed to read GetHomeInfo response")
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	//заполняем структуру с данными об умном доме пользователя
	var response userDeviceInfo
	if err = json.Unmarshal(data, &response); err != nil {
		logrus.WithError(err).Error("Failed to unmarshal GetHomeInfo response")
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	//create user's device map for using in services
	userDevices := make(map[string]*models.Device)
	for _, device := range response.Devices {
		userDevices[device.Name] = &models.Device{
			Name:        device.Name,
			ID:          device.Id,
			ActualState: device.Capabilities[0].State.Value,
		}
	}

	logrus.Infof("Successfully retrieved home info with %d devices", len(userDevices))
	return userDevices, nil
}

// TurnOnOffAction performs an on/off action on a specified device.
// Arguments:
//   - token: OAuth token for authentication.
//   - id: device ID to perform the action on.
//   - Value: current state (true for on, false for off); the action toggles this value.
//
// Returns an error if the request fails.
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
							Value:    !value,
						},
					},
				},
			},
		},
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		err = fmt.Errorf("failed to marshal request body: %w", err)
		logrus.WithError(err).Error("Error preparing TurnOnOffAction request")
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sh.endpoint+DevicesActionsPath, bytes.NewBuffer(jsonBody))
	if err != nil {
		err = fmt.Errorf("failed to create request: %w", err)
		logrus.WithError(err).Error("Error creating TurnOnOffAction request")
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := sh.client.Do(req)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to execute TurnOnOffAction for device %s", id)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err = res.Body.Close(); err != nil {
			logrus.WithError(err).Errorf("Failed to close response body: %v", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(res.Body)
		err = fmt.Errorf("unexpected status code: %d, body: %s", res.StatusCode, string(data))
		logrus.WithError(err).Errorf("TurnOnOffAction failed with status: %s", res.Status)
		return err
	}
	return nil
}
