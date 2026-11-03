# Rule: Keamanan

Sumber: Subbab 2.4 (Kebutuhan Nonfungsional – Keamanan), Subbab 3.4
(Rencana hak akses), dan Tabel 6.1 (baris "Konfigurasi middleware peran
atau Row Level Security keliru...") di proposal.

## Koneksi database

Backend terhubung ke Postgres Supabase lewat connection string **direct/
session pooler** dengan role yang bisa menulis tabel aplikasi secara penuh
(default role `postgres` dari dashboard Supabase, yang secara default
bypass RLS karena jadi pemilik tabel). **Jangan** gunakan `anon key` atau
koneksi lewat PostgREST untuk operasi backend — itu jalur yang memang
dibatasi RLS dan bukan dirancang untuk service backend.

Implikasi: karena backend bypass RLS, **backend sendiri jadi satu-satunya
tempat otorisasi benar-benar ditegakkan** untuk jalur ini. Middleware JWT +
peran (lihat di bawah) bukan lapisan opsional.

## Autentikasi & otorisasi endpoint

- Endpoint publik (tidak butuh token): `GET /api/produk`,
  `GET /api/produk/{id}`, `POST /api/pesanan`, `GET /api/donasi/ringkasan`.
- Endpoint `/api/admin/*` (termasuk yang dipakai Relawan, misalnya
  `/api/admin/produk`) **wajib** melewati dua middleware berurutan:
  1. Verifikasi JWT Supabase (skill `supabase-jwt-auth-middleware`) → kalau
     tidak ada token atau invalid → **401**.
  2. Pemeriksaan peran berdasarkan `profil_pengguna.peran` → kalau peran
     tidak berwenang untuk route itu → **403**.
- Relawan hanya boleh akses endpoint produk (`/api/admin/produk*`). Admin
  boleh akses semua endpoint `/api/admin/*` (superset Relawan). Tegakkan
  ini di definisi route, bukan di dalam logic handler (supaya mudah
  diaudit — lihat semua route dalam satu file router dan langsung terlihat
  middleware apa yang dipasang).

## Validasi & integritas data

- Setiap endpoint yang menerima body/query dari klien **memvalidasi**
  tipe, format, dan rentang nilai sebelum diproses (nama tidak kosong,
  nomor WhatsApp berformat wajar, nominal donasi numerik ≥ 0, dst — lihat
  Tabel 4.1 skenario T-03, T-05 untuk daftar kasus yang harus ditolak
  dengan pesan jelas).
- Nilai yang menentukan uang (total_belanja, harga_saat_pesan) **selalu**
  dihitung ulang di server dari data di database saat itu juga — tidak
  pernah dipercaya dari request body, bahkan kalau field-nya ada.
- Karakter khusus di input teks (nama, catatan — lihat T-06 "nama dengan
  karakter khusus") harus disimpan apa adanya lewat parameter binding
  GORM, **bukan** string concatenation ke SQL mentah.

## Query database

- Selalu pakai parameter binding GORM (`Where("kolom = ?", nilai)`,
  struct-based query) untuk nilai apa pun yang berasal dari input
  pengguna. Jangan pernah membangun klausa SQL dengan `fmt.Sprintf` lalu
  menjalankannya lewat `Raw`/`Exec` memakai nilai dari request.
- Kalau terpaksa butuh `Raw`/`Exec` untuk query kompleks (agregasi rekap,
  misalnya), nilai variabel tetap lewat placeholder `?`, tidak pernah
  disisipkan langsung ke string query.

## Upload foto

Backend tidak menangani file upload sama sekali (lihat
`00-project-scope.md`). Yang backend lakukan hanya memvalidasi bahwa field
`foto` yang dikirim berupa string path/URL yang wajar (bukan validasi isi
file, karena isi file tidak pernah melewati backend).

## Rahasia & konfigurasi

- Semua secret (connection string, JWT secret Supabase) lewat environment
  variable, dibaca sekali saat startup, tidak pernah di-log.
- `.env` wajib di `.gitignore`. `.env.example` yang boleh di-commit berisi
  nama variabel saja, bukan nilai asli.
- CORS: izinkan hanya origin frontend yang terdaftar di env var
  `CORS_ALLOWED_ORIGINS`, jangan `*` kalau endpoint `/api/admin/*` ikut
  kena kebijakan CORS yang sama.

## Hal yang eksplisit TIDAK dijanjikan proposal

Jangan mengklaim atau mengimplementasikan hal berikut seolah bagian dari
lingkup yang disetujui, kecuali user memintanya secara eksplisit:
automated security scanning (mis. alat pemindai kerentanan), rate limiting
sebagai fitur formal, atau pengujian keamanan otomatis dalam pipeline CI.
Proposal menyebut pengujian API dilakukan **manual** via Postman (Subbab
1.6 & 4.2). Kalau user memang ingin lapisan ini ditambahkan, itu perubahan
scope yang perlu didiskusikan dulu di chat, bukan ditambahkan diam-diam
oleh agent.
