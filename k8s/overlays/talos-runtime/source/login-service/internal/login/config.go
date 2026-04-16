package login

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

type Config struct {
	Port              string
	AppEnv            string
	LogLevel          string
	MongoHost         string
	MongoPort         int
	MongoDatabase     string
	MongoCollection   string
	MongoUsername     string
	MongoPassword     string
	MongoAuthDatabase string
	MongoTimeoutMs    int
}

func LoadConfig() Config {
	return Config{
		Port:              envOr("PORT", "3000"),
		AppEnv:            envOr("APP_ENV", "production"),
		LogLevel:          envOr("LOG_LEVEL", "INFO"),
		MongoHost:         envOr("MONGODB_HOST", "192.168.0.100"),
		MongoPort:         envInt("MONGODB_PORT", 32017),
		MongoDatabase:     envOr("MONGODB_DATABASE", "geek_hub"),
		MongoCollection:   envOr("MONGODB_COLLECTION", "user_login_events"),
		MongoUsername:     envOr("MONGODB_USERNAME", ""),
		MongoPassword:     envOr("MONGODB_PASSWORD", ""),
		MongoAuthDatabase: envOr("MONGODB_AUTH_DATABASE", "admin"),
		MongoTimeoutMs:    envInt("MONGODB_CONNECT_TIMEOUT_MS", 2500),
	}
}

func (c Config) MongoURI() string {
	if c.MongoUsername == "" || c.MongoPassword == "" {
		return fmt.Sprintf("mongodb://%s:%d", c.MongoHost, c.MongoPort)
	}

	return fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/?authSource=%s",
		url.QueryEscape(c.MongoUsername),
		url.QueryEscape(c.MongoPassword),
		c.MongoHost,
		c.MongoPort,
		url.QueryEscape(c.MongoAuthDatabase),
	)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
