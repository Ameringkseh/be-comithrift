# PLAN.md — Rencana Kerja Backend COMITHRIFT

Turunan dari Tabel 5.1 (Rencana Kerja) proposal yang sudah ditandatangani,
dipecah jadi task backend yang bisa langsung dikerjakan Antigravity satu per
satu. Setiap task mencantumkan kode kebutuhan (F-xx), kode endpoint (Tabel
3.7), dan kode skenario uji (T-xx / P-xx dari Tabel 4.1–4.2) supaya hasilnya
bisa ditelusuri balik ke proposal.

**Cara pakai:** minta Antigravity mengerjakan satu task pada satu waktu
("kerjakan Fase 2 task 2.1"), bukan seluruh fase sekaligus. Setelah task
selesai dan diverifikasi manual (jalankan request Postman-nya), centang di
`PROGRESS.md`, baru lanjut task berikutnya.

Fase 0 bukan bagian dari Tabel 5.1 — itu pekerjaan scaffolding standar yang
perlu ada sebelum Minggu 5 bisa dimulai. Fase 1–5 di bawah sejajar dengan
Minggu 5–10 di Tabel 5.1.

---

## Fase 0 — Scaffolding (sebelum Minggu 5)

Tidak tercantum eksplisit di proposal; ini prasyarat teknis agar Fase 1 bisa
jalan.

- [x] **0.1** Inisialisasi modul Go (`go mod init`), struktur folder sesuai
      §7 `AGENTS.md`.
- [x] **0.2** Setup koneksi ke PostgreSQL Supabase via GORM, baca
      connection string dari env var (`DATABASE_URL`), bukan hardcode.
      Gunakan connection string **direct/session**, bukan `anon key` —
      lihat `.agents/rules/30-security.md` §"Koneksi database".
- [x] **0.3** Health-check endpoint `GET /healthz` (disebut di Tabel 6.1
      sebagai mitigasi risiko "REST API menjadi titik kegagalan tunggal").
- [x] **0.4** Setup `.env` lokal dari `.env.example`, pastikan `.env` masuk
      `.gitignore`.
- [x] **0.5** Setup folder `postman/` dengan environment kosong
      (`comithrift.postman_environment.json`) berisi `base_url`.

## Fase 1 — Skema basis data & migrasi (≈ Minggu 3–4 & awal Minggu 5)

Sesuai Tabel 3.3 (Rancangan Data Awal) dan "Aturan integritas yang
direncanakan" di Subbab 3.4 proposal.

- [x] **1.1** Model GORM `Produk` — field sesuai Tabel 3.3: id, nama_produk,
      harga, deskripsi, ukuran, kondisi, kategori, foto (path/URL dari
      Supabase Storage), status (enum: `tersedia`/`dipesan`/`terjual`).
- [x] **1.2** Model GORM `Pesanan` — kode_pesanan (unik), tanggal,
      nama_pembeli, kontak_whatsapp, metode_serah_terima, total_belanja,
      nominal_donasi, status_pesanan (enum: `menunggu_pembayaran`/`dibayar`/
      `selesai`/`dibatalkan`), catatan, id_pencatat (nullable, FK ke
      profil_pengguna — diisi saat Admin update status).
- [x] **1.3** Model GORM `ItemPesanan` — id_pesanan, id_produk,
      harga_saat_pesan (snapshot harga di waktu pesan, bukan referensi
      harga produk saat ini).
- [x] **1.4** Model GORM `PenyaluranDonasi` — tanggal, nominal, keterangan,
      id_pencatat.
- [x] **1.5** Model GORM `ProfilPengguna` — id sama dengan id akun Supabase
      Auth, nama, peran (enum: `Admin`/`Relawan`). **Tidak menyimpan kata
      sandi** (dikelola Supabase Auth).
- [x] **1.6** Model GORM `InfoProgram` — nama_yayasan, deskripsi_program,
      info_pembayaran, kontak_whatsapp_admin. Satu baris konfigurasi.
- [x] **1.7** Migrasi: constraint harga > 0, nominal_donasi ≥ 0, nominal
      penyaluran ≥ 0 (lihat `.agents/rules/20-data-model.md`).
- [x] **1.8** Seed data uji fiktif minimal (beberapa produk, satu info_program)
      untuk dipakai testing lokal — jangan data pribadi nyata (sesuai
      batasan proyek).

## Fase 2 — Autentikasi & middleware peran (Minggu 5–7)

- [x] **2.1** Middleware verifikasi JWT Supabase Auth pada setiap request ke
      `/api/admin/*`. Lihat skill `supabase-jwt-auth-middleware`.
