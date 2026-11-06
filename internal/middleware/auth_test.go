package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"be-comithrift/internal/middleware"
	"be-comithrift/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func createTestToken(secret, sub string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": sub,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func TestVerifyJWT(t *testing.T) {
	secret := "super-secret-test-key-123"

	app := fiber.New()
	app.Get("/test-auth", middleware.VerifyJWT(secret), func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(string)
		return c.JSON(fiber.Map{"user_id": userID})
	})

	// 1. Tanpa Authorization header -> 401 TOKEN_TIDAK_ADA
	reqNoHeader := httptest.NewRequest(http.MethodGet, "/test-auth", nil)
	resp, err := app.Test(reqNoHeader)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without header, got status: %d", resp.StatusCode)
	}

	var errResp errorResponse
	json.NewDecoder(resp.Body).Decode(&errResp)
	if errResp.Error.Code != "TOKEN_TIDAK_ADA" {
		t.Errorf("expected code TOKEN_TIDAK_ADA, got: %s", errResp.Error.Code)
	}

	// 2. Token invalid -> 401 TOKEN_TIDAK_VALID
	reqInvalid := httptest.NewRequest(http.MethodGet, "/test-auth", nil)
	reqInvalid.Header.Set("Authorization", "Bearer invalid-token-string")
	resp, err = app.Test(reqInvalid)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 with invalid token, got status: %d", resp.StatusCode)
	}

	// 3. Token valid dengan klaim sub -> 200
	validToken := createTestToken(secret, "user-uuid-12345")
	reqValid := httptest.NewRequest(http.MethodGet, "/test-auth", nil)
	reqValid.Header.Set("Authorization", "Bearer "+validToken)
	resp, err = app.Test(reqValid)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got status: %d", resp.StatusCode)
	}
}

func TestRequireRole(t *testing.T) {
	secret := "super-secret-test-key-123"

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqlDB.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm with sqlmock: %v", err)
	}

	app := fiber.New()

	// Route khusus Admin saja (misalnya /admin/pesanan)
	app.Get("/admin/pesanan", middleware.VerifyJWT(secret), middleware.RequireRole(gormDB, models.PeranAdmin), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	// Route untuk Relawan dan Admin (misalnya /admin/produk)
	app.Get("/admin/produk", middleware.VerifyJWT(secret), middleware.RequireRole(gormDB, models.PeranRelawan, models.PeranAdmin), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	// Skenario 1 (P-04): Request tanpa token ke route terproteksi -> 401 Unauthorized
	reqNoToken := httptest.NewRequest(http.MethodGet, "/admin/pesanan", nil)
	resp, _ := app.Test(reqNoToken)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for request without token, got: %d", resp.StatusCode)
	}

	// Skenario 2 (P-05): Relawan akses route khusus Admin -> 403 Forbidden
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("relawan-uuid-2", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("relawan-uuid-2", "Relawan Test", models.PeranRelawan))

	relawanToken := createTestToken(secret, "relawan-uuid-2")
	reqRelawanToAdmin := httptest.NewRequest(http.MethodGet, "/admin/pesanan", nil)
	reqRelawanToAdmin.Header.Set("Authorization", "Bearer "+relawanToken)
	resp, _ = app.Test(reqRelawanToAdmin)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 when Relawan accesses Admin-only route, got: %d", resp.StatusCode)
	}
	var errResp errorResponse
	json.NewDecoder(resp.Body).Decode(&errResp)
	if errResp.Error.Code != "AKSES_DITOLAK" {
		t.Errorf("expected code AKSES_DITOLAK, got: %s", errResp.Error.Code)
	}

	// Skenario 3: Akun valid tapi tidak terdaftar di database profil_pengguna -> 403 Forbidden
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("unknown-uuid-999", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"})) // empty rows

	unknownUserToken := createTestToken(secret, "unknown-uuid-999")
	reqUnknown := httptest.NewRequest(http.MethodGet, "/admin/pesanan", nil)
	reqUnknown.Header.Set("Authorization", "Bearer "+unknownUserToken)
	resp, _ = app.Test(reqUnknown)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for unknown profile, got: %d", resp.StatusCode)
	}
	json.NewDecoder(resp.Body).Decode(&errResp)
	if errResp.Error.Code != "PERAN_TIDAK_DIKENALI" {
		t.Errorf("expected code PERAN_TIDAK_DIKENALI, got: %s", errResp.Error.Code)
	}

	// Skenario 4: Relawan akses route Relawan/Admin (/admin/produk) -> 200 OK
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("relawan-uuid-2", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("relawan-uuid-2", "Relawan Test", models.PeranRelawan))

	reqRelawanToProduk := httptest.NewRequest(http.MethodGet, "/admin/produk", nil)
	reqRelawanToProduk.Header.Set("Authorization", "Bearer "+relawanToken)
	resp, _ = app.Test(reqRelawanToProduk)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 when Relawan accesses /admin/produk, got: %d", resp.StatusCode)
	}

	// Skenario 5: Admin akses route Admin (/admin/pesanan) -> 200 OK
	mock.ExpectQuery(`SELECT \* FROM "profil_pengguna" WHERE id_pengguna = \$1 ORDER BY "profil_pengguna"\."id_pengguna" LIMIT \$2`).
		WithArgs("admin-uuid-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id_pengguna", "nama", "peran"}).
			AddRow("admin-uuid-1", "Admin Test", models.PeranAdmin))

	adminToken := createTestToken(secret, "admin-uuid-1")
	reqAdmin := httptest.NewRequest(http.MethodGet, "/admin/pesanan", nil)
	reqAdmin.Header.Set("Authorization", "Bearer "+adminToken)
	resp, _ = app.Test(reqAdmin)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 when Admin accesses /admin/pesanan, got: %d", resp.StatusCode)
	}
}
