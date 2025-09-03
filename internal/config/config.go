package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds environment-driven configuration.
type Config struct {
	Port            string
	HeliusURL       string
	MongoURI        string
	MongoDB         string
	RateLimitRPM    int
	CacheTTL        time.Duration
	KeyCacheTTL     time.Duration
	BalanceTimeout  time.Duration
	MaxConcurrency  int
	SolCommitment   string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getint(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getdur(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// Load loads configuration from environment variables with sane defaults.
func Load() Config {
	return Config{
		Port:           getenv("PORT", "8080"),
		HeliusURL:      getenv("HELIUS_RPC_URL", ""),
		MongoURI:       getenv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:        getenv("MONGO_DB", "solapi"),
		RateLimitRPM:   getint("RATE_LIMIT_RPM", 10),
		CacheTTL:       getdur("CACHE_TTL", 10*time.Second),
		KeyCacheTTL:    getdur("KEY_CACHE_TTL", 60*time.Second),
		BalanceTimeout: getdur("BALANCE_TIMEOUT", 3*time.Second),
		MaxConcurrency: getint("MAX_CONCURRENCY", 16),
		SolCommitment:  getenv("SOL_COMMITMENT", "finalized"),
	}
}
