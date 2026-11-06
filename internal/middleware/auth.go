package middleware

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
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

// VerifyJWT memverifikasi token JWT Supabase Auth pada request
// Sumber: .agents/skills/supabase-jwt-auth-middleware/SKILL.md & .agents/rules/30-security.md
func VerifyJWT(secret string) fiber.Handler {
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
			return []byte(secret), nil
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

