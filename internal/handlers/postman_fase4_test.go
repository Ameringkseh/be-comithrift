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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupFullTestApp(t *testing.T) (*fiber.App, sqlmock.Sqlmock, string) {
	secret := "test-full-postman-secret"
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

	app := fiber.New()

	produkService := services.NewProdukService(gormDB)
	produkHandler := handlers.NewProdukHandler(produkService)

	pesananService := services.NewPesananService(gormDB)
	pesananHandler := handlers.NewPesananHandler(pesananService)

	donasiService := services.NewDonasiService(gormDB)
	donasiHandler := handlers.NewDonasiHandler(donasiService)

	rekapService := services.NewRekapService(gormDB)
	rekapHandler := handlers.NewRekapHandler(rekapService)

	api := app.Group("/api")
	api.Get("/produk", produkHandler.GetList)
	api.Get("/produk/:id", produkHandler.GetByID)
	api.Post("/pesanan", pesananHandler.Create)
	api.Get("/donasi/ringkasan", donasiHandler.GetRingkasan)

	admin := api.Group("/admin")
	admin.Use(middleware.VerifyJWT(secret))

	produkAdmin := admin.Group("/produk")
	produkAdmin.Use(middleware.RequireRole(gormDB, models.PeranRelawan, models.PeranAdmin))
	produkAdmin.Post("", produkHandler.Create)
	produkAdmin.Post("/", produkHandler.Create)

	pesananAdmin := admin.Group("/pesanan")
	pesananAdmin.Use(middleware.RequireRole(gormDB, models.PeranAdmin))
	pesananAdmin.Get("", pesananHandler.GetList)
	pesananAdmin.Get("/", pesananHandler.GetList)
	pesananAdmin.Patch("/:id/status", pesananHandler.UpdateStatus)

	donasiAdmin := admin.Group("/donasi")
	donasiAdmin.Use(middleware.RequireRole(gormDB, models.PeranAdmin))
	donasiAdmin.Post("/penyaluran", donasiHandler.CatatPenyaluran)

	admin.Get("/rekap", middleware.RequireRole(gormDB, models.PeranAdmin), rekapHandler.GetRekap)

	return app, mock, secret
}

