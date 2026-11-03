package db

import (
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB menginisialisasi koneksi GORM ke PostgreSQL Supabase via DATABASE_URL
func InitDB(databaseURL string, env string) (*gorm.DB, error) {
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL kosong: silakan isi DATABASE_URL di file .env")
	}

	logMode := logger.Info
	if env == "production" {
		logMode = logger.Warn
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logMode),
	})
	if err != nil {
		return nil, fmt.Errorf("gagal terhubung ke database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil instance sql.DB: %w", err)
	}

	// Konfigurasi connection pooling standar
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db
	log.Println("Koneksi database PostgreSQL berhasil diinisialisasi")
	return db, nil
}

