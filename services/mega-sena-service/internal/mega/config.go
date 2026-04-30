package mega

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

type Config struct {
	Port                      string
	LogLevel                  string
	MongoHost                 string
	MongoPort                 int
	MongoDatabase             string
	MongoMegaSenaCollection   string
	MongoSimulationCollection string
	MongoCounterCollection    string
	MongoUsername             string
	MongoPassword             string
	MongoAuthDatabase         string
	MongoTimeoutMs            int
}

func LoadConfig() Config {
	return Config{
		Port:                      envOr("PORT", "3000"),
		LogLevel:                  envOr("LOG_LEVEL", "INFO"),
		MongoHost:                 envOr("MONGODB_HOST", "192.168.0.100"),
		MongoPort:                 envInt("MONGODB_PORT", 32017),
		MongoDatabase:             envOr("MONGODB_DATABASE", "geek_hub"),
		MongoMegaSenaCollection:   envOr("MONGODB_MEGA_SENA_COLLECTION", "mega_sena_resultados"),
		MongoSimulationCollection: envOr("MONGODB_MEGA_SENA_SIMULATION_COLLECTION", "mega_sena_simulacoes"),
		MongoCounterCollection:    envOr("MONGODB_COUNTER_COLLECTION", "counters"),
		MongoUsername:             envOr("MONGODB_USERNAME", ""),
		MongoPassword:             envOr("MONGODB_PASSWORD", ""),
		MongoAuthDatabase:         envOr("MONGODB_AUTH_DATABASE", "admin"),
		MongoTimeoutMs:            envInt("MONGODB_CONNECT_TIMEOUT_MS", 2500),
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