// Menjalankan simulasi skenario Postman P-02 s/d P-07 dan T-11 s/d T-14 secara menyeluruh
func TestPostmanFase4Scenarios(t *testing.T) {
	app, mock, secret := setupFullTestApp(t)
	tokenAdmin := createTestToken(secret, "admin-1")
	tokenRelawan := createTestToken(secret, "relawan-1")

	// ==========================================
	// Skenario P-02: Membuat pesanan valid (expect 201)
	// ==========================================
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE id_produk = \$1 ORDER BY "produk"\."id_produk" LIMIT \$2 FOR UPDATE`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).
			AddRow(1, "Jaket Denim", 85000, models.StatusProdukTersedia))
	mock.ExpectExec(`UPDATE "produk"`).
		WithArgs(models.StatusProdukDipesan, sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`INSERT INTO "pesanan"`).
		WillReturnRows(sqlmock.NewRows([]string{"id_pesanan"}).AddRow(10))
	mock.ExpectQuery(`INSERT INTO "item_pesanan"`).
		WillReturnRows(sqlmock.NewRows([]string{"id_item"}).AddRow(20))
	mock.ExpectCommit()

	payloadP02 := map[string]interface{}{
		"nama_pembeli":        "Budi Santoso",
		"kontak_whatsapp":     "081234567890",
		"metode_serah_terima": "Ambil di tempat",
		"nominal_donasi":      15000,
		"catatan":             "Ambil sabtu",
		"items": []map[string]interface{}{
			{"id_produk": 1},
		},
	}
	bodyP02, _ := json.Marshal(payloadP02)
	reqP02 := httptest.NewRequest(http.MethodPost, "/api/pesanan", bytes.NewReader(bodyP02))
	reqP02.Header.Set("Content-Type", "application/json")
	respP02, err := app.Test(reqP02)
	if err != nil || respP02.StatusCode != http.StatusCreated {
		t.Fatalf("P-02 failed: expected 201, got: %d", respP02.StatusCode)
	}

	// ==========================================
	// Skenario P-03: Checkout produk tidak tersedia (expect 400)
	// ==========================================
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE id_produk = \$1 ORDER BY "produk"\."id_produk" LIMIT \$2 FOR UPDATE`).
		WithArgs(2, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).
			AddRow(2, "Kaos Vintage", 45000, models.StatusProdukDipesan))
	mock.ExpectRollback()

	payloadP03 := map[string]interface{}{
		"nama_pembeli":        "Andi",
		"kontak_whatsapp":     "089876543210",
		"metode_serah_terima": "Kirim kurir",
		"items": []map[string]interface{}{
			{"id_produk": 2},
		},
	}
	bodyP03, _ := json.Marshal(payloadP03)
	reqP03 := httptest.NewRequest(http.MethodPost, "/api/pesanan", bytes.NewReader(bodyP03))
	reqP03.Header.Set("Content-Type", "application/json")
	respP03, err := app.Test(reqP03)
	if err != nil || respP03.StatusCode != http.StatusBadRequest {
		t.Fatalf("P-03 failed: expected 400, got: %d", respP03.StatusCode)
	}

	// ==========================================
	// Skenario T-11: Mengambil daftar pesanan oleh Admin (expect 200)
	// ==========================================
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-1", "Admin Utama", models.PeranAdmin))
	now := time.Now()
	mock.ExpectQuery(`SELECT \* FROM "pesanan" ORDER BY tanggal DESC, id_pesanan DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"id_pesanan", "kode_pesanan", "tanggal", "nama_pembeli", "kontak_whatsapp", "metode_serah_terima", "total_belanja", "nominal_donasi", "status_pesanan"}).
			AddRow(10, "CMT-20261119-AA11", now, "Budi", "081234567890", "Ambil di tempat", 85000, 15000, models.StatusPesananMenungguPembayaran))
	mock.ExpectQuery(`SELECT \* FROM "item_pesanan" WHERE "item_pesanan"\."id_pesanan" = \$1`).
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{"id_item_pesanan", "id_pesanan", "id_produk", "harga_saat_pesan"}).
			AddRow(20, 10, 1, 85000))
	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE "produk"\."id_produk" = \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).
			AddRow(1, "Jaket Denim", 85000, models.StatusProdukDipesan))

	reqT11 := httptest.NewRequest(http.MethodGet, "/api/admin/pesanan", nil)
	reqT11.Header.Set("Authorization", "Bearer "+tokenAdmin)
	respT11, err := app.Test(reqT11)
	if err != nil || respT11.StatusCode != http.StatusOK {
		t.Fatalf("T-11 failed: expected 200, got: %d", respT11.StatusCode)
	}

	// ==========================================
	// Skenario P-05: Mengakses pesanan admin dengan token Relawan (expect 403)
	// ==========================================
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("relawan-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("relawan-1", "Relawan Toko", models.PeranRelawan))

	reqP05 := httptest.NewRequest(http.MethodGet, "/api/admin/pesanan", nil)
	reqP05.Header.Set("Authorization", "Bearer "+tokenRelawan)
	respP05, err := app.Test(reqP05)
	if err != nil || respP05.StatusCode != http.StatusForbidden {
		t.Fatalf("P-05 failed: expected 403, got: %d", respP05.StatusCode)
	}

	// ==========================================
	// Skenario P-06 (T-12): Verifikasi bayar pesanan (expect 200)
	// ==========================================
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-1", "Admin Utama", models.PeranAdmin))
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "pesanan" WHERE id_pesanan = \$1 ORDER BY "pesanan"\."id_pesanan" LIMIT \$2 FOR UPDATE`).
		WithArgs(10, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pesanan", "kode_pesanan", "tanggal", "nama_pembeli", "kontak_whatsapp", "metode_serah_terima", "total_belanja", "nominal_donasi", "status_pesanan"}).
			AddRow(10, "CMT-20261119-AA11", now, "Budi", "081234567890", "Ambil di tempat", 85000, 15000, models.StatusPesananMenungguPembayaran))
	mock.ExpectExec(`UPDATE "pesanan"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT "id_produk" FROM "item_pesanan" WHERE id_pesanan = \$1`).
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk"}).AddRow(1))
	mock.ExpectExec(`UPDATE "produk"`).
		WithArgs(models.StatusProdukTerjual, sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT \* FROM "pesanan" WHERE id_pesanan = \$1 ORDER BY "pesanan"\."id_pesanan" LIMIT \$2`).
		WithArgs(10, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pesanan", "kode_pesanan", "tanggal", "nama_pembeli", "kontak_whatsapp", "metode_serah_terima", "total_belanja", "nominal_donasi", "status_pesanan", "id_pencatat"}).
			AddRow(10, "CMT-20261119-AA11", now, "Budi", "081234567890", "Ambil di tempat", 85000, 15000, models.StatusPesananDibayar, "admin-1"))
	mock.ExpectQuery(`SELECT \* FROM "item_pesanan" WHERE "item_pesanan"\."id_pesanan" = \$1`).
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{"id_item_pesanan", "id_pesanan", "id_produk", "harga_saat_pesan"}).
			AddRow(20, 10, 1, 85000))
	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE "produk"\."id_produk" = \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).
			AddRow(1, "Jaket Denim", 85000, models.StatusProdukTerjual))
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE "profil_pengguna"\."id_pengguna" = \$1`).
		WithArgs("admin-1").
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-1", "Admin Utama", models.PeranAdmin))
	mock.ExpectCommit()

	payloadP06 := map[string]interface{}{"status_pesanan": "dibayar"}
	bodyP06, _ := json.Marshal(payloadP06)
	reqP06 := httptest.NewRequest(http.MethodPatch, "/api/admin/pesanan/10/status", bytes.NewReader(bodyP06))
	reqP06.Header.Set("Authorization", "Bearer "+tokenAdmin)
	reqP06.Header.Set("Content-Type", "application/json")
	respP06, err := app.Test(reqP06)
	if err != nil || respP06.StatusCode != http.StatusOK {
		t.Fatalf("P-06 failed: expected 200, got: %d", respP06.StatusCode)
	}

	// ==========================================
	// Skenario T-13: Ringkasan donasi publik (expect 200)
	// ==========================================
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(nominal_donasi\), 0\) FROM "pesanan" WHERE status_pesanan IN \(\$1,\$2\)`).
		WithArgs(models.StatusPesananDibayar, models.StatusPesananSelesai).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(15000))
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(nominal\), 0\) FROM "penyaluran_donasi"`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(0))
	mock.ExpectQuery(`SELECT \* FROM "penyaluran_donasi" ORDER BY tanggal DESC, id_penyaluran DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"id_penyaluran", "tanggal", "nominal", "keterangan", "id_pencatat"}))
	mock.ExpectQuery(`SELECT \* FROM "info_program" ORDER BY "info_program"\."id_info" LIMIT \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id_info", "nama_yayasan", "deskripsi_program", "info_pembayaran", "kontak_whatsapp_admin"}).
			AddRow(1, "Yayasan Peduli Berbagi", "Tebar Cinta", "BCA 123", "08123"))

	reqT13 := httptest.NewRequest(http.MethodGet, "/api/donasi/ringkasan", nil)
	respT13, err := app.Test(reqT13)
	if err != nil || respT13.StatusCode != http.StatusOK {
		t.Fatalf("T-13 failed: expected 200, got: %d", respT13.StatusCode)
	}

	// ==========================================
	// Skenario P-07: Penyaluran melebihi saldo (expect 400)
	// ==========================================
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-1", "Admin Utama", models.PeranAdmin))
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(nominal_donasi\), 0\) FROM "pesanan" WHERE status_pesanan IN \(\$1,\$2\)`).
		WithArgs(models.StatusPesananDibayar, models.StatusPesananSelesai).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(15000))
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(nominal\), 0\) FROM "penyaluran_donasi"`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(0))
	mock.ExpectRollback()

	payloadP07 := map[string]interface{}{
		"nominal":    50000, // melebihi saldo 15000
		"keterangan": "Penyaluran lebih",
	}
	bodyP07, _ := json.Marshal(payloadP07)
	reqP07 := httptest.NewRequest(http.MethodPost, "/api/admin/donasi/penyaluran", bytes.NewReader(bodyP07))
	reqP07.Header.Set("Authorization", "Bearer "+tokenAdmin)
	reqP07.Header.Set("Content-Type", "application/json")
	respP07, err := app.Test(reqP07)
	if err != nil || respP07.StatusCode != http.StatusBadRequest {
		t.Fatalf("P-07 failed: expected 400, got: %d", respP07.StatusCode)
	}

	// ==========================================
	// Skenario T-14: Rekap data Admin (expect 200)
	// ==========================================
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-1", "Admin Utama", models.PeranAdmin))
	mock.ExpectQuery(`SELECT status_pesanan, count\(\*\) as count FROM "pesanan" GROUP BY "status_pesanan"`).
		WillReturnRows(sqlmock.NewRows([]string{"status_pesanan", "count"}).
			AddRow(models.StatusPesananDibayar, 1))
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(total_belanja\), 0\) FROM "pesanan" WHERE status_pesanan IN \(\$1,\$2\)`).
		WithArgs(models.StatusPesananDibayar, models.StatusPesananSelesai).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(85000))
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(nominal_donasi\), 0\) FROM "pesanan" WHERE status_pesanan IN \(\$1,\$2\)`).
		WithArgs(models.StatusPesananDibayar, models.StatusPesananSelesai).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(15000))
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(nominal\), 0\) FROM "penyaluran_donasi"`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(0))
	mock.ExpectQuery(`SELECT status, count\(\*\) as count FROM "produk" GROUP BY "status"`).
		WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).
			AddRow(models.StatusProdukTerjual, 1))

	reqT14 := httptest.NewRequest(http.MethodGet, "/api/admin/rekap", nil)
	reqT14.Header.Set("Authorization", "Bearer "+tokenAdmin)
	respT14, err := app.Test(reqT14)
	if err != nil || respT14.StatusCode != http.StatusOK {
		t.Fatalf("T-14 failed: expected 200, got: %d", respT14.StatusCode)
	}
}
