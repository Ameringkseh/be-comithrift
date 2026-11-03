---
name: fiber-handler-pattern
description: Pola standar untuk menulis route, handler, dan service Fiber v2 di backend COMITHRIFT — bagaimana memisahkan parsing request, logika bisnis, dan pembentukan response, serta format error yang konsisten. Gunakan skill ini setiap kali membuat atau mengubah endpoint REST API (file di internal/handlers, internal/services, atau pendaftaran route di cmd/api).
---

# Pola Handler Fiber — COMITHRIFT

## Kapan dipakai

Setiap kali menulis atau mengubah: route baru, handler, service yang
dipanggil handler, atau struct request/response untuk endpoint di Tabel
3.7 proposal (lihat `PLAN.md` Fase 3–4 dan `.agents/rules/40-api-conventions.md`).

## Lapisan

```
Route (cmd/api atau internal/router)
  → Middleware (auth, role — kalau perlu)
    → Handler (internal/handlers)  — parse & validasi bentuk request, panggil service, map hasil ke response
      → Service (internal/services) — logika bisnis, transaksi DB, aturan integritas
        → Model/GORM (internal/models, internal/db)
```

Handler **tidak** langsung memanggil GORM. Semua akses database lewat
service, supaya logika bisnis (termasuk aturan dari `20-data-model.md`)
terkumpul di satu tempat dan bisa dipakai ulang/ditelusuri.

## Pola request/response

```go
// internal/handlers/produk.go
type createProdukRequest struct {
    NamaProduk string `json:"nama_produk" validate:"required"`
    Harga      int    `json:"harga" validate:"required,gt=0"`
    Deskripsi  string `json:"deskripsi"`
    Ukuran     string `json:"ukuran" validate:"required"`
    Kondisi    string `json:"kondisi" validate:"required"`
    Kategori   string `json:"kategori" validate:"required"`
    Foto       string `json:"foto"` // path/URL dari Supabase Storage, bukan file
}

func (h *ProdukHandler) Create(c *fiber.Ctx) error {
    var req createProdukRequest
    if err := c.BodyParser(&req); err != nil {
        return respondError(c, fiber.StatusBadRequest, "BODY_TIDAK_VALID", "Format request tidak bisa dibaca.")
    }
    if err := h.validator.Struct(req); err != nil {
        return respondError(c, fiber.StatusBadRequest, "VALIDASI_GAGAL", formatValidationError(err))
    }

    produk, err := h.produkService.Create(c.Context(), req.ToDomain())
    if err != nil {
        return mapServiceError(c, err) // lihat "Pemetaan error" di bawah
    }
    return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": produk})
}
```

## Pemetaan error

Service mengembalikan error bertipe (bukan string bebas), handler memetakan
ke status code HTTP yang benar sesuai `40-api-conventions.md`:

```go
// internal/services/errors.go
var (
    ErrNotFound       = errors.New("resource tidak ditemukan")
    ErrProdukTidakTersedia = errors.New("produk tidak tersedia")
    ErrValidasi       = errors.New("validasi gagal")
)

func mapServiceError(c *fiber.Ctx, err error) error {
    switch {
    case errors.Is(err, ErrNotFound):
        return respondError(c, fiber.StatusNotFound, "TIDAK_DITEMUKAN", err.Error())
    case errors.Is(err, ErrProdukTidakTersedia):
        return respondError(c, fiber.StatusBadRequest, "PRODUK_TIDAK_TERSEDIA", err.Error())
    case errors.Is(err, ErrValidasi):
        return respondError(c, fiber.StatusBadRequest, "VALIDASI_GAGAL", err.Error())
    default:
        log.Error().Err(err).Msg("unhandled service error")
        return respondError(c, fiber.StatusInternalServerError, "KESALAHAN_SERVER", "Terjadi kesalahan pada server.")
    }
}
```

Jangan biarkan error mentah dari GORM/driver database sampai ke response
body — selalu lewat `mapServiceError` supaya pesan yang keluar ramah dan
tidak membocorkan detail internal (lihat `.agents/rules/30-security.md`).

## Checklist cepat sebelum anggap handler selesai

- Field request divalidasi (tag `validate` + pengecekan bisnis di service).
- Tidak ada akses GORM langsung di file handler.
- Error dipetakan lewat helper terpusat, bukan `c.SendString(err.Error())`.
- Response sukses memakai bentuk `{"data": ...}` sesuai konvensi.
