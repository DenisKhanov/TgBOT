package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"time"
)

type DetectLangReq struct {
	Text              string   `json:"text"`
	LanguageCodeHints []string `json:"language"`
}

type DetectLangRes struct {
	LanguageCode string `json:"languageCode"`
}

func DetectLangAPI(text string) (string, error) {
	endpoint := "https://translate.api.cloud.yandex.net/translate/v2/detect"
	token, err := os.ReadFile("translateToken.txt")
	if err != nil {
		logrus.Error(err)
	}

	reqBody := DetectLangReq{
		Text:              text,
		LanguageCodeHints: []string{"ru", "en"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		err = fmt.Errorf("error marshalling JSON: %v", err)
		logrus.Error(err)
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		err = fmt.Errorf("failed to create request with ctx: %w", err)
		logrus.Error(err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", string(token))

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
	fmt.Println(string(data))
	var response DetectLangRes
	err = json.Unmarshal(data, &response)
	if err != nil {
		logrus.Error(err)
		return "", err
	}

	logrus.Info("Статус-код ", res.Status)
	logrus.Info("Определен язык это - ", response.LanguageCode)

	return response.LanguageCode, nil
}
