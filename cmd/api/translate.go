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

type Yandex interface {
	TranslateAPI(text string) (string, error)
	DetectLangAPI(text string) (string, error)
}
type YandexAPI struct {
	token        string
	endTranslate string
	endDetect    string
	GlossaryPair
	GlossaryData
	GlossaryConfig
	TranslateRequest
	TranslateResponse
	Translation
	DetectLangReq
	DetectLangRes
}

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

type DetectLangReq struct {
	Text              string   `json:"text"`
	LanguageCodeHints []string `json:"language"`
}

type DetectLangRes struct {
	LanguageCode string `json:"languageCode"`
}

func NewYandexAPI(endTranslate, endDetect, token string) *YandexAPI {
	return &YandexAPI{
		endTranslate: endTranslate,
		endDetect:    endDetect,
		token:        token,
	}
}

func (y *YandexAPI) TranslateAPI(text string) (string, error) {

	detectedLang, err := y.DetectLangAPI(text)
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, y.endTranslate, bytes.NewBuffer(jsonBody))
	if err != nil {
		err = fmt.Errorf("failed to create request with ctx: %w", err)
		logrus.Error(err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", y.token)

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

func (y *YandexAPI) DetectLangAPI(text string) (string, error) {
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, y.endDetect, bytes.NewBuffer(jsonBody))
	if err != nil {
		err = fmt.Errorf("failed to create request with ctx: %w", err)
		logrus.Error(err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", y.token)

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
