package handlers_test

import (
	"bytes"
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

func createDonasiTestToken(secret, sub string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": sub,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func setupAdminDonasiTestApp(t *testing.T) (*fiber.App, sqlmock.Sqlmock, string) {
	secret := "test-secret-admin-donasi"
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

	svc := services.NewDonasiService(gormDB)
	handler := handlers.NewDonasiHandler(svc)

	app := fiber.New()
	api := app.Group("/api")
	api.Get("/donasi/ringkasan", handler.GetRingkasan)

	admin := api.Group("/admin")
	admin.Use(middleware.VerifyJWT(secret))
	donasiAdmin := admin.Group("/donasi")
	donasiAdmin.Use(middleware.RequireRole(gormDB, models.PeranAdmin))
	donasiAdmin.Post("/penyaluran", handler.CatatPenyaluran)

	return app, mock, secret
}

func setupDonasiTestApp(t *testing.T) (*fiber.App, sqlmock.Sqlmock) {
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

	svc := services.NewDonasiService(gormDB)
	handler := handlers.NewDonasiHandler(svc)

	app := fiber.New()
	api := app.Group("/api")
	api.Get("/donasi/ringkasan", handler.GetRingkasan)

	return app, mock
}

// Skenario F-10, T-13: Mengambil ringkasan donasi publik (status 200)
func TestGetDonasiRingkasan_Success(t *testing.T) {
	app, mock := setupDonasiTestApp(t)

	// 1. Mock SUM nominal_donasi pesanan dibayar/selesai
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(nominal_donasi\), 0\) FROM "pesanan" WHERE status_pesanan IN \(\$1,\$2\)`).
		WithArgs(models.StatusPesananDibayar, models.StatusPesananSelesai).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(150000))

	// 2. Mock SUM nominal penyaluran_donasi
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(nominal\), 0\) FROM "penyaluran_donasi"`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(50000))

	// 3. Mock SELECT riwayat penyaluran_donasi
	now := time.Now()
	mock.ExpectQuery(`SELECT \* FROM "penyaluran_donasi" ORDER BY tanggal DESC, id_penyaluran DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"id_penyaluran", "tanggal", "nominal", "keterangan", "id_pencatat"}).
			AddRow(1, now, 50000, "Penyaluran sembako panti asuhan", "admin-uuid-1"))

	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE "profil_pengguna"\."id_pengguna" = \$1`).
		WithArgs("admin-uuid-1").
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	// 4. Mock SELECT info_program
	mock.ExpectQuery(`SELECT \* FROM "info_program" ORDER BY "info_program"\."id_info" LIMIT \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id_info", "nama_yayasan", "deskripsi_program", "info_pembayaran", "kontak_whatsapp_admin"}).
			AddRow(1, "Yayasan Peduli Berbagi", "Program Tebar Cinta", "BCA 123", "08123456789"))

	req := httptest.NewRequest(http.MethodGet, "/api/donasi/ringkasan", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", resp.StatusCode)
	}

	var res struct {
		Data services.RingkasanDonasi `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&res)

	// Verifikasi perhitungan
	if res.Data.TotalTerkumpul != 150000 {
		t.Errorf("expected total_terkumpul 150000, got: %d", res.Data.TotalTerkumpul)
	}
	if res.Data.TotalTersalurkan != 50000 {
		t.Errorf("expected total_tersalurkan 50000, got: %d", res.Data.TotalTersalurkan)
	}
	if res.Data.SaldoDonasi != 100000 {
		t.Errorf("expected saldo_donasi 100000, got: %d", res.Data.SaldoDonasi)
	}
	if len(res.Data.RiwayatPenyaluran) != 1 {
		t.Errorf("expected 1 riwayat penyaluran, got: %d", len(res.Data.RiwayatPenyaluran))
	}
	if res.Data.InfoProgram == nil || res.Data.InfoProgram.NamaYayasan != "Yayasan Peduli Berbagi" {
		t.Errorf("expected info_program yayasan name, got: %+v", res.Data.InfoProgram)
	}
}

// Saat basis data belum memiliki transaksi atau penyaluran (angka 0 dan slice kosong)
func TestGetDonasiRingkasan_Empty(t *testing.T) {
	app, mock := setupDonasiTestApp(t)

	mock.ExpectQuery(`SELECT COALESCE\(SUM\(nominal_donasi\), 0\) FROM "pesanan" WHERE status_pesanan IN \(\$1,\$2\)`).
		WithArgs(models.StatusPesananDibayar, models.StatusPesananSelesai).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(0))

	mock.ExpectQuery(`SELECT COALESCE\(SUM\(nominal\), 0\) FROM "penyaluran_donasi"`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(0))

	mock.ExpectQuery(`SELECT \* FROM "penyaluran_donasi" ORDER BY tanggal DESC, id_penyaluran DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"id_penyaluran", "tanggal", "nominal", "keterangan", "id_pencatat"}))

	mock.ExpectQuery(`SELECT \* FROM "info_program" ORDER BY "info_program"\."id_info" LIMIT \$1`).
		WithArgs(1).
		WillReturnError(gorm.ErrRecordNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/donasi/ringkasan", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", resp.StatusCode)
	}

	var res struct {
		Data services.RingkasanDonasi `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&res)

	if res.Data.TotalTerkumpul != 0 || res.Data.TotalTersalurkan != 0 || res.Data.SaldoDonasi != 0 {
		t.Errorf("expected all 0s, got: %+v", res.Data)
	}
	if len(res.Data.RiwayatPenyaluran) != 0 {
		t.Errorf("expected empty riwayat, got: %d", len(res.Data.RiwayatPenyaluran))
	}
}

