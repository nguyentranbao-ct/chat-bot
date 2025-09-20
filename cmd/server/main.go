package main

import (
	"context"
	"log"

	"github.com/nguyentranbao-ct/chat-bot/internal/app"
)

func main() {
	log.Println("Starting chat-bot service...")

	application := app.NewApp()
	if err := application.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	<-application.Done()
	log.Println("Application stopped")
}
