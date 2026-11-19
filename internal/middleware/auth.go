package middleware

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

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

type jwkKey struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

var (
	jwksMu     sync.RWMutex
	jwksCache  = make(map[string]*ecdsa.PublicKey)
	httpClient = &http.Client{Timeout: 10 * time.Second}
)

// SetJWKSCacheForTest menginjeksikan public key untuk pengujian unit offline tanpa internet
func SetJWKSCacheForTest(kid string, pubKey *ecdsa.PublicKey) {
	jwksMu.Lock()
	defer jwksMu.Unlock()
	jwksCache[kid] = pubKey
}

func parseECPublicKey(xStr, yStr string) (*ecdsa.PublicKey, error) {
	cleanX := strings.TrimRight(xStr, "=")
	cleanY := strings.TrimRight(yStr, "=")

	xBytes, err := base64.RawURLEncoding.DecodeString(cleanX)
	if err != nil {
		return nil, fmt.Errorf("gagal decode x JWK: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(cleanY)
	if err != nil {
		return nil, fmt.Errorf("gagal decode y JWK: %w", err)
	}

	curve := elliptic.P256()
	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)

	if !curve.IsOnCurve(x, y) {
		return nil, fmt.Errorf("koordinat EC JWK tidak berada pada kurva P-256")
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}, nil
}

func fetchJWKSPublicKey(kid, jwksURL string) (*ecdsa.PublicKey, error) {
	jwksMu.RLock()
	if key, ok := jwksCache[kid]; ok {
		jwksMu.RUnlock()
		return key, nil
	}
	jwksMu.RUnlock()

	if jwksURL == "" {
		return nil, fmt.Errorf("URL JWKS tidak dikonfigurasi dan kunci %s tidak ada di cache", kid)
	}

	resp, err := httpClient.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil JWKS dari %s: %w", jwksURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("endpoint JWKS mengembalikan status %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("gagal decode JSON JWKS: %w", err)
	}

	jwksMu.Lock()
	defer jwksMu.Unlock()
	for _, k := range jwks.Keys {
		if k.Kty == "EC" && k.Crv == "P-256" {
			pubKey, err := parseECPublicKey(k.X, k.Y)
			if err == nil {
				jwksCache[k.Kid] = pubKey
			}
		}
	}

	if key, ok := jwksCache[kid]; ok {
		return key, nil
	}

	return nil, fmt.Errorf("kunci public untuk kid %s tidak ditemukan di JWKS", kid)
}

func resolveJWKSURL(rawToken, projectURL string) string {
	if projectURL != "" {
		return strings.TrimRight(projectURL, "/") + "/auth/v1/.well-known/jwks.json"
	}
	parser := jwt.NewParser()
	unverified, _, _ := parser.ParseUnverified(rawToken, jwt.MapClaims{})
	if unverified != nil {
		if claims, ok := unverified.Claims.(jwt.MapClaims); ok {
			if iss, ok := claims["iss"].(string); ok && iss != "" {
				return strings.TrimRight(iss, "/") + "/.well-known/jwks.json"
			}
		}
	}
	return ""
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

// VerifyJWT memverifikasi token JWT Supabase Auth pada request.
// Mendukung algoritma ES256 (asimetris JWKS standar Supabase modern) dan HS256 (simetris secret legacy/test).
// Mengembalikan 401 Unauthorized jika token tidak ada, tidak valid, atau kedaluwarsa (P-04).
// Sumber: .agents/skills/supabase-jwt-auth-middleware/SKILL.md & .agents/rules/30-security.md
func VerifyJWT(secret string, projectURL ...string) fiber.Handler {
	secretBytes := parseJWTSecret(secret)
	var defaultProjectURL string
	if len(projectURL) > 0 {
		defaultProjectURL = projectURL[0]
	}

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
			// 1. Dukung algoritma ES256 (Supabase Auth ECC P-256 via JWKS)
			if t.Method.Alg() == "ES256" {
				kid, _ := t.Header["kid"].(string)
				if kid == "" {
					return nil, fmt.Errorf("token ES256 tidak memuat kid di header")
				}
				jwksURL := resolveJWKSURL(raw, defaultProjectURL)
				return fetchJWKSPublicKey(kid, jwksURL)
			}

			// 2. Dukung algoritma HS256 (Supabase Auth symmetric secret legacy / test)
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); ok {
				return secretBytes, nil
			}

			return nil, fmt.Errorf("algoritma token tidak didukung: %v", t.Header["alg"])
		})

		// Fallback untuk HS256: jika verifikasi dengan secretBytes gagal, coba dengan raw []byte(secret)
		if (err != nil || !token.Valid) && strings.Contains(raw, ".") {
			parser := jwt.NewParser()
			if unverified, _, _ := parser.ParseUnverified(raw, jwt.MapClaims{}); unverified != nil && unverified.Method.Alg() == "HS256" {
				token, err = jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
					return []byte(secret), nil
				})
			}
		}

		if err != nil || !token.Valid {
			log.Printf("[VerifyJWT] Token tidak valid: %v", err)
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
