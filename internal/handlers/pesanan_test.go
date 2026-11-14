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

func createPesananTestToken(secret, sub string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": sub,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func setupAdminPesananTestApp(t *testing.T) (*fiber.App, sqlmock.Sqlmock, string) {
	secret := "test-secret-admin-pesanan"
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

	svc := services.NewPesananService(gormDB)
	handler := handlers.NewPesananHandler(svc)

	app := fiber.New()
	api := app.Group("/api")
	api.Post("/pesanan", handler.Create)

	admin := api.Group("/admin")
	admin.Use(middleware.VerifyJWT(secret))
	pesananAdmin := admin.Group("/pesanan")
	pesananAdmin.Use(middleware.RequireRole(gormDB, models.PeranAdmin))
	pesananAdmin.Get("", handler.GetList)
	pesananAdmin.Get("/", handler.GetList)
	pesananAdmin.Patch("/:id/status", handler.UpdateStatus)

	return app, mock, secret
}

func setupPesananTestApp(t *testing.T) (*fiber.App, sqlmock.Sqlmock) {
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

	svc := services.NewPesananService(gormDB)
	handler := handlers.NewPesananHandler(svc)

	app := fiber.New()
	app.Post("/api/pesanan", handler.Create)

	return app, mock
}

// Skenario P-02: Membuat pesanan dengan data valid (Status 201)
func TestCreatePesananSuccess_P02(t *testing.T) {
	app, mock := setupPesananTestApp(t)

	// 1. Transaction BEGIN
	mock.ExpectBegin()

	// 2. SELECT FOR UPDATE produk 1 (tersedia, harga 85000)
	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE id_produk = \$1 ORDER BY "produk"\."id_produk" LIMIT \$2 FOR UPDATE`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).
			AddRow(1, "Jaket Denim Vintage", 85000, models.StatusProdukTersedia))

	// 3. Update status produk menjadi 'dipesan'
	mock.ExpectExec(`UPDATE "produk"`).
		WithArgs(models.StatusProdukDipesan, sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 4. INSERT into pesanan
	mock.ExpectQuery(`INSERT INTO "pesanan"`).
		WillReturnRows(sqlmock.NewRows([]string{"id_pesanan"}).AddRow(100))

	// 5. INSERT into item_pesanan
	mock.ExpectQuery(`INSERT INTO "item_pesanan"`).
		WillReturnRows(sqlmock.NewRows([]string{"id_item"}).AddRow(200))

	// 6. Transaction COMMIT
	mock.ExpectCommit()

	payload := map[string]interface{}{
		"nama_pembeli":        "Budi Santoso",
		"kontak_whatsapp":     "081234567890",
		"metode_serah_terima": "Ambil di tempat",
		"nominal_donasi":      15000,
		"catatan":             "Ambil sabtu sore",
		"items": []map[string]interface{}{
			{"id_produk": 1},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/pesanan", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created for P-02, got: %d", resp.StatusCode)
	}

	var res struct {
		Data models.Pesanan `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&res)

	// Verifikasi total_belanja dihitung dari harga DB (85000)
	if res.Data.TotalBelanja != 85000 {
		t.Errorf("expected total_belanja 85000, got: %d", res.Data.TotalBelanja)
	}
	if res.Data.StatusPesanan != models.StatusPesananMenungguPembayaran {
		t.Errorf("expected status_pesanan menunggu_pembayaran, got: %s", res.Data.StatusPesanan)
	}
}

