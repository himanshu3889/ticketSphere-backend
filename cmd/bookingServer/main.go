package main

import (
	"fmt"

	"github.com/himanshu3889/ticketSphere-backend/configs"
	"github.com/himanshu3889/ticketSphere-backend/logger"
	"github.com/joho/godotenv"
)

func main() {
	// Load the envs
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No .env file found, relying on system environment variables")
	}

	serverYmlFilePath := "configs/booking_server_config.yml"

	cfg, err := configs.InitializeConfigs(serverYmlFilePath)
	if err != nil {
		fmt.Printf("failed to load config: %v", err)
	}

	customLogger := logger.New(cfg)

	server, err := NewServer(cfg, customLogger)
	if err != nil {
		customLogger.Fatalf("failed to create booking server: %v", err)
	}

	server.Run()
}
