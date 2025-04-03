package main

import (
	"context"
	"github.com/DenisKhanov/TgBOT/internal/app/tbot"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()

	app, err := tbot.NewApp(ctx)
	if err != nil {
		logrus.Fatalf("Failed to initialize application: %v", err)
	}

	app.Run()
}
