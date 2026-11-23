package handlers

import (
	"encoding/base64"
	"strings"
	"time"

	"be-comithrift/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func parseSecretBytes(secret string) []byte {
	trimmed := strings.TrimSpace(secret)
	if len(trimmed) == 88 || strings.HasSuffix(trimmed, "=") {
		if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(decoded) > 0 {
			return decoded
		}
	}
	if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(decoded) >= 32 {
		return decoded
	}
	return []byte(secret)
}

// DevTokensHandler memastikan akun Admin & Relawan tersinkronisasi di tabel profil_pengguna database,
// serta mengembalikan token JWT yang valid dan ID pesanan uji untuk keperluan pengujian lokal.
func DevTokensHandler(db *gorm.DB, secret string) fiber.Handler {
	secretBytes := parseSecretBytes(secret)

	return func(c *fiber.Ctx) error {
		// 1. Pastikan profil Admin ada di database
		var admin models.ProfilPengguna
		if err := db.Where("peran = ?", models.PeranAdmin).First(&admin).Error; err != nil {
			admin = models.ProfilPengguna{
				IDPengguna: "00000000-0000-0000-0000-000000000001",
				Nama:       "Admin Fiktif COMITHRIFT",
				Peran:      models.PeranAdmin,
			}
			db.Save(&admin)
		}

		// 2. Pastikan profil Relawan ada di database
		var relawan models.ProfilPengguna
		if err := db.Where("peran = ?", models.PeranRelawan).First(&relawan).Error; err != nil {
			relawan = models.ProfilPengguna{
				IDPengguna: "00000000-0000-0000-0000-000000000002",
				Nama:       "Relawan Fiktif COMITHRIFT",
				Peran:      models.PeranRelawan,
			}
			db.Save(&relawan)
		}

		// 3. Ambil ID pesanan uji (atau buat jika belum ada)
		var pesanan models.Pesanan
		if err := db.Order("id_pesanan desc").First(&pesanan).Error; err != nil {
			pesanan = models.Pesanan{
				KodePesanan:       "CMT-SAMPLE-RBAC",
				Tanggal:           time.Now(),
				NamaPembeli:       "Pembeli Sampel Uji",
				KontakWhatsapp:    "081234567890",
				MetodeSerahTerima: "pengiriman",
				TotalBelanja:      50000,
				NominalDonasi:     10000,
				StatusPesanan:     models.StatusPesananMenungguPembayaran,
			}
			db.Create(&pesanan)
		}

		// 4. Buat token JWT yang ditandatangani untuk masing-masing user
		now := time.Now()
		exp := now.Add(365 * 24 * time.Hour).Unix()

		adminClaims := jwt.MapClaims{
			"sub":  admin.IDPengguna,
			"role": "authenticated",
			"exp":  exp,
		}
		adminToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, adminClaims).SignedString(secretBytes)
		if err != nil {
			adminToken, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, adminClaims).SignedString([]byte(secret))
		}

		relawanClaims := jwt.MapClaims{
			"sub":  relawan.IDPengguna,
			"role": "authenticated",
			"exp":  exp,
		}
		relawanToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, relawanClaims).SignedString(secretBytes)
		if err != nil {
			relawanToken, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, relawanClaims).SignedString([]byte(secret))
		}

		return c.JSON(fiber.Map{
			"sukses": true,
			"data": fiber.Map{
				"token_admin":   adminToken,
				"token_relawan": relawanToken,
				"pesanan_id":    pesanan.IDPesanan,
			},
		})
	}
}
