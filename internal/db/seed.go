package db

import (
	"log"
	"time"

	"be-comithrift/internal/models"

	"gorm.io/gorm"
)

// GetCleanThriftProduk mengembalikan daftar produk pakaian bekas riil dengan foto Unsplash HD yang stabil
func GetCleanThriftProduk() []models.Produk {
	return []models.Produk{
		{
			NamaProduk: "Jaket Denim Vintage Blue",
			Harga:      85000,
			Deskripsi:  "Jaket denim tebal model oversize warna faded blue. Kancing lengkap dan kondisi sangat mulus.",
			Ukuran:     "L",
			Kondisi:    "Sangat Baik",
			Kategori:   "Jaket",
			Foto:       "https://images.unsplash.com/photo-1576995853123-5a10305d93c0?auto=format&fit=crop&w=800&q=80",
			Status:     models.StatusProdukTersedia,
		},
		{
			NamaProduk: "Kemeja Flanel Tartan Retro",
			Harga:      45000,
			Deskripsi:  "Kemeja flanel bahan katun adem, motif tartan cokelat krem cocok untuk outer santai.",
			Ukuran:     "M",
			Kondisi:    "Baik",
			Kategori:   "Kemeja",
			Foto:       "https://images.unsplash.com/photo-1596755094514-f87e34085b2c?auto=format&fit=crop&w=800&q=80",
			Status:     models.StatusProdukTersedia,
		},
		{
			NamaProduk: "Tebar Cinta Hoodie Streetwear",
			Harga:      170000,
			Deskripsi:  "Hoodie katun fleece tebal warna pinkish neutral. Hangat, nyaman, dan hasil kurasi relawan.",
			Ukuran:     "XL",
			Kondisi:    "Sangat Baik",
			Kategori:   "Streetwear",
			Foto:       "https://images.unsplash.com/photo-1556905055-8f358a7a47b2?auto=format&fit=crop&w=800&q=80",
			Status:     models.StatusProdukTersedia,
		},
		{
			NamaProduk: "Kaos Band Washed Vintage",
			Harga:      35000,
			Deskripsi:  "Kaos bahan combed washed black, sablon depan belakang masih sangat jelas dan berkarakter.",
			Ukuran:     "L",
			Kondisi:    "Baik",
			Kategori:   "Kaos",
			Foto:       "https://images.unsplash.com/photo-1521572267360-ee0c2909d518?auto=format&fit=crop&w=800&q=80",
			Status:     models.StatusProdukTersedia,
		},
		{
			NamaProduk: "Celana Corduroy Earth Tone",
			Harga:      60000,
			Deskripsi:  "Celana corduroy lurus warna cokelat tanah vintage, potongan regular fit nyaman.",
			Ukuran:     "32",
			Kondisi:    "Sangat Baik",
			Kategori:   "Celana",
			Foto:       "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?auto=format&fit=crop&w=800&q=80",
			Status:     models.StatusProdukTersedia,
		},
		{
			NamaProduk: "Neo-Brutalism Cap Streetwear",
			Harga:      125000,
			Deskripsi:  "Topi trucker berdesain clean minimalis warna putih bersih dengan aksen bordir kokoh.",
			Ukuran:     "All Size",
			Kondisi:    "Sangat Baik",
			Kategori:   "Aksesoris",
			Foto:       "https://images.unsplash.com/photo-1588850561407-ed78c282e89b?auto=format&fit=crop&w=800&q=80",
			Status:     models.StatusProdukTersedia,
		},
	}
}

