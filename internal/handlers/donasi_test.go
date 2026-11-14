package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"be-comithrift/internal/handlers"
	"be-comithrift/internal/models"
	"be-comithrift/internal/services"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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