- [x] **2.2** Middleware pemeriksaan peran (Admin vs Relawan) berdasarkan
      `profil_pengguna.peran`, dijalankan setelah middleware JWT.
      Target hasil: **401** kalau token tidak ada/tidak valid, **403** kalau
      token valid tapi peran tidak berwenang (mendukung skenario P-04, P-05).

## Fase 3 — Endpoint produk & katalog (Minggu 5–7)

Dari Tabel 3.7.

- [ ] **3.1** `GET /api/produk` — publik, filter kategori & ukuran via query
      param, hanya tampilkan field yang relevan untuk katalog. (F-01, P-01)
- [ ] **3.2** `GET /api/produk/{id}` — publik, detail satu produk. (F-01)
- [ ] **3.3** `POST /api/admin/produk` — Relawan/Admin, terima metadata foto
      (path/URL yang sudah diupload frontend ke Storage), **bukan** file
      binary. (F-06)
- [ ] **3.4** `PATCH /api/admin/produk/{id}` — Relawan/Admin, ubah data atau
      status ketersediaan. (F-06)
- [ ] **3.5** `DELETE /api/admin/produk/{id}` — Relawan/Admin. (F-06)
- [ ] **3.6** Jalankan T-10 (kelola produk) secara manual dan skenario
      filter dari T-01.

## Fase 4 — Pesanan, checkout, status (Minggu 8–9)

Bagian paling sensitif — baca `.agents/rules/30-security.md` dan skill
`gorm-atomic-order` + `order-status-lifecycle` sebelum mulai.

- [ ] **4.1** `POST /api/pesanan` — publik, atomik: validasi keranjang tidak
      kosong, semua produk berstatus `tersedia`, nama & nomor WhatsApp &
      metode serah terima terisi, hitung total dari harga di DB, generate
      kode_pesanan unik, simpan pesanan+item dalam satu transaksi, tandai
      produk `dipesan`. (F-02, F-03, F-04, F-05, P-02, P-03)
- [ ] **4.2** `GET /api/admin/pesanan` — Admin, filter berdasarkan status.
      (F-08)
- [ ] **4.3** `PATCH /api/admin/pesanan/{id}/status` — Admin, transisi status
      + efek ke status produk (lihat skill `order-status-lifecycle` untuk
      state machine yang diusulkan). (F-08, P-06)
- [ ] **4.4** `GET /api/donasi/ringkasan` — publik, total terkumpul (dari
      pesanan `dibayar`/`selesai`), total tersalurkan, riwayat penyaluran.
      (F-10, T-13)
- [ ] **4.5** `POST /api/admin/donasi/penyaluran` — Admin, validasi nominal
      tidak melebihi donasi yang belum tersalurkan. (F-09, P-07)
- [ ] **4.6** `GET /api/admin/rekap` — Admin, jumlah pesanan per status,
      total penjualan, total donasi. (F-11, T-14)
- [ ] **4.7** Jalankan T-02 s/d T-06, T-11 s/d T-14, dan P-02 s/d P-07 secara
      manual via Postman.

## Fase 5 — Pengujian API & koleksi Postman (Minggu 10)

- [ ] **5.1** Lengkapi koleksi Postman agar mencakup seluruh baris Tabel 4.2
      (P-01–P-07), simpan di `postman/comithrift.postman_collection.json`.
- [ ] **5.2** Tulis catatan hasil tiap skenario (lulus/gagal + bukti) untuk
      dilampirkan sebagai Lampiran 7 di laporan akhir — **ini bagian laporan
      akademik, tulis manual/bersama, bukan tugas Antigravity**.
- [ ] **5.3** Review silang: minta rekan (frontend) mencoba endpoint via
      Postman, catat temuan di `PROGRESS.md`.

## Fase 6 — Pengembangan opsional (hanya jika seluruh T-/P- prioritas tinggi lulus & disetujui pembimbing)

- [ ] **6.1** Integrasi Midtrans sandbox — **jangan mulai** sebelum
      mendapat izin eksplisit dari pembimbing (lihat Subbab 1.6 proposal).
      Jika dikerjakan, akan butuh peninjauan arsitektur ulang (kemungkinan
      komponen server tambahan).

---

## Catatan untuk Antigravity

- Jangan loncat fase. Fase 4 (pesanan) butuh Fase 1–3 selesai karena
  bergantung pada model Produk dan middleware auth.
- Kalau ragu apakah sebuah detail (nama field, aturan transisi status, dsb.)
  ada di proposal atau hanya asumsi implementasi, **tandai dengan jelas**
  mana yang eksplisit dari proposal dan mana yang keputusan teknis tambahan,
  lalu tanyakan ke user kalau menyangkut hal yang akan dipresentasikan ke
  dosen pembimbing.
