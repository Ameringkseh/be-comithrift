package main

import (
	"log"

	"be-comithrift/internal/config"
	"be-comithrift/internal/db"
)

func main() {
	log.Println("== COMITHRIFT DATABASE RESET & SEED TOOL ==")
	cfg := config.LoadConfig()

	gormDB, err := db.InitDB(cfg.DatabaseURL, cfg.Env)
	if err != nil {
		log.Fatalf("Gagal terhubung ke database: %v", err)
	}

	if err := db.AutoMigrate(gormDB); err != nil {
		log.Fatalf("Gagal migrasi database: %v", err)
	}

	if err := db.ResetAndSeedAll(gormDB); err != nil {
		log.Fatalf("Gagal melakukan reset & seed: %v", err)
	}

	log.Println("Berhasil! Database telah direset dengan 6 produk thrift berkualitas tinggi dan data sinkron.")
}
