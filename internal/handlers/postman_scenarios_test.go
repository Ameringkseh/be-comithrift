package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"be-comithrift/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
)

// Menjalankan simulasi skenario Postman untuk kelola produk:
// 1. GET /api/produk?kategori=Jaket&ukuran=L (P-01)
// 2. GET /api/produk/99999 (404)
// 3. POST /api/admin/produk tanpa token (P-04 -> 401)
// 4. POST /api/admin/produk dengan harga negatif (400)
// 5. PATCH /api/admin/produk/1 (200)
// 6. DELETE /api/admin/produk/1 (200)
func TestPostmanCollectionScenarios(t *testing.T) {
	app, mock, secret := setupTestApp(t)

	// Skenario 1: P-01 - List produk katalog dengan filter (expect 200)
	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE status = \$1 AND LOWER\(kategori\) = \$2 AND LOWER\(ukuran\) = \$3 ORDER BY created_at DESC`).
		WithArgs(models.StatusProdukTersedia, "jaket", "l").
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "ukuran", "kategori", "status"}).
			AddRow(1, "Jaket Denim Vintage", 85000, "L", "Jaket", "tersedia"))

	req1 := httptest.NewRequest(http.MethodGet, "/api/produk?kategori=Jaket&ukuran=L", nil)
	resp1, err := app.Test(req1)
	if err != nil || resp1.StatusCode != http.StatusOK {
		t.Fatalf("P-01 failed: expected 200, got: %d", resp1.StatusCode)
	}

	// Skenario 2: Detail produk tidak ada (expect 404)
	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE id_produk = \$1 ORDER BY "produk"\."id_produk" LIMIT \$2`).
		WithArgs(99999, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk"})) // empty

	req2 := httptest.NewRequest(http.MethodGet, "/api/produk/99999", nil)
	resp2, err := app.Test(req2)
	if err != nil || resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("Skenario 404 failed: expected 404, got: %d", resp2.StatusCode)
	}

	// Skenario 3: P-04 - Post admin tanpa token (expect 401)
	req3 := httptest.NewRequest(http.MethodPost, "/api/admin/produk", bytes.NewReader([]byte(`{"nama_produk":"Test"}`)))
	req3.Header.Set("Content-Type", "application/json")
	resp3, err := app.Test(req3)
	if err != nil || resp3.StatusCode != http.StatusUnauthorized {
		t.Fatalf("P-04 failed: expected 401, got: %d", resp3.StatusCode)
	}

	// Skenario 4: Post admin dengan harga negatif (expect 400)
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("relawan-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("relawan-1", "Relawan Test", models.PeranRelawan))

	badPayload := map[string]interface{}{
		"nama_produk": "Baju Minus",
		"harga":       -5000,
		"ukuran":      "M",
		"kondisi":     "Baik",
		"kategori":    "Kaos",
	}
	bodyBad, _ := json.Marshal(badPayload)
	req4 := httptest.NewRequest(http.MethodPost, "/api/admin/produk", bytes.NewReader(bodyBad))
	req4.Header.Set("Content-Type", "application/json")
	req4.Header.Set("Authorization", "Bearer "+createTestToken(secret, "relawan-1"))
	resp4, err := app.Test(req4)
	if err != nil || resp4.StatusCode != http.StatusBadRequest {
		t.Fatalf("Skenario 400 harga negatif failed: expected 400, got: %d", resp4.StatusCode)
	}

	// Skenario 5: PATCH produk update harga (expect 200)
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("relawan-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("relawan-1", "Relawan Test", models.PeranRelawan))

	// GetByID pertama
	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE id_produk = \$1 ORDER BY "produk"\."id_produk" LIMIT \$2`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).
			AddRow(1, "Jaket Denim", 85000, "tersedia"))

	// Update query
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "produk"`).
		WithArgs(int64(90000), sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// GetByID setelah update
	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE id_produk = \$1 ORDER BY "produk"\."id_produk" LIMIT \$2`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).
			AddRow(1, "Jaket Denim", 90000, "tersedia"))

	patchPayload := map[string]interface{}{"harga": 90000}
	bodyPatch, _ := json.Marshal(patchPayload)
	req5 := httptest.NewRequest(http.MethodPatch, "/api/admin/produk/1", bytes.NewReader(bodyPatch))
	req5.Header.Set("Content-Type", "application/json")
	req5.Header.Set("Authorization", "Bearer "+createTestToken(secret, "relawan-1"))
	resp5, err := app.Test(req5)
	if err != nil || resp5.StatusCode != http.StatusOK {
		t.Fatalf("Skenario PATCH failed: expected 200, got: %d", resp5.StatusCode)
	}

	// Skenario 6: DELETE produk (expect 200)
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-1", "Admin Test", models.PeranAdmin))

	// GetByID
	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE id_produk = \$1 ORDER BY "produk"\."id_produk" LIMIT \$2`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).
			AddRow(1, "Jaket Denim", 90000, "tersedia"))

	// Cek pesanan aktif
	mock.ExpectQuery(`SELECT count\(\*\) FROM "item_pesanan" JOIN pesanan ON pesanan\.id_pesanan = item_pesanan\.id_pesanan WHERE item_pesanan\.id_produk = \$1 AND pesanan\.status_pesanan IN \('menunggu_pembayaran', 'dibayar'\)`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// Delete exec
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "produk" WHERE "produk"\."id_produk" = \$1`).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	req6 := httptest.NewRequest(http.MethodDelete, "/api/admin/produk/1", nil)
	req6.Header.Set("Authorization", "Bearer "+createTestToken(secret, "admin-1"))
	resp6, err := app.Test(req6)
	if err != nil || resp6.StatusCode != http.StatusOK {
		t.Fatalf("Skenario DELETE failed: expected 200, got: %d", resp6.StatusCode)
	}
}
