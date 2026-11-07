# PROGRESS.md — Log Pengerjaan Backend COMITHRIFT

Antigravity: perbarui file ini setiap selesai satu task dari `PLAN.md`.
Jangan hapus baris lama — tambah baris baru di tabel yang sesuai.

## Status fase

| Fase | Nama                              | Status        |
| ---- | ---------------------------------- | ------------- |
| 0    | Scaffolding                        | Selesai & diuji |
| 1    | Skema basis data & migrasi         | Selesai & diuji |
| 2    | Autentikasi & middleware peran     | Selesai & diuji |
| 3    | Endpoint produk & katalog          | Selesai & diuji |
| 4    | Pesanan, checkout, status          | Belum mulai   |
| 5    | Pengujian API & koleksi Postman    | Belum mulai   |
| 6    | Opsional (Midtrans)                | Tidak dikerjakan — menunggu izin pembimbing |

(Ubah status jadi: `Belum mulai` / `Sedang dikerjakan` / `Selesai, belum diuji` / `Selesai & diuji`.)

## Log task

| Tanggal    | Task (kode) | Ringkasan perubahan                                                        | Commit  | Diuji? (skenario)                  |
| ---------- | ----------- | -------------------------------------------------------------------------- | ------- | ---------------------------------- |
| 2026-11-04 | 0.1–0.5     | Scaffolding project: Go module, config, db pooler, healthz, env, & postman | 36bb692 | go build & go vet pass             |
| 2026-11-04 | 1.1–1.6     | Definisi model GORM: Produk, Pesanan, ItemPesanan, PenyaluranDonasi, ProfilPengguna, InfoProgram | 36bb692 | go build & go vet pass             |
| 2026-11-05 | 1.7–1.8     | Migrasi GORM + SQL up/down dengan CHECK constraints, serta seed data uji fiktif | 5214d0e | go build & go vet pass             |
| 2026-11-07 | 2.1         | Middleware VerifyJWT verifikasi signature HS256 Supabase & ekstraksi user_id | c1eb863 | go test (PASS) & go build/vet pass |
| 2026-11-07 | 2.2         | Middleware RequireRole verifikasi peran (Admin vs Relawan) via DB (401 vs 403) | 5239305 | go test (PASS) & go build/vet pass |
| 2026-11-07 | 3.1–3.5     | Endpoint CRUD produk publik & admin (GET list/detail, POST, PATCH, DELETE) | 7592aeb | go test (PASS) & go build/vet pass |
| 2026-11-07 | 3.6         | Koleksi Postman request endpoint produk (sukses+gagal, P-01, P-04) & simulasi uji |         | go test (PASS) & go build/vet pass |

## Temuan review silang (dari rekan frontend / Postman)

| Tanggal | Endpoint | Masalah | Status |
| ------- | -------- | ------- | ------ |
|         |          |         |        |

## Keputusan yang masih perlu dikonfirmasi ke pembimbing

- [ ] Batas waktu pembatalan pesanan yang tidak dibayar (usulan awal di
      proposal: 24 jam, diterapkan manual oleh Admin — lihat Tabel 2.6).
- [ ] Aturan transisi status pesanan persis (lihat skill
      `order-status-lifecycle`, state machine di sana masih usulan
      implementasi, bukan kutipan literal proposal).
- [ ] (tambahkan item lain di sini seiring berjalan)
