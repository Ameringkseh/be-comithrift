package main

import (
	"log"

	"be-comithrift/internal/config"
	"be-comithrift/internal/db"
	"be-comithrift/internal/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	cfg := config.LoadConfig()

	// Inisialisasi koneksi DB (jangan crash jika belum dikonfigurasi saat dev lokal)
	if _, err := db.InitDB(cfg.DatabaseURL, cfg.Env); err != nil {
		log.Printf("Peringatan inisialisasi DB: %v\n", err)
	}

	app := fiber.New(fiber.Config{
		AppName: "COMITHRIFT Backend API",
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSAllowedOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PATCH, DELETE, OPTIONS",
	}))

	// Endpoint Health-check (Fase 0.3)
	app.Get("/healthz", handlers.HealthCheckHandler)

	log.Printf("COMITHRIFT API berjalan di port %s (%s)\n", cfg.Port, cfg.Env)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}

