---
name: order-status-lifecycle
description: State machine yang diusulkan untuk transisi status_pesanan (menunggu_pembayaran, dibayar, selesai, dibatalkan) dan efeknya ke status produk terkait. Gunakan skill ini saat mengimplementasikan PATCH /api/admin/pesanan/{id}/status atau logika lain yang mengubah status pesanan/produk.
---

# Siklus Status Pesanan — COMITHRIFT

> **Catatan sumber:** proposal (Subbab 3.1, Subbab 3.3 "Pembayaran dan
> verifikasi") menyatakan Admin mengubah status pesanan, dan dua aturan
> eksplisit: *"pesanan dibayar menjadikan produk terjual"* serta
> *"pesanan dibatalkan melepas produk kembali tersedia"*. Diagram transisi
> lengkap di bawah ini (termasuk transisi mana saja yang diizinkan dari
> status apa) adalah **usulan implementasi**, bukan kutipan literal
> proposal — masih ditandai "perlu dikonfirmasi" di `PROGRESS.md`. Kalau
> dosen pembimbing memberi aturan berbeda, update skill ini.

## State machine yang diusulkan

```
menunggu_pembayaran ──(Admin verifikasi bayar)──▶ dibayar ──(serah terima selesai)──▶ selesai
        │                                              │
        └──(Admin batalkan / lewat batas waktu)──▶ dibatalkan ◀──(kasus khusus, opsional)──┘
```

| Dari                   | Ke                   | Efek ke status produk            | Siapa  |
| ----------------------- | --------------------- | ---------------------------------- | ------ |
| `menunggu_pembayaran`   | `dibayar`              | produk → `terjual`                 | Admin  |
| `menunggu_pembayaran`   | `dibatalkan`           | produk → `tersedia`                | Admin  |
| `dibayar`                | `selesai`              | produk tetap `terjual`             | Admin  |
| `dibayar`                | `dibatalkan` (opsional, mis. pembatalan setelah bayar) | produk → `tersedia` | Admin |
| `selesai`, `dibatalkan` | — (status akhir, tidak bisa diubah lagi) | —                      | —      |

Transisi di luar tabel ini (misalnya `selesai` → `dibayar`) **ditolak**
dengan 400, bukan dibiarkan lewat begitu saja.

## Implementasi

```go
// internal/services/pesanan.go
var transisiDiizinkan = map[string][]string{
    models.StatusPesananMenungguPembayaran: {models.StatusPesananDibayar, models.StatusPesananDibatalkan},
    models.StatusPesananDibayar:            {models.StatusPesananSelesai, models.StatusPesananDibatalkan},
}

func (s *PesananService) UpdateStatus(ctx context.Context, idPesanan uint, statusBaru string, idAdmin string) error {
    return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        var pesanan models.Pesanan
        if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
            First(&pesanan, "id_pesanan = ?", idPesanan).Error; err != nil {
            return fmt.Errorf("%w: pesanan %v", ErrNotFound, idPesanan)
        }

        diizinkan := transisiDiizinkan[pesanan.StatusPesanan]
        if !contains(diizinkan, statusBaru) {
            return fmt.Errorf("%w: tidak bisa ubah status dari %s ke %s",
                ErrValidasi, pesanan.StatusPesanan, statusBaru)
        }

        if err := tx.Model(&pesanan).Updates(map[string]interface{}{
            "status_pesanan": statusBaru,
            "id_pencatat":    idAdmin,
        }).Error; err != nil {
            return err
        }

        // Efek ke status produk, dalam transaksi yang sama.
        switch statusBaru {
        case models.StatusPesananDibayar:
            return tx.Model(&models.Produk{}).
                Where("id_produk IN (?)", s.idProdukDariPesanan(tx, idPesanan)).
                Update("status", models.StatusProdukTerjual).Error
        case models.StatusPesananDibatalkan:
            return tx.Model(&models.Produk{}).
                Where("id_produk IN (?)", s.idProdukDariPesanan(tx, idPesanan)).
                Update("status", models.StatusProdukTersedia).Error
        }
        return nil // selesai: tidak mengubah status produk
    })
}
```

## Poin wajib

1. Perubahan status pesanan **dan** status produk terkait terjadi dalam
   **satu transaksi** yang sama — jangan update pesanan lalu update produk
   di request/transaksi terpisah (bisa membuat data tidak konsisten kalau
   salah satu gagal).
2. Tolak transisi yang tidak ada di tabel (400 `TRANSISI_TIDAK_VALID`),
   jangan diam-diam menerima status apa pun yang dikirim klien.
3. `id_pencatat` diisi dari user id Admin yang sedang login (dari
   middleware auth), bukan dari body request.
4. Uji manual (skenario P-06 + T-11): ubah ke `dibayar` → cek produk jadi
   `terjual`; ubah ke `dibatalkan` dari `menunggu_pembayaran` → cek produk
   kembali `tersedia`; coba transisi tidak valid (mis. `selesai` →
   `dibayar`) → harus ditolak 400.
