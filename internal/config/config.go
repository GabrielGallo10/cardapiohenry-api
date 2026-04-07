package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName     string
	Environment string
	HTTPPort    string
	DatabaseURL string
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("arquivo .env nao encontrado, usando variaveis de ambiente")
	}

	cfg := Config{
		AppName:     getEnv("APP_NAME", "cardapio-henry-api"),
		Environment: getEnv("APP_ENV", "development"),
		HTTPPort:    getEnv("HTTP_PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/cardapio_henry?sslmode=disable"),
	}

	return cfg
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
