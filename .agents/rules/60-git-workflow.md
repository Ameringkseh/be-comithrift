# Rule: Git & Kontribusi

Sumber: Subbab 4.1 ("Pengujian dilakukan silang... Pembagian tugas
disepakati...") dan Lembar Pernyataan Kontribusi di proposal — riwayat
commit adalah **bukti kontribusi individu** yang diperiksa pembimbing/
penguji (Lampiran 6). Perlakukan histori git sebagai dokumen akademik, bukan
cuma housekeeping.

## Commit

- Satu commit = satu perubahan logis yang bisa dijelaskan dalam satu
  kalimat. Jangan menumpuk banyak task `PLAN.md` dalam satu commit besar.
- Format pesan: `<tipe>(<lingkup>): <ringkasan singkat> (<kode kebutuhan/endpoint>)`
  - Tipe: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`.
  - Lingkup: nama entitas/fitur (`produk`, `pesanan`, `auth`, `donasi`).
  - Kode: kode F-xx/Tabel 3.7/P-xx yang relevan, supaya bisa ditelusuri
    balik ke proposal saat diperiksa.
  - Contoh: `feat(pesanan): POST /api/pesanan atomik (F-04, F-05, P-02, P-03)`
  - Contoh: `fix(auth): tolak token kedaluwarsa dengan 401, bukan 500`

## Branch

- `main` selalu dalam keadaan bisa dijalankan (`go build` sukses, server
  bisa start dengan `.env` contoh).
- Kerjakan per fase/task di branch terpisah kalau mau (`feat/fase-4-pesanan`),
  merge setelah skenario Postman terkait dicoba lulus.

## Sebelum menandai task selesai (checklist singkat)

1. `go build ./...` sukses, tidak ada warning linter yang diabaikan tanpa
   alasan.
2. Request Postman terkait (lihat `50-testing-postman.md`) sudah dicoba
   manual dan hasilnya sesuai ekspektasi di Tabel 4.1/4.2.
3. `PROGRESS.md` sudah diperbarui (centang task, catat commit hash).
4. Tidak ada secret/`.env` ter-commit — cek `git status` sebelum commit.

## Review silang

Proposal mencatat pengujian dilakukan silang antaranggota (rekan menguji
modul yang dikerjakan mahasiswa lain). Kalau user minta "siapkan untuk
rekan saya coba", pastikan: README singkat cara menjalankan server lokal +
environment Postman terisi `base_url` sudah tersedia dan up to date.
