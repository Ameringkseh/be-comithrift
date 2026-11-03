# Rule: Definition of Done per Endpoint

Sebuah task endpoint di `PLAN.md` baru boleh ditandai selesai kalau **semua**
berikut terpenuhi. Antigravity: tunjukkan checklist ini ke user sebagai
bagian dari laporan penyelesaian task, jangan hanya bilang "sudah selesai".

1. **Fungsional** — endpoint mengembalikan hasil benar untuk kasus sukses
   sesuai Tabel 3.7/4.2.
2. **Validasi** — kasus input tidak valid ditolak dengan status code dan
   pesan yang sesuai `40-api-conventions.md` (bukan 500 untuk kesalahan
   input pengguna).
3. **Otorisasi** — kalau endpoint `/api/admin/*`: dicoba tanpa token (harus
   401) dan dengan token peran yang salah (harus 403).
4. **Integritas data** — kalau endpoint menulis data uang/status: nilai
   dihitung ulang di server, aturan integritas di `20-data-model.md`
   ditegakkan (bukan cuma diasumsikan benar dari input).
5. **Konkurensi** — khusus `POST /api/pesanan` dan
   `PATCH /api/admin/pesanan/{id}/status`: dicoba kondisi balapan sederhana
   (dua request nyaris bersamaan untuk produk yang sama) dan hasilnya
   benar (hanya satu yang berhasil).
6. **Postman** — request sukses + minimal satu request gagal ada di
   koleksi, sudah dicoba jalan.
7. **Tercatat** — `PROGRESS.md` diperbarui, commit mengikuti
   `60-git-workflow.md`.

Kalau salah satu poin di atas sengaja dilewati (misalnya #5 untuk endpoint
yang memang tidak rawan race condition), katakan secara eksplisit ke user
kenapa dilewati — jangan diam-diam diabaikan.
