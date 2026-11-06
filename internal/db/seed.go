package db

import (
	"log"

	"be-comithrift/internal/models"

	"gorm.io/gorm"
)

// SeedData memasukkan data uji fiktif minimal ke database untuk pengujian lokal
// Sumber: Batasan proposal (data fiktif) & .agents/rules/20-data-model.md
func SeedData(database *gorm.DB) error {
	log.Println("Memeriksa dan memasukkan seed data uji fiktif...")

	// 1. Seed Info Program (Cukup 1 baris konfigurasi jika belum ada)
	var infoCount int64
	database.Model(&models.InfoProgram{}).Count(&infoCount)
	if infoCount == 0 {
		info := models.InfoProgram{
			NamaYayasan:         "Yayasan Peduli Insan Berbagi",
			DeskripsiProgram:    "Program Tebar Cinta COMITHRIFT menyalurkan 100% keuntungan dan alokasi donasi thrift untuk pendidikan anak yatim piatu.",
			InfoPembayaran:      "BCA 1234567890 a.n. Yayasan Peduli Insan / Mandiri 0987654321",
			KontakWhatsappAdmin: "081234567890",
		}
		if err := database.Create(&info).Error; err != nil {
			log.Printf("Gagal seed InfoProgram: %v", err)
		} else {
			log.Println("Seed InfoProgram berhasil dibuat")
		}
	}

	// 2. Seed Profil Pengguna Dummy (Admin & Relawan dummy untuk pengujian lokal)
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
			if err := database.Create(&p).Error; err != nil {
				log.Printf("Gagal seed ProfilPengguna (%s): %v", p.Nama, err)
			}
		}
		log.Println("Seed ProfilPengguna dummy berhasil dibuat")
	}

	// 3. Seed Produk Uji Fiktif (Beberapa kategori & ukuran dengan status 'tersedia')
	var produkCount int64
	database.Model(&models.Produk{}).Count(&produkCount)
	if produkCount == 0 {
		dummyProduk := []models.Produk{
			{
				NamaProduk: "Jaket Denim Vintage Blue",
				Harga:      85000,
				Deskripsi:  "Jaket denim tebal model oversize warna faded blue. Kancing lengkap dan kondisi mulus.",
				Ukuran:     "L",
				Kondisi:    "Sangat Baik",
				Kategori:   "Jaket",
				Foto:       "https://supabase.co/storage/v1/object/public/produk/sample-jaket-denim.jpg",
				Status:     models.StatusProdukTersedia,
			},
			{
				NamaProduk: "Kemeja Flanel Kotak Earth Tone",
				Harga:      45000,
				Deskripsi:  "Kemeja flanel bahan katun adem, motif tartan cokelat krem cocok untuk outer.",
				Ukuran:     "M",
				Kondisi:    "Baik",
				Kategori:   "Kemeja",
				Foto:       "https://supabase.co/storage/v1/object/public/produk/sample-flanel-tartan.jpg",
				Status:     models.StatusProdukTersedia,
			},
			{
				NamaProduk: "Kaos Band Vintage Hitam",
				Harga:      35000,
				Deskripsi:  "Kaos bahan combed washed black, sablon depan belakang masih sangat jelas.",
				Ukuran:     "XL",
				Kondisi:    "Baik",
				Kategori:   "Kaos",
				Foto:       "https://supabase.co/storage/v1/object/public/produk/sample-kaos-band.jpg",
				Status:     models.StatusProdukTersedia,
			},
			{
				NamaProduk: "Celana Corduroy Brown Chino",
				Harga:      60000,
				Deskripsi:  "Celana corduroy lurus warna cokelat tanah, nyaman dipakai harian.",
				Ukuran:     "32",
				Kondisi:    "Sangat Baik",
				Kategori:   "Celana",
				Foto:       "https://supabase.co/storage/v1/object/public/produk/sample-celana-corduroy.jpg",
				Status:     models.StatusProdukTersedia,
			},
		}

		for _, prod := range dummyProduk {
			if err := database.Create(&prod).Error; err != nil {
				log.Printf("Gagal seed Produk (%s): %v", prod.NamaProduk, err)
			}
		}
		log.Println("Seed Produk dummy berhasil dibuat")
	}

	return nil
}

