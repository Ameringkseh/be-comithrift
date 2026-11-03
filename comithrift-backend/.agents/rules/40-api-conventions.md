# Rule: Konvensi REST API

Sumber: Tabel 3.7 (Rancangan Endpoint REST API) dan Tabel 4.2 (Rencana
Pengujian REST API / Postman) di proposal. Daftar endpoint lengkap:

| Metode | Endpoint                               | Akses           |
| ------ | --------------------------------------- | --------------- |
| GET    | `/api/produk`                           | Publik          |
| GET    | `/api/produk/{id}`                      | Publik          |
| POST   | `/api/pesanan`                          | Publik          |
| GET    | `/api/donasi/ringkasan`                 | Publik          |
| POST   | `/api/admin/produk`                     | Relawan, Admin  |
| PATCH  | `/api/admin/produk/{id}`                | Relawan, Admin  |
| DELETE | `/api/admin/produk/{id}`                | Relawan, Admin  |
| GET    | `/api/admin/pesanan`                    | Admin           |
| PATCH  | `/api/admin/pesanan/{id}/status`        | Admin           |
| POST   | `/api/admin/donasi/penyaluran`          | Admin           |
| GET    | `/api/admin/rekap`                      | Admin           |

Plus `GET /healthz` (Fase 0, tidak ada di proposal — murni infrastruktur).

Jangan menambah endpoint di luar daftar ini tanpa update `PLAN.md` dan
konfirmasi user.

## Kode status HTTP (mengikuti Tabel 4.2 secara literal)

- **200** — sukses, GET/PATCH yang tidak membuat resource baru.
- **201** — sukses, `POST /api/pesanan` berhasil membuat pesanan baru
  (skenario P-02).
- **400** — input tidak valid, atau pelanggaran aturan bisnis yang bisa
  diperbaiki user (produk tidak tersedia saat checkout — P-03; nominal
  penyaluran melebihi saldo — P-07).
- **401** — tidak ada token / token tidak valid di endpoint yang butuh
  autentikasi (P-04).
- **403** — token valid tapi peran tidak berwenang untuk aksi itu (P-05).
- **404** — resource dengan id yang diminta tidak ada.
- **500** — error tak terduga di server; log detail di server, jangan
  bocorkan detail internal (stack trace, pesan driver DB) ke response body.

## Bentuk response

Konsisten di semua endpoint — pilih satu pola di awal dan pertahankan:

```json
// Sukses
{ "data": { ... } }

// Sukses, list
{ "data": [ ... ], "meta": { "total": 12 } }

// Gagal
{ "error": { "message": "Pesan yang bisa dibaca pengguna/Postman", "code": "PRODUK_TIDAK_TERSEDIA" } }
```

`code` di error pakai `SCREAMING_SNAKE_CASE` pendek yang stabil (dipakai
frontend untuk logic, `message` untuk ditampilkan apa adanya atau untuk
dicek manual di Postman). Field/`message` berbahasa Indonesia, konsisten
dengan istilah di proposal (contoh error untuk donasi tidak valid, bukan
"invalid input" generik).

## Struktur kode (Fiber)

```
internal/
  handlers/      # terima request, validasi bentuk, panggil service, bentuk response
  services/      # logika bisnis (hitung total, aturan transisi status, dsb.)
  models/        # struct GORM
  middleware/    # auth JWT, cek peran, CORS, logger
  db/            # koneksi GORM, migrasi
  config/        # baca & validasi env var saat startup
cmd/api/main.go  # wiring: baca config, connect db, daftarkan route, Listen
```

Handler setipis mungkin: parse request → panggil service → map hasil/error
ke response HTTP. Logika bisnis (termasuk transaksi GORM) tinggal di
service, supaya bisa ditelusuri/ditest terpisah dari detail HTTP.

## Penamaan route Fiber

Group berdasarkan prefix supaya middleware jelas terlihat menempel di mana:

```go
api := app.Group("/api")
api.Get("/produk", h.ListProduk)
api.Get("/produk/:id", h.GetProduk)
api.Post("/pesanan", h.CreatePesanan)
api.Get("/donasi/ringkasan", h.RingkasanDonasi)

admin := api.Group("/admin", middleware.VerifyJWT(), middleware.RequireRole("Admin", "Relawan"))
admin.Post("/produk", h.CreateProduk)
admin.Patch("/produk/:id", h.UpdateProduk)
admin.Delete("/produk/:id", h.DeleteProduk)

adminOnly := api.Group("/admin", middleware.VerifyJWT(), middleware.RequireRole("Admin"))
adminOnly.Get("/pesanan", h.ListPesanan)
adminOnly.Patch("/pesanan/:id/status", h.UpdateStatusPesanan)
adminOnly.Post("/donasi/penyaluran", h.CatatPenyaluran)
adminOnly.Get("/rekap", h.Rekap)
```

(Contoh ilustratif — sesuaikan nama fungsi/struktur sesuai kode yang
benar-benar ditulis, tapi pola pengelompokan middleware per role ini wajib
dipertahankan supaya endpoint Relawan tidak pernah tanpa sengaja terbuka
untuk peran lain atau sebaliknya.)
