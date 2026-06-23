package config

import (
	"fmt"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv  string
	AppPort string

	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string
	DatabaseURL string

	JWTSecret string

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	GoogleOAuthState   string
}

func Load() Config {
	_ = godotenv.Load()

	cfg := Config{
		AppEnv:     getEnv("APP_ENV", "local"),
		AppPort:    getEnv("APP_PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "cermin_db"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		JWTSecret: getEnv("JWT_SECRET", "change-this-secret"),

		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/google/callback"),
		GoogleOAuthState:   getEnv("GOOGLE_OAUTH_STATE", "change-this-state"),
	}

	cfg.DatabaseURL = buildDatabaseURL(cfg)

	return cfg
}

func buildDatabaseURL(cfg Config) string {
	databaseURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host:   fmt.Sprintf("%s:%s", cfg.DBHost, cfg.DBPort),
		Path:   cfg.DBName,
	}

	query := databaseURL.Query()
	query.Set("sslmode", cfg.DBSSLMode)
	databaseURL.RawQuery = query.Encode()

	return databaseURL.String()
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
