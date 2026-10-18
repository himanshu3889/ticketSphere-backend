package main

import (
	"fmt"

	"github.com/himanshu3889/ticketSphere-backend/configs"
	"github.com/himanshu3889/ticketSphere-backend/logger"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No .env file found, relying on system environment variables")
	}

	cfg, err := configs.Load[configs.BookingConfig]("configs/booking.yml")
	if err != nil {
		fmt.Printf("failed to load config: %v", err)
	}

	customLogger := logger.New(cfg.App, cfg.Server, cfg.Loki)

	server, err := NewServer(cfg, customLogger)
	if err != nil {
		customLogger.Fatalf("failed to create booking server: %v", err)
	}

	server.Run()
}
