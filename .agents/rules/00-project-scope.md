# Rule: Ruang Lingkup Proyek

## Apa ini

Backend REST API untuk COMITHRIFT — platform jual-beli pakaian bekas dengan
donasi terintegrasi "Tebar Cinta". Dikerjakan untuk mata kuliah Proyek 3,
bukan produk komersial. Spesifikasi mengikat ada di proposal yang sudah
ditandatangani (`docs/PROPOSAL_PROYEK_3_COMITHRIFT_signed.pdf`).

## Siapa penggunanya

- **Pembeli/Donatur** — anonim, tidak login, hanya berinteraksi lewat
  endpoint publik.
- **Relawan** — login, kelola produk saja.
- **Admin** — login, kelola produk + pesanan + donasi + rekap (superset
  dari Relawan).

## Yang DI DALAM lingkup backend

- Skema database & migrasi (GORM).
- Seluruh endpoint REST API di Tabel 3.7 proposal.
- Middleware autentikasi (verifikasi JWT Supabase) dan otorisasi (cek peran).
- Logika transaksional: pembuatan pesanan atomik, perhitungan total server-side,
  perubahan status pesanan/produk.
- Dokumentasi & koleksi Postman untuk pengujian endpoint.

## Yang DI LUAR lingkup backend (jangan dikerjakan di sini)

- Tampilan web (HTML/CSS/JS, Tailwind) — itu repo/folder Aditya.
- Upload file foto itu sendiri — frontend upload langsung ke Supabase
  Storage; backend hanya menerima dan menyimpan **metadata/path** hasil
  upload tersebut.
- Pengiriman pesan WhatsApp — backend tidak pernah memanggil API WhatsApp
  apa pun; backend hanya mengembalikan data (kode pesanan, rincian, total)
  yang dipakai frontend untuk menyusun link `wa.me`.
- Payment gateway (Midtrans) — eksplisit opsional dan ditunda sampai semua
  skenario uji prioritas tinggi lulus **dan** disetujui pembimbing (Subbab
  1.6 proposal). Jangan mulai kerjakan tanpa instruksi eksplisit dari user.
- Pelacakan pengiriman, kalkulasi ongkos kirim, COD — eksplisit tidak
  termasuk (Subbab 1.6).
- CI/CD otomatis untuk testing — proposal eksplisit menyatakan pengujian
  API dilakukan manual via Postman, koleksi disimpan sebagai bukti, bukan
  sebagai automated testing berkelanjutan.

## Prinsip saat ragu

Proposal ini akan dipresentasikan ke dosen pembimbing dan penguji. Jangan
menambah fitur, field, atau endpoint "karena best practice umum" tanpa
menyebutkan secara eksplisit ke user bahwa itu bukan dari proposal. Lebih
baik tanya daripada berasumsi untuk apa pun yang akan terlihat di demo atau
disebut dalam laporan.
