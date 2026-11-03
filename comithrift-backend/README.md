# Kit Antigravity — Backend COMITHRIFT

Ini bukan kode aplikasi — ini **file konfigurasi/instruksi** supaya
Antigravity langsung paham konteks, aturan, dan rencana kerja backend
COMITHRIFT setiap kali kamu buka workspace ini, tanpa perlu menjelaskan
ulang dari nol tiap sesi chat.

## Cara pakai

1. **Taruh folder ini sebagai root repo backend-mu** (atau salin isinya ke
   root repo yang sudah ada). Struktur folder (`cmd/`, `internal/`,
   `migrations/`, `postman/`) akan dibuat Antigravity sendiri di Fase 0
   `PLAN.md` — kamu tidak perlu membuatnya manual dulu.
2. Salin file proposal final ke `docs/PROPOSAL_PROYEK_3_COMITHRIFT_signed.pdf`
   (folder `docs/` boleh kamu buat manual, isinya hanya dokumen referensi,
   bukan dibaca otomatis oleh Antigravity kecuali kamu arahkan).
3. Isi `.env` dari `.env.example` (lihat isi file itu untuk cara dapat
   connection string & JWT secret dari dashboard Supabase).
4. Buka workspace ini di Antigravity. File `AGENTS.md` di root akan terbaca
   otomatis sebagai rules utama.
5. Mulai dengan **Planning mode**, minta sesuatu seperti:
   > "Baca AGENTS.md dan PLAN.md, lalu rencanakan Fase 0."

   Setelah itu untuk tiap sesi kerja berikutnya:
   > "Kerjakan Fase 2 task 2.1 dari PLAN.md."

   Jangan minta "kerjakan semua backend sekaligus" — pecah per task supaya
   gampang diverifikasi dan sesuai alur `PROGRESS.md`.

## Isi kit ini

| File/folder                        | Fungsi                                                             |
| ----------------------------------- | -------------------------------------------------------------------- |
| `AGENTS.md`                         | Rules utama, dibaca otomatis Antigravity di setiap task.             |
| `PLAN.md`                           | Rencana kerja backend, dipecah per fase & task, dengan kode rujukan ke proposal. |
| `PROGRESS.md`                       | Log progres — minta Antigravity update ini tiap task selesai.        |
| `.env.example`                      | Daftar environment variable yang dibutuhkan (tanpa nilai asli).      |
| `.agents/rules/*.md`                | Aturan teknis per topik (arsitektur, model data, keamanan, konvensi API, testing, git, definition of done). |
| `.agents/skills/*/SKILL.md`         | Panduan pola kode spesifik yang otomatis aktif saat relevan (handler Fiber, transaksi pesanan atomik, middleware JWT, state machine status pesanan, koleksi Postman). |

## Dasar penyusunan

Semua isi file di atas disusun berdasarkan **proposal yang sudah disetujui
dan ditandatangani** (Tabel 2.3, 3.1–3.7, 4.1–4.2, Tabel 5.1, Tabel 6.1).
Satu bagian yang ditandai eksplisit sebagai **usulan implementasi, bukan
kutipan literal proposal**: detail state machine transisi status pesanan di
skill `order-status-lifecycle` (proposal hanya menyebutkan dua aturan:
dibayar → produk terjual, dibatalkan → produk tersedia; urutan transisi
lengkapnya adalah interpretasi teknis). Item ini juga dicatat di
`PROGRESS.md` sebagai hal yang perlu dikonfirmasi ke pembimbing.

Kalau proposal direvisi lagi setelah ini (misalnya ada endpoint yang
berubah saat bimbingan), perbarui `PLAN.md` dan `.agents/rules/40-api-conventions.md`
secara manual — kit ini tidak otomatis sinkron dengan dokumen.
