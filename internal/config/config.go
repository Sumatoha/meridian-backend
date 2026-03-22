package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port               int
	DatabaseURL        string
	SupabaseURL        string
	SupabaseJWTSecret  string
	SupabaseStorageURL string
	SupabaseServiceKey string
	AnthropicAPIKey    string
	MetaAppID          string
	MetaAppSecret      string
	DodoAPIKey         string
	DodoWebhookSecret  string
	KaspiMerchantID    string
	KaspiSecret        string
	Environment        string
}

func Load() (Config, error) {
	port := 8080
	if v := os.Getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid PORT: %w", err)
		}
		port = p
	}

	cfg := Config{
		Port:               port,
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		SupabaseURL:        os.Getenv("SUPABASE_URL"),
		SupabaseJWTSecret:  os.Getenv("SUPABASE_JWT_SECRET"),
		SupabaseStorageURL: os.Getenv("SUPABASE_STORAGE_URL"),
		SupabaseServiceKey: os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		AnthropicAPIKey:    os.Getenv("ANTHROPIC_API_KEY"),
		MetaAppID:          os.Getenv("META_APP_ID"),
		MetaAppSecret:      os.Getenv("META_APP_SECRET"),
		DodoAPIKey:         os.Getenv("DODO_API_KEY"),
		DodoWebhookSecret:  os.Getenv("DODO_WEBHOOK_SECRET"),
		KaspiMerchantID:    os.Getenv("KASPI_MERCHANT_ID"),
		KaspiSecret:        os.Getenv("KASPI_SECRET"),
		Environment:        getEnvOr("ENVIRONMENT", "development"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.SupabaseJWTSecret == "" {
		return Config{}, fmt.Errorf("SUPABASE_JWT_SECRET is required")
	}
	if cfg.AnthropicAPIKey == "" {
		return Config{}, fmt.Errorf("ANTHROPIC_API_KEY is required")
	}

	return cfg, nil
}

func (c Config) IsProduction() bool {
	return c.Environment == "production"
}

func getEnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
