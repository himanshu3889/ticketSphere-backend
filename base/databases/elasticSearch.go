package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/himanshu3889/ticketSphere-backend/configs"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/sirupsen/logrus"
)

func ConnectElasticsearch(appConfigs *configs.AppConfigurations, logger *logrus.Logger) (*elasticsearch.Client, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{
			fmt.Sprintf("http://%s:%s",
				appConfigs.Elasticsearch.Host,
				appConfigs.Elasticsearch.Port),
		},
		Username:      appConfigs.Elasticsearch.Username,
		Password:      appConfigs.Elasticsearch.Password,
		RetryOnStatus: []int{502, 503, 504, 429},
		MaxRetries:    5,
		RetryBackoff: func(i int) time.Duration {
			return time.Duration(1<<i) * 100 * time.Millisecond
		},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		logger.WithError(err).Fatal("Elasticsearch client creation failed")
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := client.Info(client.Info.WithContext(ctx))
	if err != nil {
		logger.WithError(err).Fatal("Elasticsearch connection failed")
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		logger.Fatalf("Elasticsearch error: %s", res.String())
		return nil, errors.New(res.String())
	}

	logger.Info("Elasticsearch connected successfully")
	return client, nil
}

// Helper methods
func GetIndex(ctx context.Context, indexName string, esClient *elasticsearch.Client) (*esapi.Response, error) {
	return esClient.Indices.Get([]string{indexName},
		esClient.Indices.Get.WithContext(ctx),
	)
}

func CreateIndex(ctx context.Context, indexName string, mapping string, esClient *elasticsearch.Client) error {
	res, err := esClient.Indices.Create(indexName,
		esClient.Indices.Create.WithBody(strings.NewReader(mapping)),
		esClient.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch error: %s", res.String())
	}
	return nil
}
