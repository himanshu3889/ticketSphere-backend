package utils

import (
	"errors"
	"strconv"

	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
)

// InitSnowflake initializes the generator with a Machine ID (e.g., 1)
// In a real distributed system, this ID comes from an env var unique to the pod/server
func InitSnowflake(machineID int64, logger *logrus.Logger) (*snowflake.Node, error) {
	snowflakeNode, err := snowflake.NewNode(machineID)
	if err != nil {
		logger.Fatalf("Failed to initialize snowflake node: %v", err)
		return nil, err
	}
	return snowflakeNode, nil
}

// GenerateID returns a new ID
func GenerateSnowflakeID(snowflakeNode *snowflake.Node) snowflake.ID {
	return snowflakeNode.Generate()
}

// isValidSnowflake checks if a string is a valid snowflake ID
func ValidSnowflakeID(id string) (snowflake.ID, error) {
	// Try to parse as uint64
	parsedID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, errors.New("Invalid ID")
	}

	// Create a snowflake ID from the parsed uint64
	snowflakeID := snowflake.ID(parsedID)

	// Validate by checking if it can extract basic components
	// Snowflake IDs should be > 0 and have reasonable timestamp
	if snowflakeID.Int64() > 0 && snowflakeID.Time() > 0 {
		return snowflakeID, nil
	}
	return 0, errors.New("Invalid ID")
}
