package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	Env                 string
	DatabaseURL         string
	SupabaseProjectURL  string
	SupabaseJWTSecret   string
	CORSAllowedOrigins  string
	LogLevel            string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Info: File .env tidak ditemukan, membaca konfigurasi dari environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	return &Config{
		Port:               port,
		Env:                env,
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		SupabaseProjectURL: os.Getenv("SUPABASE_PROJECT_URL"),
		SupabaseJWTSecret:  os.Getenv("SUPABASE_JWT_SECRET"),
		CORSAllowedOrigins: os.Getenv("CORS_ALLOWED_ORIGINS"),
		LogLevel:           os.Getenv("LOG_LEVEL"),
	}
}

