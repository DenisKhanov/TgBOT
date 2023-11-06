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

type GlossaryPair struct {
	SourceText     string `json:"sourceText"`
	TranslatedText string `json:"translatedText"`
}

// GlossaryData holds an array of GlossaryPair objects.
type GlossaryData struct {
	GlossaryPairs []GlossaryPair `json:"glossaryPairs"`
}

// GlossaryConfig contains GlossaryData.
type GlossaryConfig struct {
	GlossaryData GlossaryData `json:"glossaryData"`
}

// TranslateRequest is the top-level structure for the translation request.
type TranslateRequest struct {
	SourceLanguageCode string         `json:"sourceLanguageCode"`
	TargetLanguageCode string         `json:"targetLanguageCode"`
	Format             string         `json:"format"`
	Texts              []string       `json:"texts"`
	FolderId           string         `json:"folderId"`
	Model              string         `json:"model"`
	GlossaryConfig     GlossaryConfig `json:"glossaryConfig"`
	Speller            bool           `json:"speller"`
}

type Translation struct {
	Text                 string `json:"text"`
	DetectedLanguageCode string `json:"detectedLanguageCode"`
}
type TranslateResponse struct {
	Translations []Translation `json:"translations"`
}

func TranslateAPI(text string) (string, error) {
	endpoint := "https://translate.api.cloud.yandex.net/translate/v2/translate"
	token, err := os.ReadFile("translateToken.txt")
	if err != nil {
		logrus.Error(err)
	}

	detectedLang, err := DetectLangAPI(text)
	if err != nil {
		logrus.Error("Ошибка при определении языка: ", err)
		return "", err
	}
	targetLang := "ru"
	if detectedLang == "ru" {
		targetLang = "en"
	}
	reqBody := TranslateRequest{
		SourceLanguageCode: detectedLang,
		TargetLanguageCode: targetLang,
		Format:             "PLAIN_TEXT",
		Texts:              []string{text},
		Speller:            true,
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
	var response TranslateResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		logrus.Error(err)
		return "", err
	}

	logrus.Info("Статус-код ", res.Status)
	logrus.Info("Переведенный текст - ", response.Translations[0])

	result := response.Translations[0]

	return result.Text, nil
}
