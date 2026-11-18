package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"be-comithrift/internal/handlers"
	"be-comithrift/internal/middleware"
	"be-comithrift/internal/models"
	"be-comithrift/internal/services"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func createRekapTestToken(secret, sub string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": sub,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func setupAdminRekapTestApp(t *testing.T) (*fiber.App, sqlmock.Sqlmock, string) {
	secret := "test-secret-admin-rekap"
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	svc := services.NewRekapService(gormDB)
	handler := handlers.NewRekapHandler(svc)

	app := fiber.New()
	api := app.Group("/api")

	admin := api.Group("/admin")
	admin.Use(middleware.VerifyJWT(secret))
	admin.Get("/rekap", middleware.RequireRole(gormDB, models.PeranAdmin), handler.GetRekap)

	return app, mock, secret
}

// Skenario F-11, T-14: Admin mengambil rekapitulasi data pesanan, penjualan, donasi, dan produk (status 200)
func TestAdminGetRekap_Success_T14(t *testing.T) {
	app, mock, secret := setupAdminRekapTestApp(t)
	token := createRekapTestToken(secret, "admin-uuid-1")

	// Middleware RequireRole
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-uuid-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	// 1. Rekap status pesanan (GROUP BY status_pesanan)
	mock.ExpectQuery(`SELECT status_pesanan, count\(\*\) as count FROM "pesanan" GROUP BY "status_pesanan"`).
		WillReturnRows(sqlmock.NewRows([]string{"status_pesanan", "count"}).
			AddRow(models.StatusPesananMenungguPembayaran, 2).
			AddRow(models.StatusPesananDibayar, 3).
			AddRow(models.StatusPesananSelesai, 5).
			AddRow(models.StatusPesananDibatalkan, 1))

	// 2. Total penjualan (pesanan dibayar/selesai)
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(total_belanja\), 0\) FROM "pesanan" WHERE status_pesanan IN \(\$1,\$2\)`).
		WithArgs(models.StatusPesananDibayar, models.StatusPesananSelesai).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(850000))

	// 3. Donasi terkumpul (pesanan dibayar/selesai)
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(nominal_donasi\), 0\) FROM "pesanan" WHERE status_pesanan IN \(\$1,\$2\)`).
		WithArgs(models.StatusPesananDibayar, models.StatusPesananSelesai).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(100000))

	// 4. Donasi tersalurkan
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(nominal\), 0\) FROM "penyaluran_donasi"`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(40000))

	// 5. Rekap produk (GROUP BY status)
	mock.ExpectQuery(`SELECT status, count\(\*\) as count FROM "produk" GROUP BY "status"`).
		WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).
			AddRow(models.StatusProdukTersedia, 10).
			AddRow(models.StatusProdukDipesan, 2).
			AddRow(models.StatusProdukTerjual, 8))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/rekap", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for T-14 rekap, got: %d", resp.StatusCode)
	}

	var res struct {
		Data services.RekapResponse `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&res)

	// Verifikasi rekap pesanan
	if res.Data.Pesanan.Total != 11 {
		t.Errorf("expected pesanan total 11, got: %d", res.Data.Pesanan.Total)
	}
	if res.Data.Pesanan.MenungguPembayaran != 2 || res.Data.Pesanan.Dibayar != 3 || res.Data.Pesanan.Selesai != 5 || res.Data.Pesanan.Dibatalkan != 1 {
		t.Errorf("unexpected pesanan rekap breakdown: %+v", res.Data.Pesanan)
	}

	// Verifikasi penjualan
	if res.Data.TotalPenjualan != 850000 {
		t.Errorf("expected total_penjualan 850000, got: %d", res.Data.TotalPenjualan)
	}

	// Verifikasi donasi
	if res.Data.Donasi.TotalTerkumpul != 100000 || res.Data.Donasi.TotalTersalurkan != 40000 || res.Data.Donasi.SaldoDonasi != 60000 {
		t.Errorf("unexpected donasi rekap: %+v", res.Data.Donasi)
	}

	// Verifikasi produk
	if res.Data.Produk.Total != 20 || res.Data.Produk.Tersedia != 10 || res.Data.Produk.Dipesan != 2 || res.Data.Produk.Terjual != 8 {
		t.Errorf("unexpected produk rekap: %+v", res.Data.Produk)
	}
}

// Otorisasi: Akses tanpa token ditolak (401 Unauthorized)
func TestAdminGetRekap_Unauthorized(t *testing.T) {
	app, _, _ := setupAdminRekapTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/rekap", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got: %d", resp.StatusCode)
	}
}

// Otorisasi: Relawan tidak berhak mengakses rekap (403 Forbidden)
func TestAdminGetRekap_ForbiddenRelawan(t *testing.T) {
	app, mock, secret := setupAdminRekapTestApp(t)
	token := createRekapTestToken(secret, "relawan-uuid-1")

	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("relawan-uuid-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("relawan-uuid-1", "Relawan Toko", models.PeranRelawan))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/rekap", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for Relawan, got: %d", resp.StatusCode)
	}
}
