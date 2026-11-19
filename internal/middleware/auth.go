package middleware

import (
	"encoding/base64"
	"fmt"
	"strings"

	"be-comithrift/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// RespondError helper konsisten untuk format error JSON di middleware & handler
// Format: { "error": { "message": "...", "code": "..." } }
// Sumber: .agents/rules/40-api-conventions.md
func RespondError(c *fiber.Ctx, statusCode int, code string, message string) error {
	return c.Status(statusCode).JSON(fiber.Map{
		"error": fiber.Map{
			"code":    code,
			"message": message,
		},
	})
}

// parseJWTSecret mengonversi secret string ke byte array.
// Supabase Cloud JWT secret berformat Base64 (64 bytes/512-bit, 88 karakter).
// Jika format base64 valid, gunakan hasil decode; jika bukan (misal plain string di unit test), gunakan []byte(secret).
func parseJWTSecret(secret string) []byte {
	trimmed := strings.TrimSpace(secret)
	if len(trimmed) == 88 || strings.HasSuffix(trimmed, "=") {
		if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(decoded) > 0 {
			return decoded
		}
	}
	if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(decoded) >= 32 {
		return decoded
	}
	return []byte(secret)
}

// VerifyJWT memverifikasi token JWT Supabase Auth pada request
// Mengembalikan 401 Unauthorized jika token tidak ada, tidak valid, atau kedaluwarsa (P-04).
// Sumber: .agents/skills/supabase-jwt-auth-middleware/SKILL.md & .agents/rules/30-security.md
func VerifyJWT(secret string) fiber.Handler {
	secretBytes := parseJWTSecret(secret)

	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return RespondError(c, fiber.StatusUnauthorized, "TOKEN_TIDAK_ADA", "Token autentikasi diperlukan.")
		}
		raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if raw == "" {
			return RespondError(c, fiber.StatusUnauthorized, "TOKEN_TIDAK_ADA", "Token autentikasi diperlukan.")
		}

		token, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("algoritma token tidak didukung: %v", t.Header["alg"])
			}
			return secretBytes, nil
		})

		if err != nil || !token.Valid {
			return RespondError(c, fiber.StatusUnauthorized, "TOKEN_TIDAK_VALID", "Token autentikasi tidak valid atau kedaluwarsa.")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return RespondError(c, fiber.StatusUnauthorized, "TOKEN_TIDAK_VALID", "Token autentikasi tidak valid.")
		}

		userID, _ := claims["sub"].(string)
		if userID == "" {
			return RespondError(c, fiber.StatusUnauthorized, "TOKEN_TIDAK_VALID", "Token tidak memuat identitas pengguna.")
		}

		// Simpan user_id ke c.Locals untuk digunakan middleware berikutnya dan handler
		c.Locals("user_id", userID)
		return c.Next()
	}
}

// RequireRole memeriksa peran pengguna (Admin vs Relawan) berdasarkan tabel profil_pengguna di database.
// Dijalankan setelah VerifyJWT. Mengembalikan 403 Forbidden jika token valid namun peran tidak berwenang (P-05).
// Sumber: .agents/skills/supabase-jwt-auth-middleware/SKILL.md & .agents/rules/40-api-conventions.md
func RequireRole(db *gorm.DB, allowed ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		val := c.Locals("user_id")
		userID, ok := val.(string)
		if !ok || userID == "" {
			return RespondError(c, fiber.StatusUnauthorized, "TOKEN_TIDAK_VALID", "Identitas pengguna tidak ditemukan.")
		}

		var profil models.ProfilPengguna
		if err := db.First(&profil, "id_pengguna = ?", userID).Error; err != nil {
			return RespondError(c, fiber.StatusForbidden, "PERAN_TIDAK_DIKENALI", "Akun tidak terdaftar sebagai pengelola.")
		}

		for _, role := range allowed {
			if profil.Peran == role {
				c.Locals("peran", profil.Peran)
				return c.Next()
			}
		}

		return RespondError(c, fiber.StatusForbidden, "AKSES_DITOLAK", "Peran Anda tidak berwenang untuk aksi ini.")
	}
}
