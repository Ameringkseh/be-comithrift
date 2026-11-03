# Rule: Model Data

Sumber: Tabel 3.3 (Rancangan Data Awal) dan "Aturan integritas yang
direncanakan" (Subbab 3.4) di proposal. Field di bawah ini adalah contoh
atribut dari proposal — nama kolom persis dan tipe data boleh disesuaikan ke
konvensi Go (camelCase di struct, snake_case di kolom via tag GORM), tapi
**jangan hilangkan atau mengganti makna field tanpa alasan**.

## Entitas

### `produk`
`id_produk (PK)`, `nama_produk`, `harga`, `deskripsi`, `ukuran`, `kondisi`,
`kategori`, `foto` (path/URL dari Storage), `status`
(`tersedia` | `dipesan` | `terjual`).

### `pesanan`
`id_pesanan (PK)`, `kode_pesanan` (unik, generate di server), `tanggal`,
`nama_pembeli`, `kontak_whatsapp`, `metode_serah_terima`, `total_belanja`,
`nominal_donasi`, `status_pesanan`
(`menunggu_pembayaran` | `dibayar` | `selesai` | `dibatalkan`), `catatan`,
`id_pencatat` (FK → profil_pengguna, nullable — diisi saat Admin mengubah
status).

### `item_pesanan`
`id_item (PK)`, `id_pesanan (FK)`, `id_produk (FK)`, `harga_saat_pesan`
(snapshot harga produk **pada saat dipesan** — jangan join ke harga produk
saat ini untuk menghitung ulang riwayat pesanan lama).

### `penyaluran_donasi`
`id_penyaluran (PK)`, `tanggal`, `nominal`, `keterangan`,
`id_pencatat (FK)`.

### `profil_pengguna`
`id_pengguna (PK, = id akun Supabase Auth)`, `nama`, `peran`
(`Admin` | `Relawan`). **Tidak ada kolom kata sandi** — autentikasi penuh
didelegasikan ke Supabase Auth.

### `info_program`
`id_info (PK)`, `nama_yayasan`, `deskripsi_program`, `info_pembayaran`,
`kontak_whatsapp_admin`. Satu baris saja (konfigurasi program donasi),
tidak perlu CRUD penuh — cukup dibaca, dan diisi manual/seed sekali.

## Aturan integritas (wajib ditegakkan di kode, bukan cuma diasumsikan)

1. `total_belanja` **= jumlah `harga_saat_pesan` seluruh item** pada
   pesanan itu, dihitung di backend dari harga produk di database saat
   pesanan dibuat — tidak pernah dari input klien.
2. `harga` > 0. `nominal_donasi` ≥ 0. `nominal` pada penyaluran_donasi ≥ 0.
3. Produk hanya boleh masuk pesanan baru kalau status-nya `tersedia`. Satu
   produk hanya boleh berada di **satu pesanan aktif** (status
   `menunggu_pembayaran`, `dibayar`, atau `selesai`) pada satu waktu — lihat
   skill `gorm-atomic-order` untuk cara menegakkan ini tanpa race condition.
4. Donasi terkumpul = jumlah `nominal_donasi` pada pesanan berstatus
   `dibayar` atau `selesai` (pesanan `menunggu_pembayaran`/`dibatalkan`
   **tidak dihitung**).
5. Donasi tersalurkan = jumlah `nominal` di `penyaluran_donasi`.
6. Nominal penyaluran donasi baru **tidak boleh melebihi** (donasi
   terkumpul − donasi tersalurkan). Tolak dengan 400 kalau melebihi
   (skenario P-07).

## Konvensi GORM

- Satu file model per entitas di `internal/models/`.
- Pakai tipe `uint`/`uuid.UUID` konsisten untuk semua primary key (pilih
  salah satu di awal, jangan campur).
- Enum (`status`, `peran`) sebagai string dengan constraint `CHECK` di
  migrasi, bukan integer magic number — supaya terbaca di query manual dan
  cocok dengan respons JSON ke frontend.
- Gunakan `gorm.Model`-style timestamp (`CreatedAt`, `UpdatedAt`) di semua
  tabel untuk kebutuhan audit/debug, meski tidak eksplisit diminta proposal.
- Uang (harga, total, nominal) disimpan sebagai integer (rupiah, tanpa
  desimal) — bukan float — untuk menghindari galat pembulatan.
