package inventoryStore

import (
	"errors"

	"github.com/bwmarrin/snowflake"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type Store struct {
	PostgresDB    *sqlx.DB
	SnowflakeNode *snowflake.Node
	logger        *logrus.Logger
}

func New(postgresDB *sqlx.DB, snowflakeNode *snowflake.Node, logger *logrus.Logger) (*Store, error) {
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
