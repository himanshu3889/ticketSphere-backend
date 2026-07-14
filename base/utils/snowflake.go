package utils

import (
	"errors"
	"math"
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

func ValidInt64SnowflakeID(id int64) (snowflake.ID, error) {
	// Create a snowflake ID from the int64 value
	snowflakeID := snowflake.ID(id)

	// Validate by checking if it can extract basic components
	// Snowflake IDs should be > 0 and have reasonable timestamp
	if snowflakeID.Int64() > 0 && snowflakeID.Time() > 0 {
		return snowflakeID, nil
	}
	return 0, errors.New("Invalid ID")
}

// isValidSnowflake checks if a string is a valid snowflake ID
func ValidSnowflakeID(id string) (snowflake.ID, error) {
	// Try to parse as uint64
	parsedID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, errors.New("Invalid ID")
	}

	return ValidInt64SnowflakeID(int64(parsedID))
}

// GetMaxSnowflakeID returns the absolute highest possible Snowflake ID.
func GetMaxSnowflakeID() snowflake.ID {
	return snowflake.ID(math.MaxInt64)
}
