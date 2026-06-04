package baseCDC

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func SetupConnector(kafkaConnectURL, configFilePath string, logger *logrus.Logger) error {
	const (
		maxRetries = 5
		baseDelay  = 8 * time.Second
		maxDelay   = 30 * time.Second
	)

	configData, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	connectorName, err := extractConnectorName(configData)
	if err != nil {
		return fmt.Errorf("failed to extract connector name: %w", err)
	}

	// Execution closure for the actual work
	operation := func() error {
		logger.WithFields(logrus.Fields{
			"connector": connectorName,
			"url":       kafkaConnectURL,
		}).Info("Attempting Debezium connector setup")

		if connectorExists(kafkaConnectURL, connectorName, logger) {
			return updateConnector(kafkaConnectURL, connectorName, configData, logger)
		}
		return createConnector(kafkaConnectURL, configData, logger)
	}

	// Retry Loop
	for i := 0; i < maxRetries; i++ {
		err := operation()
		if err == nil {
			logger.WithField("connector", connectorName).Info("Debezium connector synchronized successfully")
			return checkStatus(kafkaConnectURL, connectorName, logger)
		}

		// Fail fast if it's NOT a connection issue
		if !strings.Contains(err.Error(), "connection refused") {
			return fmt.Errorf("permanent failure during setup: %w", err)
		}

		if i == maxRetries-1 {
			return fmt.Errorf("Debezium connector unavailable after %d attempts: %w", maxRetries, err)
		}

		// Calculate Backoff: base = min(baseDelay * 2^i, maxDelay)
		base := min(baseDelay*time.Duration(1<<uint(i-1)), maxDelay) // base = min(baseDelay * 2^i, 30000)
		half := base / 2
		jitter := time.Duration(rand.Intn(int(half)))
		delay := half + jitter

		logger.WithError(err).Warnf("Debezium not ready (attempt %d/%d), retrying in %v...", i+1, maxRetries, delay)
		time.Sleep(delay)
	}

	return nil
}

// Extract the connector name from the json
func extractConnectorName(configData []byte) (string, error) {
	var config struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(configData, &config); err != nil {
		return "", err
	}
	if config.Name == "" {
		return "", fmt.Errorf("Debezium connector name not found in JSON")
	}
	return config.Name, nil
}

// Does connector exist ?
func connectorExists(kafkaConnectURL, name string, logger *logrus.Logger) bool {
	resp, err := http.Get(fmt.Sprintf("%s/connectors/%s", kafkaConnectURL, name))
	if err != nil {
		logger.WithError(err).Debug("Failed to check debezium connector existence")
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Create a new connector
func createConnector(kafkaConnectURL string, configData []byte, logger *logrus.Logger) error {
	url := fmt.Sprintf("%s/connectors", kafkaConnectURL)

	resp, err := http.Post(url, "application/json", bytes.NewReader(configData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		logger.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("Failed to create debezium connector")
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

// Update the connector
func updateConnector(kafkaConnectURL, connectorName string, configData []byte, logger *logrus.Logger) error {
	// Parse to get just the config part
	var fullConfig map[string]interface{}
	if err := json.Unmarshal(configData, &fullConfig); err != nil {
		return err
	}

	config, ok := fullConfig["config"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid config format")
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/connectors/%s/config", kafkaConnectURL, connectorName)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(configJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("Failed to update debezium connector")
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

// Get the connector status and verify tasks are running
func checkStatus(kafkaConnectURL, connectorName string, logger *logrus.Logger) error {
	url := fmt.Sprintf("%s/connectors/%s/status", kafkaConnectURL, connectorName)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Parse the JSON status
	var status struct {
		Connector struct {
			State string `json:"state"`
		} `json:"connector"`
		Tasks []struct {
			State string `json:"state"`
			Trace string `json:"trace"`
		} `json:"tasks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return fmt.Errorf("failed to decode status JSON: %w", err)
	}

	// Check if the connector itself failed
	if status.Connector.State == "FAILED" {
		return fmt.Errorf("Debezium connector %s is in FAILED state", connectorName)
	}

	// Check if any internal tasks failed
	for _, task := range status.Tasks {
		if task.State == "FAILED" {
			logger.Errorf("Debezium Task Failed Trace: %s", task.Trace)
			return fmt.Errorf("task for %s failed", connectorName)
		}
	}

	logger.WithField("connector", connectorName).Info("Debezium connector and all tasks are RUNNING")
	return nil
}