// SeedData memasukkan data uji fiktif minimal ke database jika tabel masih kosong
func SeedData(database *gorm.DB) error {
	log.Println("Memeriksa dan memasukkan seed data uji fiktif...")

	// 1. Seed Info Program
	var infoCount int64
	database.Model(&models.InfoProgram{}).Count(&infoCount)
	if infoCount == 0 {
		info := models.InfoProgram{
			NamaYayasan:         "Yayasan Peduli Insan Berbagi",
			DeskripsiProgram:    "Program Tebar Cinta COMITHRIFT menyalurkan 100% keuntungan dan alokasi donasi thrift untuk pendidikan dan nutrisi anak yatim piatu.",
			InfoPembayaran:      "BCA 1234567890 a.n. Yayasan Peduli Insan / Mandiri 0987654321",
			KontakWhatsappAdmin: "081234567890",
		}
		if err := database.Create(&info).Error; err != nil {
			log.Printf("Gagal seed InfoProgram: %v", err)
		} else {
			log.Println("Seed InfoProgram berhasil dibuat")
		}
	}

	// 2. Seed Profil Pengguna Dummy (Admin & Relawan)
	var profilCount int64
	database.Model(&models.ProfilPengguna{}).Count(&profilCount)
	if profilCount == 0 {
		dummyProfiles := []models.ProfilPengguna{
			{
				IDPengguna: "00000000-0000-0000-0000-000000000001",
				Nama:       "Admin Fiktif COMITHRIFT",
				Peran:      models.PeranAdmin,
			},
			{
				IDPengguna: "00000000-0000-0000-0000-000000000002",
				Nama:       "Relawan Fiktif COMITHRIFT",
				Peran:      models.PeranRelawan,
			},
		}
		for _, p := range dummyProfiles {
			database.Save(&p)
		}
		log.Println("Seed ProfilPengguna dummy berhasil dibuat")
	}

	// 3. Seed Produk Uji Bersih
	var produkCount int64
	database.Model(&models.Produk{}).Count(&produkCount)
	if produkCount == 0 {
		dummyProduk := GetCleanThriftProduk()
		for _, prod := range dummyProduk {
			if err := database.Create(&prod).Error; err != nil {
				log.Printf("Gagal seed Produk (%s): %v", prod.NamaProduk, err)
			}
		}
		log.Println("Seed Produk dummy berhasil dibuat")
	}

	return nil
}

// ResetAndSeedAll membersihkan produk sampah dan pesanan lama, lalu mengisikan data produk thrift berkualitas tinggi
func ResetAndSeedAll(database *gorm.DB) error {
	log.Println("Memulai reset dan sinkronisasi penuh database COMITHRIFT...")

	// 1. Bersihkan item_pesanan & pesanan terlebih dahulu (agar tidak bentrok foreign key)
	if err := database.Exec("DELETE FROM item_pesanan").Error; err != nil {
		log.Printf("Info pembersihan item_pesanan: %v", err)
	}
	if err := database.Exec("DELETE FROM pesanan").Error; err != nil {
		log.Printf("Info pembersihan pesanan: %v", err)
	}

	// 2. Bersihkan produk
	if err := database.Exec("DELETE FROM produk").Error; err != nil {
		log.Printf("Info pembersihan produk: %v", err)
	}

	// 3. Masukkan 6 produk thrift berkualitas tinggi
	products := GetCleanThriftProduk()
	for _, p := range products {
		if err := database.Create(&p).Error; err != nil {
			log.Printf("Gagal memasukkan produk %s: %v", p.NamaProduk, err)
			return err
		}
	}

	// 4. Pastikan Profil Pengguna Admin & Relawan ada
	admin := models.ProfilPengguna{
		IDPengguna: "00000000-0000-0000-0000-000000000001",
		Nama:       "Admin Fiktif COMITHRIFT",
		Peran:      models.PeranAdmin,
	}
	database.Save(&admin)

	relawan := models.ProfilPengguna{
		IDPengguna: "00000000-0000-0000-0000-000000000002",
		Nama:       "Relawan Fiktif COMITHRIFT",
		Peran:      models.PeranRelawan,
	}
	database.Save(&relawan)

	// 5. Buat 1 pesanan sampel aktif untuk kebutuhan uji diagnostik RBAC (TC02 / TC04)
	sampleOrder := models.Pesanan{
		KodePesanan:       "CMT-SAMPLE-RBAC",
		Tanggal:           time.Now(),
		NamaPembeli:       "Pembeli Sampel Uji",
		KontakWhatsapp:    "081234567890",
		MetodeSerahTerima: "pengiriman",
		TotalBelanja:      85000,
		NominalDonasi:     15000,
		StatusPesanan:     models.StatusPesananMenungguPembayaran,
	}
	database.Create(&sampleOrder)

	log.Println("Reset dan sinkronisasi penuh database berhasil diselesaikan.")
	return nil
}


