---
name: gorm-atomic-order
description: Cara mengimplementasikan POST /api/pesanan (pembuatan pesanan) secara atomik dengan GORM agar satu produk tidak pernah terjual ke dua pembeli sekaligus, total dihitung ulang dari harga di database, dan semuanya tersimpan atau tidak sama sekali. Gunakan skill ini saat membuat atau mengubah logika checkout/pembuatan pesanan (service pesanan, endpoint POST /api/pesanan).
---

# Pembuatan Pesanan Atomik — COMITHRIFT

## Kenapa ini kritikal

Dari Tabel 6.1 proposal: "Konfigurasi middleware peran atau Row Level
Security keliru, atau checkout tidak atomik" → dampaknya "data pesanan
terbuka atau satu produk terjual ke lebih dari satu pembeli". Setiap produk
pakaian bekas hanya ada **satu unit** — kalau dua pembeli checkout produk
yang sama hampir bersamaan, hanya satu yang boleh berhasil (skenario P-03,
T-04).

## Langkah di dalam satu transaksi GORM

```go
// internal/services/pesanan.go
func (s *PesananService) Create(ctx context.Context, req CreatePesananInput) (*models.Pesanan, error) {
    if len(req.Items) == 0 {
        return nil, fmt.Errorf("%w: keranjang kosong", ErrValidasi)
    }

    var pesanan models.Pesanan
    err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        var items []models.ItemPesanan
        total := 0

        for _, it := range req.Items {
            var produk models.Produk
            // SELECT ... FOR UPDATE: kunci baris produk ini sampai transaksi
            // selesai, supaya request lain yang mencoba produk yang sama
            // harus menunggu, bukan baca status basi lalu sama-sama lolos.
            if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
                First(&produk, "id_produk = ?", it.IDProduk).Error; err != nil {
                return fmt.Errorf("%w: produk %v", ErrNotFound, it.IDProduk)
            }
            if produk.Status != models.StatusProdukTersedia {
                return fmt.Errorf("%w: %s", ErrProdukTidakTersedia, produk.NamaProduk)
            }

            items = append(items, models.ItemPesanan{
                IDProduk:        produk.IDProduk,
                HargaSaatPesan:  produk.Harga, // snapshot, bukan referensi
            })
            total += produk.Harga

            // Tandai dipesan SEKARANG, masih di dalam transaksi yang sama,
            // memakai baris yang sudah dikunci di atas.
            if err := tx.Model(&produk).Update("status", models.StatusProdukDipesan).Error; err != nil {
                return err
            }
        }

        pesanan = models.Pesanan{
            KodePesanan:        generateKodePesanan(), // contoh: "CMT-" + angka urut/acak
            NamaPembeli:        req.NamaPembeli,
            KontakWhatsapp:     req.KontakWhatsapp,
            MetodeSerahTerima:  req.MetodeSerahTerima,
            TotalBelanja:       total,
            NominalDonasi:      req.NominalDonasi, // sudah divalidasi ≥ 0 sebelum sampai sini
            StatusPesanan:      models.StatusPesananMenungguPembayaran,
            Items:              items,
        }
        return tx.Create(&pesanan).Error
    })
    if err != nil {
        return nil, err
    }
    return &pesanan, nil
}
```

## Poin yang wajib ada, bukan opsional

1. **`clause.Locking{Strength: "UPDATE"}`** (atau pendekatan setara, mis.
   `SELECT ... FOR UPDATE`) saat membaca status produk di dalam transaksi —
   ini yang mencegah race condition, bukan sekadar membaca lalu menulis
   belakangan.
2. **Total dihitung dari `produk.Harga` yang baru dibaca dari DB**, bukan
   dari field total apa pun yang mungkin dikirim klien.
3. Kalau *salah satu* item gagal (produk tidak ada / tidak tersedia),
   **seluruh transaksi rollback** — tidak ada pesanan "sebagian tersimpan".
   `tx.Transaction(func(tx *gorm.DB) error { ... })` sudah menangani ini
   otomatis asal setiap error benar-benar di-`return`, bukan di-ignore.
4. `kode_pesanan` harus unik — pastikan ada constraint unique di migrasi
   (lihat `.agents/rules/20-data-model.md`), jangan hanya mengandalkan
   generator acak tanpa constraint.
5. Validasi field non-produk (nama, kontak WhatsApp, metode serah terima,
   nominal donasi numerik ≥ 0) dilakukan **sebelum** masuk transaksi, lewat
   validator biasa (lihat skill `fiber-handler-pattern`) — jangan
   campur validasi bentuk dengan logika transaksi.

## Cara menguji manual bahwa ini benar-benar atomik

Kirim dua request `POST /api/pesanan` untuk `id_produk` yang sama secara
nyaris bersamaan (dua tab Postman "Send" dengan jeda minimal, atau skrip
kecil yang fire dua request paralel). Hasil yang benar: satu response 201,
satu response 400 `PRODUK_TIDAK_TERSEDIA`. Kalau dua-duanya 201, ada bug —
cek urutan lock vs read di atas.
