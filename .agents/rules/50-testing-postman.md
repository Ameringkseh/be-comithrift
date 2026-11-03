# Rule: Pengujian via Postman

Sumber: Subbab 4.2 dan Tabel 4.2 (Rencana Pengujian REST API) di proposal.
Pengujian backend di proyek ini **manual lewat Postman**, bukan automated
test suite / CI. Koleksi Postman adalah bukti pengujian yang akan
dilampirkan ke laporan akhir (Lampiran 7).

## Struktur folder `postman/`

```
postman/
  comithrift.postman_collection.json
  comithrift.postman_environment.json   # berisi base_url, token_admin, token_relawan (kosong secara default)
```

Environment **tidak boleh** berisi token/password asli yang di-commit —
isi field token kosong atau placeholder, token asli diisi manual tiap kali
testing lokal dan tidak di-push.

## Skenario wajib ada di koleksi (Tabel 4.2)

| Kode | Skenario                                              | Endpoint                          |
| ---- | ------------------------------------------------------ | ---------------------------------- |
| P-01 | Mengambil daftar produk dengan filter                  | `GET /api/produk`                  |
| P-02 | Membuat pesanan dengan data valid                       | `POST /api/pesanan`                |
| P-03 | Membuat pesanan untuk produk yang sudah dipesan         | `POST /api/pesanan`                |
| P-04 | Mengakses endpoint admin tanpa token                    | `/api/admin/*` apa saja            |
| P-05 | Mengakses endpoint admin dengan token peran yang salah  | `/api/admin/pesanan` pakai token Relawan |
| P-06 | Admin mengubah status pesanan                           | `PATCH /api/admin/pesanan/{id}/status` |
| P-07 | Mencatat penyaluran donasi melebihi saldo               | `POST /api/admin/donasi/penyaluran` |

Setiap kali menambah endpoint baru di Fase 3/4 (`PLAN.md`), tambahkan
minimal satu request sukses dan satu request gagal (kasus yang ditolak
dengan 400/401/403/404) untuk endpoint itu ke koleksi — jangan hanya
menambahkan kode handler tanpa request Postman-nya.

## Penulisan request

- Satu folder Postman per grup endpoint (`Produk`, `Pesanan`, `Donasi`,
  `Admin - Produk`, `Admin - Pesanan`, `Admin - Donasi & Rekap`).
- Nama request memuat kode skenario kalau berkorespondensi langsung ke
  Tabel 4.2, contoh: `P-03 - Checkout produk yang sudah dipesan (expect 400)`.
- Pakai variabel environment (`{{base_url}}`, `{{token_admin}}`,
  `{{token_relawan}}`), jangan hardcode URL/token di tiap request.
- Tambahkan test script ringan di tab "Tests" Postman untuk assert status
  code yang diharapkan (`pm.test(...)`), supaya "Run collection" bisa
  dipakai sebagai cek cepat sebelum demo — ini bantu, bukan pengganti
  pencatatan manual hasil di laporan.

## Setelah testing

Catatan hasil aktual (lulus/gagal, tangkapan layar/response body) untuk
laporan **ditulis manual oleh mahasiswa**, bukan digenerate otomatis oleh
Antigravity — proposal eksplisit menyatakan "tabel ini adalah rencana
pengujian, bukan hasil pengujian" dan hasil harus diisi apa adanya setelah
implementasi.
