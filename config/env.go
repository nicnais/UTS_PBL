package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Settings struct {
	Host, Port, Secret, Issuer, AllowedOrigins string
	AccessTTL, RefreshTTL                      time.Duration
}

func LoadEnv() error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func GetEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func Load() (Settings, error) {
	if err := LoadEnv(); err != nil {
		return Settings{}, err
	}
	s := Settings{
		Host: GetEnv("APP_HOST", "127.0.0.1"), Port: GetEnv("APP_PORT", "3000"),
		Secret: os.Getenv("JWT_SECRET"), Issuer: GetEnv("JWT_ISSUER", "laundry-api"),
		AllowedOrigins: GetEnv("ALLOWED_ORIGINS", "http://localhost:5173"),
	}
	if len(s.Secret) < 32 {
		return s, fmt.Errorf("JWT_SECRET wajib berisi minimal 32 karakter acak; jalankan go run ./cmd/secret")
	}
	minutes, err := boundedEnv("JWT_ACCESS_TTL_MINUTES", 15, 1, 60)
	if err != nil {
		return s, err
	}
	days, err := boundedEnv("JWT_REFRESH_TTL_DAYS", 7, 1, 30)
	if err != nil {
		return s, err
	}
	s.AccessTTL = time.Duration(minutes) * time.Minute
	s.RefreshTTL = time.Duration(days) * 24 * time.Hour
	return s, nil
}

func boundedEnv(key string, fallback, min, max int) (int, error) {
	n, err := strconv.Atoi(GetEnv(key, strconv.Itoa(fallback)))
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("%s harus %d sampai %d", key, min, max)
	}
	return n, nil
}