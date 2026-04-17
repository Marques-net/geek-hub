package tictactoe

import (
	"os"
	"strconv"
)

type Config struct {
	Port               string
	MetricsPort        string
	AppEnv             string
	LogLevel           string
	RedisHost          string
	RedisPort          int
	RedisPassword      string
	RedisKeyPrefix     string
	GameTTLSeconds     int
	RoomClockSeconds   int
	BotEngineHost      string
	BotEnginePort      int
	BotEngineTimeoutMs int
}

func LoadConfig() Config {
	return Config{
		Port:               envOr("PORT", "50052"),
		MetricsPort:        envOr("METRICS_PORT", "9090"),
		AppEnv:             envOr("APP_ENV", "production"),
		LogLevel:           envOr("LOG_LEVEL", "INFO"),
		RedisHost:          envOr("REDIS_HOST", "redis"),
		RedisPort:          envInt("REDIS_PORT", 6379),
		RedisPassword:      os.Getenv("REDIS_PASSWORD"),
		RedisKeyPrefix:     envOr("REDIS_KEY_PREFIX", "geek-hub"),
		GameTTLSeconds:     envInt("GAME_TTL_SECONDS", 86400),
		RoomClockSeconds:   envInt("ROOM_CLOCK_SECONDS", 600),
		BotEngineHost:      envOr("BOT_ENGINE_HOST", "bot-engine"),
		BotEnginePort:      envInt("BOT_ENGINE_PORT", 50051),
		BotEngineTimeoutMs: envInt("BOT_ENGINE_TIMEOUT_MS", 1500),
	}
}

func (c Config) RoomClockMs() int64 {
	return int64(c.RoomClockSeconds) * 1000
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
