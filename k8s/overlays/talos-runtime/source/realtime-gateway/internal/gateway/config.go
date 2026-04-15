package gateway

import (
	"os"
	"strconv"
)

type Config struct {
	Port               string
	AppEnv             string
	LogLevel           string
	AllowedOrigins     string
	RoomClockSeconds   int
	MatchCoreHost      string
	MatchCorePort      int
	MatchCoreTimeoutMs int
}

func LoadConfig() Config {
	return Config{
		Port:               envOr("PORT", "3000"),
		AppEnv:             envOr("APP_ENV", "production"),
		LogLevel:           envOr("LOG_LEVEL", "INFO"),
		AllowedOrigins:     envOr("ALLOWED_ORIGINS", "http://localhost:5173,http://chess.local"),
		RoomClockSeconds:   envInt("ROOM_CLOCK_SECONDS", 600),
		MatchCoreHost:      envOr("MATCH_CORE_HOST", "match-core"),
		MatchCorePort:      envInt("MATCH_CORE_PORT", 50052),
		MatchCoreTimeoutMs: envInt("MATCH_CORE_TIMEOUT_MS", 2500),
	}
}

func (c Config) MatchCoreAddress() string {
	return c.MatchCoreHost + ":" + strconv.Itoa(c.MatchCorePort)
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
