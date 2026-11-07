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

func createTestToken(secret, sub string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": sub,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func setupTestApp(t *testing.T) (*fiber.App, sqlmock.Sqlmock, string) {
	secret := "test-secret-key-123"
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

	svc := services.NewProdukService(gormDB)
	handler := handlers.NewProdukHandler(svc)

	app := fiber.New()
	api := app.Group("/api")
	api.Get("/produk", handler.GetList)
	api.Get("/produk/:id", handler.GetByID)

	admin := api.Group("/admin")
	admin.Use(middleware.VerifyJWT(secret))
	adminProduk := admin.Group("/produk")
	adminProduk.Use(middleware.RequireRole(gormDB, models.PeranRelawan, models.PeranAdmin))
	adminProduk.Post("/", handler.Create)
	adminProduk.Patch("/:id", handler.Update)
	adminProduk.Delete("/:id", handler.Delete)

	return app, mock, secret
}

func TestGetProdukList(t *testing.T) {
	app, mock, _ := setupTestApp(t)

	// Mock query produk tersedia
	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE status = \$1 ORDER BY created_at DESC`).
		WithArgs(models.StatusProdukTersedia).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).
			AddRow(1, "Jaket Denim", 75000, "tersedia"))

	req := httptest.NewRequest(http.MethodGet, "/api/produk", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got: %d", resp.StatusCode)
	}
}

func TestGetProdukByID(t *testing.T) {
	app, mock, _ := setupTestApp(t)

	// Mock query produk detail
	mock.ExpectQuery(`SELECT \* FROM "produk" WHERE id_produk = \$1 ORDER BY "produk"\."id_produk" LIMIT \$2`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).
			AddRow(1, "Jaket Denim", 75000, "tersedia"))

	req := httptest.NewRequest(http.MethodGet, "/api/produk/1", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got: %d", resp.StatusCode)
	}
}

func TestAdminCreateProduk(t *testing.T) {
	app, mock, secret := setupTestApp(t)

	// Mock profile lookup untuk middleware RequireRole
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("relawan-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("relawan-1", "Relawan Fiktif", models.PeranRelawan))

	// Mock create produk
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "produk"`).
		WillReturnRows(sqlmock.NewRows([]string{"id_produk"}).AddRow(10))
	mock.ExpectCommit()

	payload := map[string]interface{}{
		"nama_produk": "Celana Chino",
		"harga":       50000,
		"ukuran":      "32",
		"kondisi":     "Baik",
		"kategori":    "Celana",
		"foto":        "https://example.com/foto.jpg",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/produk", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+createTestToken(secret, "relawan-1"))

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got: %d", resp.StatusCode)
	}
}

