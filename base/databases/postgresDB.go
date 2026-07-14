package database

import (
	"fmt"
	"time"

	"github.com/himanshu3889/ticketSphere-backend/configs"

	"github.com/XSAM/otelsql"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// Initialize postgres database
func InitPostgresDB(appConfigs *configs.AppConfigurations, logger *logrus.Logger) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		appConfigs.Postgres.Host,
		appConfigs.Postgres.Port,
		appConfigs.Postgres.User,
		appConfigs.Postgres.Password,
		appConfigs.Postgres.DB,
	)

	// Open the database using otelsql (injects the tracing middleware into the database driver)
	stdDB, err := otelsql.Open("postgres", dsn,
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
	)
	if err != nil {
		logger.WithError(err).Fatal("Failed to open PostgreSQL with otelsql")
		return nil, err
	}

	// Wrap the instrumented *sql.DB with sqlx
	postgresDB := sqlx.NewDb(stdDB, "postgres")

	// Verify connection
	if err := postgresDB.Ping(); err != nil {
		logger.WithError(err).Fatal("Failed to ping PostgreSQL")
		return nil, err
	}

	// Set pool limits
	postgresDB.SetMaxOpenConns(appConfigs.Postgres.MaxOpenConns)
	postgresDB.SetMaxIdleConns(appConfigs.Postgres.MaxIdleConns)
	postgresDB.SetConnMaxLifetime(time.Duration(appConfigs.Postgres.ConnMaxLifetimeMin) * time.Minute)

	// Register otelsql metrics
	_, err = otelsql.RegisterDBStatsMetrics(stdDB, otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
	))
	if err != nil {
		logger.WithError(err).Warn("Failed to register database metrics")
	}

	logger.Info("Postgres Database connected successfully (Instrumented with Otel)")
	logger.Info("Remember to run your SQL migration scripts!")

	return postgresDB, nil
}

func ClosePostgresConnection(db *sqlx.DB) error {
	return db.Close()
}
