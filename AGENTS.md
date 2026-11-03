# AGENTS.md — COMITHRIFT Backend

Dibaca otomatis oleh Antigravity (dan tool lain yang kompatibel: Cursor, Claude
Code, Codex) di setiap task di workspace ini. Jangan dihapus atau dipindah.

## 1. Konteks proyek

COMITHRIFT adalah platform web pemasaran dan penjualan pakaian bekas yang
dipadukan dengan program donasi "Tebar Cinta". Ini adalah proyek tugas Proyek 3
(D4 Teknik Informatika, ULBI) dikerjakan oleh **dua mahasiswa**:

- **Rafli Mochamad Ramadhan (714240059) — backend.** Ini kamu/user repo ini.
  Ruang lingkup tercatat di Lembar Pernyataan Kontribusi proposal: skema basis
  data & migrasi GORM, REST API backend (Go, Fiber v2), autentikasi & peran
  (integrasi JWT Supabase), panel Relawan dan Admin (produk, foto, pesanan,
  penyaluran donasi, rekap), serta pengujian endpoint dengan Postman.
- **Aditya Firmansyah (714240015) — frontend.** Di luar scope workspace ini.

Spesifikasi lengkap dan sumber kebenaran ada di proposal yang **sudah
disetujui dan ditandatangani**: `docs/PROPOSAL_PROYEK_3_COMITHRIFT_signed.pdf`
(salin file itu ke folder `docs/` sebelum mulai kerja — lihat §7). Jangan
mengubah ruang lingkup, entitas, atau endpoint di luar apa yang ada di proposal
tanpa konfirmasi eksplisit dari user di chat.

## 2. Batas kerja (PENTING)

- **Hanya kerjakan backend.** Direktori kerja utama: `cmd/`, `internal/`,
  `migrations/`, `postman/`. Jangan membuat atau mengubah kode frontend
  (HTML/CSS/JS di `public/` atau sejenisnya) kecuali diminta eksplisit —
  itu tanggung jawab Aditya.
- **Jangan ubah isi proposal** (file di `docs/`). Itu dokumen akademik yang
  sudah ditandatangani dan dikumpulkan.
- Jika sebuah task butuh sesuatu yang tidak ada di proposal (misalnya nama
  kolom baru, endpoint baru), **tanya dulu**, jangan mengarang lalu
  mengimplementasikannya diam-diam.

## 3. Urutan baca sebelum mulai task apa pun

1. `PLAN.md` — tahap mana yang sedang dikerjakan, task apa saja di tahap itu.
2. `PROGRESS.md` — apa yang sudah selesai, apa yang masih jalan.
3. `.agents/rules/*.md` — aturan teknis per topik (arsitektur, data, keamanan,
   konvensi API, testing, git). Semua file di situ wajib diikuti; dibaca
   otomatis oleh Antigravity sebagai rules tambahan.
4. Skill terkait di `.agents/skills/` akan otomatis aktif kalau task-nya cocok
   (lihat deskripsi tiap skill).

Mulai setiap sesi kerja dengan **Planning mode**: minta Antigravity membaca
`PLAN.md` + rules terkait, susun rencana task, dan tunjukkan ke kamu dulu
sebelum menulis kode (jangan langsung Fast mode untuk task besar).

## 4. Aturan non-negotiable (ringkasan — detail di `.agents/rules/`)

1. **REST API adalah satu-satunya jalur baca/tulis** untuk data produk,
   pesanan, item_pesanan, dan donasi. Frontend tidak pernah query tabel-tabel
   itu langsung ke Supabase. (Pengecualian: frontend login via Supabase Auth
   dan upload foto langsung ke Supabase Storage — itu BUKAN tugas backend.)
2. **Total pesanan selalu dihitung ulang di server** dari harga produk di
   database, tidak pernah dipercaya dari body request.
3. **Endpoint `/api/admin/*` wajib divalidasi JWT Supabase + role** lewat
   middleware, sebelum masuk ke handler. Endpoint publik (`/api/produk`,
   `/api/pesanan` POST, `/api/donasi/ringkasan`) tidak butuh token.
4. **Row Level Security di Supabase tetap aktif** sebagai lapisan pertahanan
   tambahan di level database — backend tidak boleh jadi alasan untuk
   mematikannya.
5. **Pembuatan pesanan (`POST /api/pesanan`) harus atomik** (satu transaksi
   GORM): validasi produk tersedia, hitung total, simpan pesanan + item,
   ubah status produk jadi `dipesan` — semua atau tidak sama sekali. Tidak
   boleh ada race condition yang membuat satu produk terjual ke dua pembeli.
6. Tidak ada payment gateway di lingkup inti. Tidak ada pengiriman pesan
   WhatsApp otomatis dari backend — backend hanya mengembalikan data yang
   dibutuhkan frontend untuk menyusun link WhatsApp.
7. Jangan commit secret apa pun (service role key, JWT secret, connection
   string). Semua lewat environment variable — lihat `.env.example`.

## 5. Stack & keputusan teknis

Dari Tabel 3.6/3.7 proposal: Go + Fiber v2, GORM, PostgreSQL terkelola
Supabase, Supabase Auth (JWT) untuk login panel, Supabase Storage untuk foto
(diakses langsung oleh frontend), Postman untuk pengujian API manual, Git/
GitHub untuk version control.

Hal yang **tidak** disebutkan eksplisit di proposal dan jadi keputusan
implementasi standar (boleh didiskusikan ulang dengan pembimbing, bukan
mengubah proposal): versi Go 1.22+, struktur folder standar Go
(`cmd/api`, `internal/...`), migrasi via `gorm.io/gorm` AutoMigrate atau
file SQL bernomor di `migrations/` — pilih salah satu dan konsisten.

## 6. Setiap selesai satu task

- Update `PROGRESS.md` (centang item, catat tanggal).
- Commit dengan pesan yang menyebut kode kebutuhan terkait, contoh:
  `feat(pesanan): POST /api/pesanan atomik (F-04, F-05, P-02, P-03)`.
  Lihat `.agents/rules/60-git-workflow.md`.
- Kalau task itu endpoint baru: tambahkan/perbarui request di koleksi
  Postman (`postman/comithrift.postman_collection.json`) sesuai
  `.agents/rules/50-testing-postman.md`, dan jalankan skenario P-xx terkait
  dari Tabel 4.2 proposal sebelum menandai task selesai.

## 7. Setup folder sekali di awal

```
docs/PROPOSAL_PROYEK_3_COMITHRIFT_signed.pdf   # salin manual, jangan di-generate ulang
cmd/api/main.go
internal/
  config/
  db/
  models/
  middleware/
  handlers/
  services/
migrations/
postman/
.env.example   # sudah ada di kit ini
```

Minta Antigravity men-scaffold struktur ini di awal sesi pertama (lihat
Fase 0 di `PLAN.md`) sebelum mulai menulis fitur.
