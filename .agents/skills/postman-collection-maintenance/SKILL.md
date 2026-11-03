---
name: postman-collection-maintenance
description: Cara menambah atau memperbarui request di postman/comithrift.postman_collection.json agar tetap konsisten dengan Tabel 4.2 proposal dan siap dipakai sebagai bukti Lampiran 7. Gunakan skill ini setiap kali menambahkan endpoint baru atau mengubah perilaku endpoint yang sudah ada.
---

# Memelihara Koleksi Postman — COMITHRIFT

## Kapan dipakai

Setiap kali sebuah task `PLAN.md` menghasilkan endpoint baru atau mengubah
kontrak endpoint yang ada (field baru, validasi baru, status code baru).
Lihat juga `.agents/rules/50-testing-postman.md` untuk aturan lengkap.

## File yang disentuh

- `postman/comithrift.postman_collection.json` — daftar request.
- `postman/comithrift.postman_environment.json` — variabel (`base_url`,
  `token_admin`, `token_relawan`), nilai token **kosong** secara default.

Edit lewat aplikasi Postman lalu export ulang, atau edit JSON langsung kalau
perubahannya kecil (tambah satu request) — jangan tulis ulang seluruh file
dari nol kalau hanya menambah satu entri, supaya request lain yang sudah
ada (dan histori testing-nya) tidak hilang.

## Struktur minimal satu request

```json
{
  "name": "P-02 - Checkout dengan data valid",
  "request": {
    "method": "POST",
    "header": [{ "key": "Content-Type", "value": "application/json" }],
    "url": "{{base_url}}/api/pesanan",
    "body": {
      "mode": "raw",
      "raw": "{\n  \"items\": [{ \"id_produk\": 1 }],\n  \"nama_pembeli\": \"Nama Uji\",\n  \"kontak_whatsapp\": \"081234567890\",\n  \"metode_serah_terima\": \"Ambil di tempat\",\n  \"nominal_donasi\": 10000\n}"
    }
  },
  "event": [
    {
      "listen": "test",
      "script": {
        "exec": [
          "pm.test('status 201', function () { pm.response.to.have.status(201); });",
          "pm.test('ada kode_pesanan', function () { pm.expect(pm.response.json().data.kode_pesanan).to.be.a('string'); });"
        ]
      }
    }
  ]
}
```

## Checklist saat menambah request baru

1. Tempatkan di folder yang sesuai (`Produk`, `Pesanan`, `Donasi`, `Admin -
   ...`) — buat folder baru kalau grup endpointnya belum ada.
2. Beri nama yang memuat kode skenario (`P-xx`) kalau request itu memang
   mengimplementasikan salah satu baris Tabel 4.2; kalau request tambahan
   di luar Tabel 4.2 (misalnya untuk debugging), beri nama deskriptif tanpa
   kode `P-xx` supaya tidak membingungkan saat jadi lampiran.
3. Tambahkan `pm.test(...)` minimal untuk status code yang diharapkan.
4. Untuk request admin: pastikan header `Authorization` memakai
   `{{token_admin}}` atau `{{token_relawan}}`, bukan nilai literal.
5. Coba jalankan request-nya (bukan cuma ditulis) sebelum menandai task
   `PLAN.md` selesai — lihat Definition of Done di
   `.agents/rules/70-definition-of-done.md`.

## Jangan

- Jangan commit `postman/comithrift.postman_environment.json` dengan token
  asli terisi.
- Jangan hapus request lama hanya karena endpoint-nya "sudah pasti benar" —
  biarkan tetap ada sebagai regression check untuk sesi kerja berikutnya.