// Skenario P-03: Membuat pesanan untuk produk yang sudah dipesan / tidak tersedia (Status 400 PRODUK_TIDAK_TERSEDIA)
func TestCreatePesananUnavailable_P03(t *testing.T) {
	app, mock := setupPesananTestApp(t)

	// 1. Transaction BEGIN
	mock.ExpectBegin()

	// 2. SELECT FOR UPDATE produk 1 (status: dipesan)
	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE id_produk = \$1 ORDER BY "produk"\."id_produk" LIMIT \$2 FOR UPDATE`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).
			AddRow(1, "Jaket Denim Vintage", 85000, models.StatusProdukDipesan))

	// 3. Seluruh transaksi harus ROLLBACK otomatis
	mock.ExpectRollback()

	payload := map[string]interface{}{
		"nama_pembeli":        "Andi Permana",
		"kontak_whatsapp":     "089876543210",
		"metode_serah_terima": "Kirim kurir",
		"nominal_donasi":      0,
		"items": []map[string]interface{}{
			{"id_produk": 1},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/pesanan", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for P-03, got: %d", resp.StatusCode)
	}

	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&errResp)
	if errResp.Error.Code != "PRODUK_TIDAK_TERSEDIA" {
		t.Errorf("expected error code PRODUK_TIDAK_TERSEDIA, got: %s", errResp.Error.Code)
	}
}

// TestAdminGetPesananList_Success: Admin berhasil mengambil daftar seluruh pesanan
func TestAdminGetPesananList_Success(t *testing.T) {
	app, mock, secret := setupAdminPesananTestApp(t)
	token := createPesananTestToken(secret, "admin-uuid-1")

	// 1. Mock query cek profil pengguna untuk middleware RequireRole
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-uuid-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	// 2. Mock query get list pesanan
	now := time.Now()
	mock.ExpectQuery(`SELECT \* FROM "pesanan" ORDER BY tanggal DESC, id_pesanan DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"id_pesanan", "kode_pesanan", "tanggal", "nama_pembeli", "kontak_whatsapp", "metode_serah_terima", "total_belanja", "nominal_donasi", "status_pesanan"}).
			AddRow(1, "CMT-20261114-AA11", now, "Pembeli Satu", "0811111111", "Ambil di tempat", 85000, 10000, models.StatusPesananMenungguPembayaran))

	// 3. Mock preloads
	mock.ExpectQuery(`SELECT \* FROM "item_pesanan" WHERE "item_pesanan"\."id_pesanan" = \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id_item_pesanan", "id_pesanan", "id_produk", "harga_saat_pesan"}).
			AddRow(10, 1, 100, 85000))

	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE "produk"\."id_produk" = \$1`).
		WithArgs(100).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).
			AddRow(100, "Jaket Kulit", 85000, models.StatusProdukDipesan))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/pesanan", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", resp.StatusCode)
	}

	var res struct {
		Data []models.Pesanan `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	json.NewDecoder(resp.Body).Decode(&res)

	if res.Meta.Total != 1 {
		t.Errorf("expected meta.total 1, got: %d", res.Meta.Total)
	}
	if len(res.Data) != 1 || res.Data[0].KodePesanan != "CMT-20261114-AA11" {
		t.Errorf("unexpected data returned: %+v", res.Data)
	}
}

// TestAdminGetPesananList_WithStatusFilter: Filter status_pesanan yang valid
func TestAdminGetPesananList_WithStatusFilter(t *testing.T) {
	app, mock, secret := setupAdminPesananTestApp(t)
	token := createPesananTestToken(secret, "admin-uuid-1")

	// 1. Mock query cek profil pengguna untuk middleware RequireRole
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-uuid-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	// 2. Mock query get list pesanan dengan filter status
	now := time.Now()
	mock.ExpectQuery(`SELECT \* FROM "pesanan" WHERE status_pesanan = \$1 ORDER BY tanggal DESC, id_pesanan DESC`).
		WithArgs(models.StatusPesananDibayar).
		WillReturnRows(sqlmock.NewRows([]string{"id_pesanan", "kode_pesanan", "tanggal", "nama_pembeli", "kontak_whatsapp", "metode_serah_terima", "total_belanja", "nominal_donasi", "status_pesanan"}).
			AddRow(2, "CMT-20261114-BB22", now, "Pembeli Dua", "0822222222", "Kirim kurir", 120000, 20000, models.StatusPesananDibayar))

	// 3. Mock preloads
	mock.ExpectQuery(`SELECT \* FROM "item_pesanan" WHERE "item_pesanan"\."id_pesanan" = \$1`).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"id_item_pesanan", "id_pesanan", "id_produk", "harga_saat_pesan"}).
			AddRow(20, 2, 200, 120000))

	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE "produk"\."id_produk" = \$1`).
		WithArgs(200).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).
			AddRow(200, "Kemeja Flanel", 120000, models.StatusProdukTerjual))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/pesanan?status=dibayar", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", resp.StatusCode)
	}

	var res struct {
		Data []models.Pesanan `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	json.NewDecoder(resp.Body).Decode(&res)

	if res.Meta.Total != 1 || res.Data[0].StatusPesanan != models.StatusPesananDibayar {
		t.Errorf("unexpected response: %+v", res)
	}
}

// TestAdminGetPesananList_InvalidStatus: Filter status yang tidak valid menghasilkan 400 Bad Request
func TestAdminGetPesananList_InvalidStatus(t *testing.T) {
	app, mock, secret := setupAdminPesananTestApp(t)
	token := createPesananTestToken(secret, "admin-uuid-1")

	// Mock middleware
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-uuid-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/pesanan?status=status_ngaco", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got: %d", resp.StatusCode)
	}
}

// TestAdminGetPesananList_ForbiddenForRelawan: Relawan tidak berhak mengakses (403 Forbidden)
func TestAdminGetPesananList_ForbiddenForRelawan(t *testing.T) {
	app, mock, secret := setupAdminPesananTestApp(t)
	token := createPesananTestToken(secret, "relawan-uuid-1")

	// Mock middleware return peran Relawan
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("relawan-uuid-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("relawan-uuid-1", "Relawan Toko", models.PeranRelawan))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/pesanan", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for Relawan, got: %d", resp.StatusCode)
	}
}

// Skenario P-06: Verifikasi pembayaran pesanan (menunggu_pembayaran -> dibayar)
// Efek: pesanan menjadi dibayar, id_pencatat diisi id admin, produk menjadi 'terjual' (Status 200)
func TestAdminUpdateStatusPesanan_Dibayar_Success_P06(t *testing.T) {
	app, mock, secret := setupAdminPesananTestApp(t)
	token := createPesananTestToken(secret, "admin-uuid-1")

	// Middleware RequireRole
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-uuid-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	// 1. Transaction BEGIN
	mock.ExpectBegin()

	// 2. SELECT FOR UPDATE pesanan
	now := time.Now()
	mock.ExpectQuery(`SELECT \* FROM "pesanan" WHERE id_pesanan = \$1 ORDER BY "pesanan"\."id_pesanan" LIMIT \$2 FOR UPDATE`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pesanan", "kode_pesanan", "tanggal", "nama_pembeli", "kontak_whatsapp", "metode_serah_terima", "total_belanja", "nominal_donasi", "status_pesanan"}).
			AddRow(1, "CMT-20261114-AA11", now, "Pembeli Satu", "0811111111", "Ambil di tempat", 85000, 10000, models.StatusPesananMenungguPembayaran))

	// 3. Update status_pesanan & id_pencatat
	mock.ExpectExec(`UPDATE "pesanan"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 4. Pluck id_produk from item_pesanan
	mock.ExpectQuery(`SELECT "id_produk" FROM "item_pesanan" WHERE id_pesanan = \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk"}).AddRow(100))

	// 5. Update status produk menjadi 'terjual'
	mock.ExpectExec(`UPDATE "produk"`).
		WithArgs(models.StatusProdukTerjual, sqlmock.AnyArg(), 100).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 6. Reload pesanan dengan preload
	adminID := "admin-uuid-1"
	mock.ExpectQuery(`SELECT \* FROM "pesanan" WHERE id_pesanan = \$1 ORDER BY "pesanan"\."id_pesanan" LIMIT \$2`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pesanan", "kode_pesanan", "tanggal", "nama_pembeli", "kontak_whatsapp", "metode_serah_terima", "total_belanja", "nominal_donasi", "status_pesanan", "id_pencatat"}).
			AddRow(1, "CMT-20261114-AA11", now, "Pembeli Satu", "0811111111", "Ambil di tempat", 85000, 10000, models.StatusPesananDibayar, adminID))

	mock.ExpectQuery(`SELECT \* FROM "item_pesanan" WHERE "item_pesanan"\."id_pesanan" = \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id_item_pesanan", "id_pesanan", "id_produk", "harga_saat_pesan"}).
			AddRow(10, 1, 100, 85000))

	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE "produk"\."id_produk" = \$1`).
		WithArgs(100).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).
			AddRow(100, "Jaket Kulit", 85000, models.StatusProdukTerjual))

	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE "profil_pengguna"\."id_pengguna" = \$1`).
		WithArgs("admin-uuid-1").
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	// 7. Transaction COMMIT
	mock.ExpectCommit()

	payload := map[string]interface{}{
		"status_pesanan": "dibayar",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/api/admin/pesanan/1/status", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for P-06, got: %d", resp.StatusCode)
	}

	var res struct {
		Data models.Pesanan `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&res)

	if res.Data.StatusPesanan != models.StatusPesananDibayar {
		t.Errorf("expected status_pesanan dibayar, got: %s", res.Data.StatusPesanan)
	}
	if res.Data.IDPencatat == nil || *res.Data.IDPencatat != "admin-uuid-1" {
		t.Errorf("expected id_pencatat admin-uuid-1, got: %v", res.Data.IDPencatat)
	}
}

// Batalkan pesanan (menunggu_pembayaran -> dibatalkan): produk dilepas kembali menjadi 'tersedia'
func TestAdminUpdateStatusPesanan_Dibatalkan_Success(t *testing.T) {
	app, mock, secret := setupAdminPesananTestApp(t)
	token := createPesananTestToken(secret, "admin-uuid-1")

	// Middleware RequireRole
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-uuid-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	// 1. Transaction BEGIN
	mock.ExpectBegin()

	// 2. SELECT FOR UPDATE pesanan
	now := time.Now()
	mock.ExpectQuery(`SELECT \* FROM "pesanan" WHERE id_pesanan = \$1 ORDER BY "pesanan"\."id_pesanan" LIMIT \$2 FOR UPDATE`).
		WithArgs(2, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pesanan", "kode_pesanan", "tanggal", "nama_pembeli", "kontak_whatsapp", "metode_serah_terima", "total_belanja", "nominal_donasi", "status_pesanan"}).
			AddRow(2, "CMT-20261114-BB22", now, "Pembeli Dua", "0822222222", "Kirim kurir", 50000, 0, models.StatusPesananMenungguPembayaran))

	// 3. Update status_pesanan & id_pencatat
	mock.ExpectExec(`UPDATE "pesanan"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 4. Pluck id_produk from item_pesanan
	mock.ExpectQuery(`SELECT "id_produk" FROM "item_pesanan" WHERE id_pesanan = \$1`).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk"}).AddRow(200))

	// 5. Update status produk menjadi 'tersedia' (dilepas)
	mock.ExpectExec(`UPDATE "produk"`).
		WithArgs(models.StatusProdukTersedia, sqlmock.AnyArg(), 200).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 6. Reload pesanan dengan preload
	adminID := "admin-uuid-1"
	mock.ExpectQuery(`SELECT \* FROM "pesanan" WHERE id_pesanan = \$1 ORDER BY "pesanan"\."id_pesanan" LIMIT \$2`).
		WithArgs(2, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pesanan", "kode_pesanan", "tanggal", "nama_pembeli", "kontak_whatsapp", "metode_serah_terima", "total_belanja", "nominal_donasi", "status_pesanan", "id_pencatat"}).
			AddRow(2, "CMT-20261114-BB22", now, "Pembeli Dua", "0822222222", "Kirim kurir", 50000, 0, models.StatusPesananDibatalkan, adminID))

	mock.ExpectQuery(`SELECT \* FROM "item_pesanan" WHERE "item_pesanan"\."id_pesanan" = \$1`).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"id_item_pesanan", "id_pesanan", "id_produk", "harga_saat_pesan"}).
			AddRow(20, 2, 200, 50000))

	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE "produk"\."id_produk" = \$1`).
		WithArgs(200).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).
			AddRow(200, "Kaus Polos", 50000, models.StatusProdukTersedia))

	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE "profil_pengguna"\."id_pengguna" = \$1`).
		WithArgs("admin-uuid-1").
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	// 7. Transaction COMMIT
	mock.ExpectCommit()

	payload := map[string]interface{}{
		"status_pesanan": "dibatalkan",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/api/admin/pesanan/2/status", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", resp.StatusCode)
	}

	var res struct {
		Data models.Pesanan `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&res)

	if res.Data.StatusPesanan != models.StatusPesananDibatalkan {
		t.Errorf("expected status_pesanan dibatalkan, got: %s", res.Data.StatusPesanan)
	}
}

// Transisi dari status akhir (selesai -> dibayar) harus ditolak 400 TRANSISI_TIDAK_VALID
func TestAdminUpdateStatusPesanan_InvalidTransition_Rejected(t *testing.T) {
	app, mock, secret := setupAdminPesananTestApp(t)
	token := createPesananTestToken(secret, "admin-uuid-1")

	// Middleware RequireRole
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-uuid-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	// 1. Transaction BEGIN
	mock.ExpectBegin()

	// 2. Pesanan sudah selesai
	now := time.Now()
	mock.ExpectQuery(`SELECT \* FROM "pesanan" WHERE id_pesanan = \$1 ORDER BY "pesanan"\."id_pesanan" LIMIT \$2 FOR UPDATE`).
		WithArgs(3, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pesanan", "kode_pesanan", "tanggal", "nama_pembeli", "status_pesanan"}).
			AddRow(3, "CMT-20261114-CC33", now, "Pembeli Tiga", models.StatusPesananSelesai))

	// 3. Rollback otomatis karena transisi tidak diizinkan
	mock.ExpectRollback()

	payload := map[string]interface{}{
		"status_pesanan": "dibayar",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/api/admin/pesanan/3/status", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for invalid transition, got: %d", resp.StatusCode)
	}

	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&errResp)
	if errResp.Error.Code != "TRANSISI_TIDAK_VALID" {
		t.Errorf("expected error code TRANSISI_TIDAK_VALID, got: %s", errResp.Error.Code)
	}
}

// Status enum tidak dikenal menghasilkan 400 VALIDASI_GAGAL
func TestAdminUpdateStatusPesanan_InvalidStatusEnum(t *testing.T) {
	app, mock, secret := setupAdminPesananTestApp(t)
	token := createPesananTestToken(secret, "admin-uuid-1")

	// Middleware RequireRole
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-uuid-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	payload := map[string]interface{}{
		"status_pesanan": "status_asal_asalan",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/api/admin/pesanan/1/status", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got: %d", resp.StatusCode)
	}

	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&errResp)
	if errResp.Error.Code != "VALIDASI_GAGAL" {
		t.Errorf("expected error code VALIDASI_GAGAL, got: %s", errResp.Error.Code)
	}
}

// Transisi status: dibayar -> selesai (produk tetap terjual, status_pesanan selesai)
func TestAdminUpdateStatusPesanan_Selesai_Success(t *testing.T) {
	app, mock, secret := setupAdminPesananTestApp(t)
	token := createPesananTestToken(secret, "admin-uuid-1")

	// Middleware RequireRole
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-uuid-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	// 1. Transaction BEGIN
	mock.ExpectBegin()

	// 2. SELECT FOR UPDATE pesanan (status: dibayar)
	now := time.Now()
	mock.ExpectQuery(`SELECT \* FROM "pesanan" WHERE id_pesanan = \$1 ORDER BY "pesanan"\."id_pesanan" LIMIT \$2 FOR UPDATE`).
		WithArgs(4, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pesanan", "kode_pesanan", "tanggal", "nama_pembeli", "kontak_whatsapp", "metode_serah_terima", "total_belanja", "nominal_donasi", "status_pesanan"}).
			AddRow(4, "CMT-20261114-DD44", now, "Pembeli Empat", "0844444444", "Ambil di tempat", 90000, 10000, models.StatusPesananDibayar))

	// 3. Update status_pesanan & id_pencatat
	mock.ExpectExec(`UPDATE "pesanan"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 4. Pluck id_produk from item_pesanan
	mock.ExpectQuery(`SELECT "id_produk" FROM "item_pesanan" WHERE id_pesanan = \$1`).
		WithArgs(4).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk"}).AddRow(400))

	// Untuk status 'selesai', produk tetap 'terjual' (tidak ada update ke tabel produk)

	// 5. Reload pesanan dengan preload
	adminID := "admin-uuid-1"
	mock.ExpectQuery(`SELECT \* FROM "pesanan" WHERE id_pesanan = \$1 ORDER BY "pesanan"\."id_pesanan" LIMIT \$2`).
		WithArgs(4, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pesanan", "kode_pesanan", "tanggal", "nama_pembeli", "kontak_whatsapp", "metode_serah_terima", "total_belanja", "nominal_donasi", "status_pesanan", "id_pencatat"}).
			AddRow(4, "CMT-20261114-DD44", now, "Pembeli Empat", "0844444444", "Ambil di tempat", 90000, 10000, models.StatusPesananSelesai, adminID))

	mock.ExpectQuery(`SELECT \* FROM "item_pesanan" WHERE "item_pesanan"\."id_pesanan" = \$1`).
		WithArgs(4).
		WillReturnRows(sqlmock.NewRows([]string{"id_item_pesanan", "id_pesanan", "id_produk", "harga_saat_pesan"}).
			AddRow(40, 4, 400, 90000))

	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE "produk"\."id_produk" = \$1`).
		WithArgs(400).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).
			AddRow(400, "Celana Chino", 90000, models.StatusProdukTerjual))

	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE "profil_pengguna"\."id_pengguna" = \$1`).
		WithArgs("admin-uuid-1").
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Utama", models.PeranAdmin))

	// 6. Transaction COMMIT
	mock.ExpectCommit()

	payload := map[string]interface{}{
		"status_pesanan": "selesai",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/api/admin/pesanan/4/status", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for selesai, got: %d", resp.StatusCode)
	}

	var res struct {
		Data models.Pesanan `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&res)

	if res.Data.StatusPesanan != models.StatusPesananSelesai {
		t.Errorf("expected status_pesanan selesai, got: %s", res.Data.StatusPesanan)
	}
}



