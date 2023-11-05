package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type ResponseAPI struct {
	Activity      string  `json:"activity"`
	Accessibility float64 `json:"accessibility"`
	Type          string  `json:"type"`
	Participants  int     `json:"participants"`
	Price         float64 `json:"price"`
	Link          string  `json:"link"`
	Key           string  `json:"key"`
}

func BoredAPI() (string, error) {
	endpoint := "http://www.boredapi.com/api/activity/"

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		err = fmt.Errorf("failed to create request with ctx: %w", err)
		logrus.Error(err)
		return "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logrus.Error(err)
		return "", err
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Print(err)
		return "", err
	}
	var response ResponseAPI
	err = json.Unmarshal(data, &response)
	if err != nil {
		return "", err
	}

	logrus.Info("Статус-код ", res.Status)
	fmt.Println(response.Activity)
	ruActivity, errTrans := TranslateAPI(response.Activity)
	if errTrans != nil {
		return "", errTrans
	}

	return ruActivity, nil
}
