package app

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL, Port, Env, SessionSecret string
	AdminEmail, AdminPassword             string
	PartnerEmail, PartnerPassword         string
}

func loadConfig() (Config, error) {
	c := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"), Port: envOr("PORT", "8080"),
		Env: envOr("APP_ENV", "development"), SessionSecret: os.Getenv("SESSION_SECRET"),
		AdminEmail: os.Getenv("ADMIN_EMAIL"), AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		PartnerEmail: os.Getenv("PARTNER_EMAIL"), PartnerPassword: os.Getenv("PARTNER_PASSWORD"),
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	if len(c.SessionSecret) < 32 {
		return c, fmt.Errorf("SESSION_SECRET must contain at least 32 characters")
	}
	if _, err := strconv.Atoi(c.Port); err != nil {
		return c, fmt.Errorf("invalid PORT: %w", err)
	}
	return c, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
