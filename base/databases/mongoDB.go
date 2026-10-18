package database

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/himanshu3889/ticketSphere-backend/configs"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"
)

// InitMongoDB connects to MongoDB with the otel monitor attached.
func InitMongoDB(mongoCfg *configs.MongoConfig, logger *logrus.Logger) (*mongo.Client, *mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Configure client
	username := mongoCfg.Username
	password := mongoCfg.Password
	host := mongoCfg.Host
	port := mongoCfg.Port
	database := mongoCfg.Database

	creds := url.UserPassword(username, password)
	uri := fmt.Sprintf("mongodb://%s@%s:%s/%s?authSource=%s",
		creds.String(), host, port, database, database)

	// Attach the OpenTelemetry Monitor to the client options
	clientOptions := options.Client().
		ApplyURI(uri).
		SetTimeout(10 * time.Second).
		SetMinPoolSize(mongoCfg.MinPoolSize).
		SetMaxPoolSize(mongoCfg.MaxPoolSize).
		SetMonitor(otelmongo.NewMonitor()) // <--- This is the magic line

	var mongoClient *mongo.Client
	var err error
	mongoClient, err = mongo.Connect(clientOptions)
	if err != nil {
		logger.WithFields(logrus.Fields{"URI": uri}).WithError(err).Fatal("MongoDB connection failed")
		return nil, nil, err
	}

	// Verify connection
	if err := mongoClient.Ping(ctx, nil); err != nil {
		logger.WithError(err).Fatal("MongoDB ping failed")
		return nil, nil, err
	}

	logger.Info("MongoDB connected successfully (Instrumented with Otel)")

	// Get database reference
	mongoDB := mongoClient.Database(database)

	return mongoClient, mongoDB, nil
}

// CloseMongoDBConnection disconnects the client; nil-safe.
func CloseMongoDBConnection(mongoClient *mongo.Client, logger *logrus.Logger) {
	if mongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := mongoClient.Disconnect(ctx); err != nil {
			logger.WithError(err).Error("Error disconnecting from MongoDB")
		} else {
			logger.Info("MongoDB disconnected")
		}
	}
}
