package db

import (
	"fmt"
	"log"

	"be-comithrift/internal/models"

	"gorm.io/gorm"
)

// AutoMigrate menjalankan migrasi skema tabel GORM beserta penegakan constraint
// Sumber: Tabel 3.3 proposal & .agents/rules/20-data-model.md
func AutoMigrate(database *gorm.DB) error {
	log.Println("Menjalankan AutoMigrate GORM...")

	// 1. Jalankan AutoMigrate untuk seluruh model
	err := database.AutoMigrate(
		&models.ProfilPengguna{},
		&models.Produk{},
		&models.Pesanan{},
		&models.ItemPesanan{},
		&models.PenyaluranDonasi{},
		&models.InfoProgram{},
	)
	if err != nil {
		return fmt.Errorf("gagal menjalankan AutoMigrate: %w", err)
	}

	// 2. Tambahkan / pastikan constraint CHECK PostgreSQL jika belum ada
	constraints := []struct {
		name       string
		table      string
		constraint string
	}{
		{
			name:       "chk_profil_peran",
			table:      "profil_pengguna",
			constraint: "CHECK (peran IN ('Admin', 'Relawan'))",
		},
		{
			name:       "chk_produk_harga",
			table:      "produk",
			constraint: "CHECK (harga > 0)",
		},
		{
			name:       "chk_produk_status",
			table:      "produk",
			constraint: "CHECK (status IN ('tersedia', 'dipesan', 'terjual'))",
		},
		{
			name:       "chk_pesanan_total",
			table:      "pesanan",
			constraint: "CHECK (total_belanja >= 0)",
		},
		{
			name:       "chk_pesanan_donasi",
			table:      "pesanan",
			constraint: "CHECK (nominal_donasi >= 0)",
		},
		{
			name:       "chk_pesanan_status",
			table:      "pesanan",
			constraint: "CHECK (status_pesanan IN ('menunggu_pembayaran', 'dibayar', 'selesai', 'dibatalkan'))",
		},
		{
			name:       "chk_item_harga",
			table:      "item_pesanan",
			constraint: "CHECK (harga_saat_pesan > 0)",
		},
		{
			name:       "chk_penyaluran_nominal",
			table:      "penyaluran_donasi",
			constraint: "CHECK (nominal >= 0)",
		},
	}

	for _, c := range constraints {
		sql := fmt.Sprintf(`
			DO $$
			BEGIN
				IF NOT EXISTS (
					SELECT 1 FROM pg_constraint WHERE conname = '%s'
				) THEN
					ALTER TABLE %s ADD CONSTRAINT %s %s;
				END IF;
			END $$;
		`, c.name, c.table, c.name, c.constraint)

		if err := database.Exec(sql).Error; err != nil {
			log.Printf("Catatan constraint %s pada tabel %s: %v", c.name, c.table, err)
		}
	}

	log.Println("AutoMigrate dan verifikasi constraint selesai.")
	return nil
}

