package elasticsearch

import (
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"gitlab.com/xakpro/cg-shared-libs/logger"
	"go.uber.org/zap"
)

// Config holds Elasticsearch connection configuration
type Config struct {
	Addresses []string `yaml:"addresses" env:"ELASTICSEARCH_ADDRESSES" env-default:"http://localhost:9200"`
	Username  string   `yaml:"username" env:"ELASTICSEARCH_USERNAME"`
	Password  string   `yaml:"password" env:"ELASTICSEARCH_PASSWORD"`
	CloudID   string   `yaml:"cloud_id" env:"ELASTICSEARCH_CLOUD_ID"`
	APIKey    string   `yaml:"api_key" env:"ELASTICSEARCH_API_KEY"`
}

// New creates a new Elasticsearch client
func New(cfg Config) (*elasticsearch.Client, error) {
	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		CloudID:   cfg.CloudID,
		APIKey:    cfg.APIKey,
	}

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("create elasticsearch client: %w", err)
	}

	// Test connection
	res, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("elasticsearch info: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch error: %s", res.String())
	}

	logger.Info("Elasticsearch connected",
		zap.Strings("addresses", cfg.Addresses),
	)

	return client, nil
}
