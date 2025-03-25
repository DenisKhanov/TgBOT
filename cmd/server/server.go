package main

import (
	"context"
	"github.com/DenisKhanov/TgBOT/internal/app/server"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()
	a, err := server.NewApp(ctx)
	if err != nil {
		logrus.Fatalf("failed to init app: %s", err.Error())
	}
	a.Run()
}
