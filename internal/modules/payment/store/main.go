package paymentStore

import (
	"errors"

	"github.com/bwmarrin/snowflake"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/singleflight"
)

// Store struct holds the database connections and other dependencies.
type Store struct {
	PostgresDB    *sqlx.DB
	SnowflakeNode *snowflake.Node
	logger        *logrus.Logger
	RequestGroup  singleflight.Group
}

// New initializes the store with the given dependencies.
func New(postgresDB *sqlx.DB,
	snowflakeNode *snowflake.Node,
	logger *logrus.Logger) (*Store, error) {
	if postgresDB == nil {
		return nil, errors.New("Postgres DB is nil")
	}
	if snowflakeNode == nil {
		return nil, errors.New("Snowflake Node is nil")
	}
	if logger == nil {
		return nil, errors.New("Logger is nil")
	}
	return &Store{PostgresDB: postgresDB, SnowflakeNode: snowflakeNode, logger: logger}, nil
}
