# PROGRESS.md — Log Pengerjaan Backend COMITHRIFT

Antigravity: perbarui file ini setiap selesai satu task dari `PLAN.md`.
Jangan hapus baris lama — tambah baris baru di tabel yang sesuai.

## Status fase

| Fase | Nama                              | Status        |
| ---- | ---------------------------------- | ------------- |
| 0    | Scaffolding                        | Belum mulai   |
| 1    | Skema basis data & migrasi         | Belum mulai   |
| 2    | Autentikasi & middleware peran     | Belum mulai   |
| 3    | Endpoint produk & katalog          | Belum mulai   |
| 4    | Pesanan, checkout, status          | Belum mulai   |
| 5    | Pengujian API & koleksi Postman    | Belum mulai   |
| 6    | Opsional (Midtrans)                | Tidak dikerjakan — menunggu izin pembimbing |

(Ubah status jadi: `Belum mulai` / `Sedang dikerjakan` / `Selesai, belum diuji` / `Selesai & diuji`.)

## Log task

| Tanggal | Task (kode) | Ringkasan perubahan | Commit | Diuji? (skenario) |
| ------- | ----------- | -------------------- | ------ | ------------------ |
|         |             |                       |        |                     |

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
