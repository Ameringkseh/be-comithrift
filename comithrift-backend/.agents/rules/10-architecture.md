# Rule: Arsitektur

## Prinsip inti (Subbab 3.1–3.2 proposal)

> Seluruh operasi baca/tulis data produk, pesanan, item pesanan, dan donasi
> dikirim antarmuka ke REST API backend (Go Fiber v2 dan GORM) melalui
> HTTPS, kemudian REST API mengakses basis data PostgreSQL Supabase
> menggunakan GORM. Antarmuka hanya mengakses Supabase secara langsung
> untuk login (Supabase Auth) dan unggah/ambil foto (Supabase Storage).

Konsekuensi konkret untuk kode:

- Tidak ada satu pun handler yang menerima Supabase client key (anon/publishable)
  dari frontend untuk query data produk/pesanan/donasi. Backend adalah
  satu-satunya yang punya akses ke tabel-tabel itu.
- Backend terhubung ke Postgres pakai **koneksi langsung** (role yang
  bypass RLS, biasanya `postgres`), bukan lewat Supabase REST/PostgREST,
  dan bukan pakai anon key. Lihat `30-security.md`.
- Backend tidak pernah memanggil Supabase Storage API untuk upload/ambil
  foto — itu dilakukan langsung oleh frontend. Backend hanya menyimpan
  string path/URL yang dikirim frontend sebagai field `foto` pada produk.
- Backend tidak memverifikasi password — itu tugas Supabase Auth. Backend
  hanya memverifikasi **token JWT** yang sudah diterbitkan Supabase Auth.

## Alur login panel (Admin/Relawan)

1. Frontend memanggil Supabase Auth langsung (sign in with password) →
   dapat JWT.
2. Frontend menyertakan JWT itu di header `Authorization: Bearer <token>`
   pada setiap request ke `/api/admin/*`.
3. Middleware backend memverifikasi JWT (lihat skill
   `supabase-jwt-auth-middleware`), ambil `sub` (user id) dari klaim token,
   lookup `profil_pengguna` berdasarkan id itu untuk tahu perannya.
4. Middleware peran mengizinkan/menolak berdasarkan peran tsb.

Backend **tidak pernah** menerbitkan atau memperpanjang token sendiri.

## Alur checkout (pembeli, anonim)

1. Frontend kumpulkan item keranjang + donasi di sisi klien (hanya untuk
   pratinjau tampilan).
2. Frontend `POST /api/pesanan` dengan daftar id_produk + jumlah per item,
   nominal_donasi usulan, data pembeli.
3. Backend **mengabaikan** harga/total yang (kalau ada) dikirim klien,
   hitung ulang semuanya dari harga di database, lakukan dalam satu
   transaksi (lihat skill `gorm-atomic-order`).
4. Backend mengembalikan data pesanan (termasuk `kode_pesanan` dan total
   final) supaya frontend bisa menyusun pesan WhatsApp.

## Lapisan pertahanan berlapis

Urutan pertahanan dari luar ke dalam — **jangan hilangkan salah satu** demi
"menyederhanakan":

1. Middleware JWT + peran di Fiber (lapisan utama, wajib untuk `/api/admin/*`).
2. Validasi input di handler/service (tipe data, rentang nilai, field wajib).
3. Perhitungan ulang nilai sensitif (total, status) di server, tidak pernah
   dipercaya dari klien.
4. Row Level Security di Postgres (lapisan tambahan — defense in depth,
   bukan pengganti #1).

## Referensi endpoint

Daftar endpoint lengkap (Tabel 3.7 proposal) ada di `PLAN.md` Fase 3–4, per
task. Jangan menambah endpoint di luar daftar itu tanpa konfirmasi user.
