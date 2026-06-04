package database

import (
	"context"
	"fmt"
	"time"

	"github.com/himanshu3889/ticketSphere-backend/configs"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// FIXME: USE THE PROPER LOGGER HERE AS EACH OF THE SERVICE HAS THEIR OWN SERVER
// Initialize mongoDB establishes connection and creates indexes
func InitMongoDB(appConfigs *configs.AppConfigurations, logger *logrus.Logger) (*mongo.Client, *mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Configure client
	username := appConfigs.MongoDB.Username
	password := appConfigs.MongoDB.Password
	host := appConfigs.MongoDB.Host
	port := appConfigs.MongoDB.Port
	database := appConfigs.MongoDB.Database

	// FIXME: 20217 WHY ???
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=%s",
		username, password, host, port, database, database)

	logger.Info(uri)

	clientOptions := options.Client().ApplyURI(uri).
		SetTimeout(10 * time.Second).
		SetMinPoolSize(appConfigs.MongoDB.MinPoolSize).
		SetMaxPoolSize(appConfigs.MongoDB.MaxPoolSize)

	var mongoClient *mongo.Client
	var err error
	mongoClient, err = mongo.Connect(clientOptions)
	if err != nil {
		logger.WithError(err).Fatal("MongoDB connection failed")
		return nil, nil, err
	}

	// Verify connection
	if err := mongoClient.Ping(ctx, nil); err != nil {
		logger.WithError(err).Fatal("MongoDB ping failed")
		return nil, nil, err
	}

	logger.Info("MongoDB connected successfully")

	var mongoDB *mongo.Database
	// Get database reference
	mongoDB = mongoClient.Database(database)

	// Create indexes
	logger.Info("Creating MongoDB indexes...")
	createIndexes(ctx)
	logger.Info("MongoDB indexes created successfully")
	return mongoClient, mongoDB, nil
}

// createIndexes ensures optimal query performance
// TODO: Create mongodb index using the scripts ?
func createIndexes(ctx context.Context) {
}

// DisconnectMongoDB closes the connection gracefully
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
