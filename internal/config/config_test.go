package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear envs used by config
	os.Unsetenv("PORT")
	os.Unsetenv("HELIUS_RPC_URL")
	os.Unsetenv("MONGO_URI")
	os.Unsetenv("MONGO_DB")
	os.Unsetenv("RATE_LIMIT_RPM")
	os.Unsetenv("CACHE_TTL")
	os.Unsetenv("KEY_CACHE_TTL")
	os.Unsetenv("BALANCE_TIMEOUT")
	os.Unsetenv("MAX_CONCURRENCY")
	os.Unsetenv("SOL_COMMITMENT")
	os.Unsetenv("ADMIN_TOKEN")

	c := Load()
	if c.Port != "8080" { t.Fatalf("port=%s", c.Port) }
	if c.MongoURI == "" || c.MongoDB == "" { t.Fatalf("mongo not set") }
	if c.RateLimitRPM <= 0 || c.CacheTTL <= 0 || c.KeyCacheTTL <= 0 || c.BalanceTimeout <= 0 || c.MaxConcurrency <= 0 { t.Fatalf("invalid defaults") }
}

func TestLoad_EnvOverrides(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("RATE_LIMIT_RPM", "123")
	os.Setenv("CACHE_TTL", "150ms")
	os.Setenv("KEY_CACHE_TTL", "2s")
	os.Setenv("BALANCE_TIMEOUT", "5s")
	os.Setenv("MAX_CONCURRENCY", "7")
	defer func(){
		os.Unsetenv("PORT"); os.Unsetenv("RATE_LIMIT_RPM"); os.Unsetenv("CACHE_TTL"); os.Unsetenv("KEY_CACHE_TTL"); os.Unsetenv("BALANCE_TIMEOUT"); os.Unsetenv("MAX_CONCURRENCY")
	}()
	c := Load()
	if c.Port != "9090" { t.Fatalf("port=%s", c.Port) }
	if c.RateLimitRPM != 123 { t.Fatalf("rpm=%d", c.RateLimitRPM) }
	if c.CacheTTL != 150*time.Millisecond || c.KeyCacheTTL != 2*time.Second || c.BalanceTimeout != 5*time.Second { t.Fatalf("durations not applied") }
	if c.MaxConcurrency != 7 { t.Fatalf("max=%d", c.MaxConcurrency) }
}