// Skenario P-07 (Sukses): Admin mencatat penyaluran donasi dengan nominal <= saldo donasi tersedia (status 201)
func TestAdminCatatPenyaluran_Success_P07(t *testing.T) {
	app, mock, secret := setupAdminDonasiTestApp(t)
	token := createDonasiTestToken(secret, "admin-uuid-1")

	// Middleware RequireRole
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-uuid-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	// 1. Transaction BEGIN
	mock.ExpectBegin()

	// 2. Total Terkumpul: 200000
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(nominal_donasi\), 0\) FROM "pesanan" WHERE status_pesanan IN \(\$1,\$2\)`).
		WithArgs(models.StatusPesananDibayar, models.StatusPesananSelesai).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(200000))

	// 3. Total Tersalurkan: 50000 -> Saldo tersedia = 150000
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(nominal\), 0\) FROM "penyaluran_donasi"`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(50000))

	// 4. INSERT into penyaluran_donasi (nominal 100000 <= 150000)
	mock.ExpectQuery(`INSERT INTO "penyaluran_donasi"`).
		WillReturnRows(sqlmock.NewRows([]string{"id_penyaluran"}).AddRow(1))

	// 5. Preload Pencatat
	now := time.Now()
	mock.ExpectQuery(`SELECT \* FROM "penyaluran_donasi" WHERE id_penyaluran = \$1 ORDER BY "penyaluran_donasi"\."id_penyaluran" LIMIT \$2`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_penyaluran", "tanggal", "nominal", "keterangan", "id_pencatat"}).
			AddRow(1, now, 100000, "Bantuan buku & seragam anak panti", "admin-uuid-1"))

	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE "profil_pengguna"\."id_pengguna" = \$1`).
		WithArgs("admin-uuid-1").
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	// 6. Transaction COMMIT
	mock.ExpectCommit()

	payload := map[string]interface{}{
		"nominal":    100000,
		"keterangan": "Bantuan buku & seragam anak panti",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/donasi/penyaluran", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created for P-07, got: %d", resp.StatusCode)
	}

	var res struct {
		Data models.PenyaluranDonasi `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&res)

	if res.Data.Nominal != 100000 {
		t.Errorf("expected nominal 100000, got: %d", res.Data.Nominal)
	}
	if res.Data.IDPencatat != "admin-uuid-1" {
		t.Errorf("expected id_pencatat admin-uuid-1, got: %s", res.Data.IDPencatat)
	}
}

// Skenario P-07 (Gagal): Nominal penyaluran melebihi saldo donasi tersedia (status 400 NOMINAL_MELEBIHI_SALDO)
func TestAdminCatatPenyaluran_MelebihiSaldo_P07(t *testing.T) {
	app, mock, secret := setupAdminDonasiTestApp(t)
	token := createDonasiTestToken(secret, "admin-uuid-1")

	// Middleware RequireRole
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-uuid-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	// 1. Transaction BEGIN
	mock.ExpectBegin()

	// 2. Total Terkumpul: 100000
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(nominal_donasi\), 0\) FROM "pesanan" WHERE status_pesanan IN \(\$1,\$2\)`).
		WithArgs(models.StatusPesananDibayar, models.StatusPesananSelesai).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(100000))

	// 3. Total Tersalurkan: 80000 -> Saldo tersedia = 20000
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(nominal\), 0\) FROM "penyaluran_donasi"`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(80000))

	// 4. Request nominal 50000 (> saldo 20000) -> ROLLBACK otomatis
	mock.ExpectRollback()

	payload := map[string]interface{}{
		"nominal":    50000,
		"keterangan": "Penyaluran yang melebihi saldo donasi",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/donasi/penyaluran", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for P-07 melebihi saldo, got: %d", resp.StatusCode)
	}

	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&errResp)

	if errResp.Error.Code != "NOMINAL_MELEBIHI_SALDO" {
		t.Errorf("expected error code NOMINAL_MELEBIHI_SALDO, got: %s", errResp.Error.Code)
	}
}

// Validasi input: nominal <= 0 atau keterangan kosong (status 400 VALIDASI_GAGAL)
func TestAdminCatatPenyaluran_ValidasiInput(t *testing.T) {
	app, mock, secret := setupAdminDonasiTestApp(t)
	token := createDonasiTestToken(secret, "admin-uuid-1")

	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-uuid-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	payload := map[string]interface{}{
		"nominal":    -1000,
		"keterangan": "",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/donasi/penyaluran", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for invalid input, got: %d", resp.StatusCode)
	}
}

// Otorisasi: Relawan tidak boleh mencatat penyaluran donasi (status 403 Forbidden)
func TestAdminCatatPenyaluran_ForbiddenRelawan(t *testing.T) {
	app, mock, secret := setupAdminDonasiTestApp(t)
	token := createDonasiTestToken(secret, "relawan-uuid-1")

	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("relawan-uuid-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("relawan-uuid-1", "Relawan Toko", models.PeranRelawan))

	payload := map[string]interface{}{
		"nominal":    10000,
		"keterangan": "Coba oleh relawan",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/donasi/penyaluran", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for Relawan, got: %d", resp.StatusCode)
	}
}

