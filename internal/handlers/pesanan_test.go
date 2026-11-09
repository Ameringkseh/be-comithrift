package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"be-comithrift/internal/handlers"
	"be-comithrift/internal/models"
	"be-comithrift/internal/services"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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
