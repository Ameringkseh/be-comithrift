package main

import (
	"log"

	"be-comithrift/internal/config"
	"be-comithrift/internal/db"
	"be-comithrift/internal/handlers"
	"be-comithrift/internal/middleware"
	"be-comithrift/internal/models"
	"be-comithrift/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	cfg := config.LoadConfig()

	// Inisialisasi koneksi DB (jangan crash jika belum dikonfigurasi saat dev lokal)
	gormDB, err := db.InitDB(cfg.DatabaseURL, cfg.Env)
	if err != nil {
		log.Printf("Peringatan inisialisasi DB: %v\n", err)
	} else {
		// Jalankan AutoMigrate. Jika gagal, hentikan startup untuk mencegah error berantai pada seed
		if err := db.AutoMigrate(gormDB); err != nil {
			log.Fatalf("Gagal menjalankan migrasi database: %v", err)
		}
		if cfg.Env == "development" {
			if err := db.SeedData(gormDB); err != nil {
				log.Printf("Peringatan seed DB: %v\n", err)
			}
		}
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

	// Inisialisasi Service & Handler
	produkService := services.NewProdukService(gormDB)
	produkHandler := handlers.NewProdukHandler(produkService)

	pesananService := services.NewPesananService(gormDB)
	pesananHandler := handlers.NewPesananHandler(pesananService)

	// Route Publik (Tabel 3.7 & Task 3.1, 3.2, 4.1)
	api := app.Group("/api")
	api.Get("/produk", produkHandler.GetList)
	api.Get("/produk/:id", produkHandler.GetByID)
	api.Post("/pesanan", pesananHandler.Create)

	// Route Admin/Relawan Produk dengan Middleware JWT + Role Check (Task 3.3, 3.4, 3.5)
	admin := api.Group("/admin")
	admin.Use(middleware.VerifyJWT(cfg.SupabaseJWTSecret))

	// Endpoint produk dapat diakses oleh Relawan dan Admin
	produkAdmin := admin.Group("/produk")
	produkAdmin.Use(middleware.RequireRole(gormDB, models.PeranRelawan, models.PeranAdmin))
	produkAdmin.Post("", produkHandler.Create)
	produkAdmin.Post("/", produkHandler.Create)
	produkAdmin.Patch("/:id", produkHandler.Update)
	produkAdmin.Delete("/:id", produkHandler.Delete)

	// Endpoint pesanan admin hanya dapat diakses oleh Admin (Task 4.2, 4.3)
	pesananAdmin := admin.Group("/pesanan")
	pesananAdmin.Use(middleware.RequireRole(gormDB, models.PeranAdmin))
	pesananAdmin.Get("", pesananHandler.GetList)
	pesananAdmin.Get("/", pesananHandler.GetList)

	log.Printf("COMITHRIFT API berjalan di port %s (%s)\n", cfg.Port, cfg.Env)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}
