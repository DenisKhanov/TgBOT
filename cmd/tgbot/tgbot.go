package main

import (
	"context"
	"github.com/DenisKhanov/TgBOT/internal/app/tbot"
	"github.com/sirupsen/logrus"
)

func main() {
	// Create a new application context
	ctx := context.Background()

	// Initialize the application
	app, err := tbot.NewApp(ctx)
	if err != nil {
		logrus.Fatalf("Failed to initialize application: %v", err)
	}

	// Run the application
	app.Run()
}
