package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"be-comithrift/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

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

	// 2. Token invalid -> 401 TOKEN_TIDAK_VALID
	reqInvalid := httptest.NewRequest(http.MethodGet, "/test-auth", nil)
	reqInvalid.Header.Set("Authorization", "Bearer invalid-token-string")
	resp, err = app.Test(reqInvalid)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 with invalid token, got status: %d", resp.StatusCode)
	}

	// 3. Token valid dengan klaim sub
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-uuid-12345",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}

	reqValid := httptest.NewRequest(http.MethodGet, "/test-auth", nil)
	reqValid.Header.Set("Authorization", "Bearer "+signedToken)
	resp, err = app.Test(reqValid)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got status: %d", resp.StatusCode)
	}
}

