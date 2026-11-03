---
name: supabase-jwt-auth-middleware
description: Cara memverifikasi token JWT Supabase Auth di middleware Fiber dan mengecek peran (Admin/Relawan) dari tabel profil_pengguna, untuk melindungi endpoint /api/admin/*. Gunakan skill ini saat membuat atau mengubah middleware autentikasi/otorisasi, atau saat endpoint admin mengembalikan status yang salah (401/403 tertukar atau tidak muncul).
---

# Middleware Auth & Peran — COMITHRIFT

## Alur (lihat juga `.agents/rules/10-architecture.md` §"Alur login panel")

1. Frontend login lewat Supabase Auth langsung → dapat JWT.
2. Frontend kirim `Authorization: Bearer <jwt>` ke endpoint `/api/admin/*`.
3. Middleware #1 (`VerifyJWT`): verifikasi tanda tangan & masa berlaku token
   pakai `SUPABASE_JWT_SECRET`, ambil `sub` (user id) dari klaim, simpan ke
   `fiber.Ctx` locals.
4. Middleware #2 (`RequireRole`): lookup `profil_pengguna` berdasarkan user
   id itu, cek `peran` ada di daftar yang diizinkan untuk route tsb.

Dua middleware ini **terpisah** (bukan satu fungsi besar) supaya mudah
disusun ulang per route — lihat contoh grouping di
`.agents/rules/40-api-conventions.md`.

## Verifikasi JWT

Supabase Auth menandatangani JWT dengan algoritma simetris (HS256) memakai
JWT secret project (bukan dengan anon key, dan bukan dengan pasangan kunci
publik/privat kecuali project dikonfigurasi JWKS/asymmetric — cek dashboard
Supabase project untuk memastikan metode yang dipakai sebelum
implementasi). Pola umum untuk HS256:

```go
// internal/middleware/auth.go
func VerifyJWT(secret string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        header := c.Get("Authorization")
        if !strings.HasPrefix(header, "Bearer ") {
            return respondError(c, fiber.StatusUnauthorized, "TOKEN_TIDAK_ADA", "Token autentikasi diperlukan.")
        }
        raw := strings.TrimPrefix(header, "Bearer ")

        token, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
            if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("algoritma token tidak didukung")
            }
            return []byte(secret), nil
        })
        if err != nil || !token.Valid {
            return respondError(c, fiber.StatusUnauthorized, "TOKEN_TIDAK_VALID", "Token autentikasi tidak valid atau kedaluwarsa.")
        }

        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            return respondError(c, fiber.StatusUnauthorized, "TOKEN_TIDAK_VALID", "Token autentikasi tidak valid.")
        }
        userID, _ := claims["sub"].(string)
        if userID == "" {
            return respondError(c, fiber.StatusUnauthorized, "TOKEN_TIDAK_VALID", "Token tidak memuat identitas pengguna.")
        }

        c.Locals("user_id", userID)
        return c.Next()
    }
}
```

## Pemeriksaan peran

```go
func RequireRole(db *gorm.DB, allowed ...string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        userID := c.Locals("user_id").(string)

        var profil models.ProfilPengguna
        if err := db.First(&profil, "id_pengguna = ?", userID).Error; err != nil {
            return respondError(c, fiber.StatusForbidden, "PERAN_TIDAK_DIKENALI", "Akun tidak terdaftar sebagai pengelola.")
        }

        for _, role := range allowed {
            if profil.Peran == role {
                c.Locals("peran", profil.Peran)
                return c.Next()
            }
        }
        return respondError(c, fiber.StatusForbidden, "AKSES_DITOLAK", "Peran Anda tidak berwenang untuk aksi ini.")
    }
}
```

## Hal yang wajib benar (dicek lewat Postman P-04/P-05)

- Tidak ada token sama sekali atau token invalid/kedaluwarsa → **401**,
  bukan 403 dan bukan 500.
- Token valid tapi peran tidak sesuai (mis. Relawan memanggil
  `/api/admin/pesanan`) → **403**, bukan 401.
- Jangan pernah mempercayai peran yang (kalau ada) dikirim di body/header
  request — peran **selalu** di-lookup dari `profil_pengguna` di database
  berdasarkan `sub` token, tidak pernah dari klaim JWT lain atau input
  klien.
- `SUPABASE_JWT_SECRET` dibaca dari env var (lihat `.env.example`), tidak
  pernah hardcode di kode.

## Catatan RLS (baca juga `.agents/rules/30-security.md`)

Middleware ini adalah **lapisan otorisasi utama** karena koneksi backend
ke Postgres bypass RLS. RLS di Supabase tetap diaktifkan sebagai lapisan
tambahan untuk jalur lain (kalau suatu saat ada akses langsung dari
frontend ke tabel via anon key) — middleware ini tidak boleh dianggap
"tidak penting karena ada RLS", justru sebaliknya.
