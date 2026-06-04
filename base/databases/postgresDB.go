package database

import (
	"fmt"
	"time"

	"github.com/himanshu3889/ticketSphere-backend/configs"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Initialize postgres database
func InitPostgresDB(appConfigs *configs.AppConfigurations, logger *logrus.Logger) (*sqlx.DB, error) {
	var postgresDB *sqlx.DB

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		appConfigs.Postgres.Host,
		appConfigs.Postgres.Port,
		appConfigs.Postgres.User,
		appConfigs.Postgres.Password,
		appConfigs.Postgres.DB,
	)

	var err error
	postgresDB, err = sqlx.Connect("postgres", dsn)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to PostgreSQL")
		return nil, err
	}

	// Verify connection
	if err := postgresDB.Ping(); err != nil {
		logger.WithError(err).Fatal("Failed to ping PostgreSQL")
		return nil, err
	}

	// Set pool limits (adjust as needed)
	postgresDB.SetMaxOpenConns(appConfigs.Postgres.MaxOpenConns)
	postgresDB.SetMaxIdleConns(appConfigs.Postgres.MaxIdleConns)
	postgresDB.SetConnMaxLifetime(time.Duration(appConfigs.Postgres.ConnMaxLifetimeMin) * time.Minute)

	logger.Info("Postgres Database connected successfully")

	// You must run migrations manually or use a tool like golang-migrate
	logger.Info("Remember to run your SQL migration scripts!")

	return postgresDB, nil
}

func ClosePostgresConnection(db *sqlx.DB) error {
	err := db.Close()
	return err
}
