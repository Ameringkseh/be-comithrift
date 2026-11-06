-- Migrasi Skema Basis Data COMITHRIFT
-- Sumber: Tabel 3.3 proposal & .agents/rules/20-data-model.md

-- 1. Profil Pengguna (Akun dikelola oleh Supabase Auth)
CREATE TABLE IF NOT EXISTS profil_pengguna (
    id_pengguna VARCHAR(36) PRIMARY KEY,
    nama VARCHAR(255) NOT NULL,
    peran VARCHAR(20) NOT NULL CONSTRAINT chk_profil_peran CHECK (peran IN ('Admin', 'Relawan')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. Produk
CREATE TABLE IF NOT EXISTS produk (
    id_produk BIGSERIAL PRIMARY KEY,
    nama_produk VARCHAR(255) NOT NULL,
    harga BIGINT NOT NULL CONSTRAINT chk_produk_harga CHECK (harga > 0),
    deskripsi TEXT,
    ukuran VARCHAR(50) NOT NULL,
    kondisi VARCHAR(100) NOT NULL,
    kategori VARCHAR(100) NOT NULL,
    foto TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'tersedia' CONSTRAINT chk_produk_status CHECK (status IN ('tersedia', 'dipesan', 'terjual')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index pencarian produk berdasarkan kategori, ukuran, dan status
CREATE INDEX IF NOT EXISTS idx_produk_status ON produk(status);
CREATE INDEX IF NOT EXISTS idx_produk_kategori ON produk(kategori);
CREATE INDEX IF NOT EXISTS idx_produk_ukuran ON produk(ukuran);

-- 3. Pesanan
CREATE TABLE IF NOT EXISTS pesanan (
    id_pesanan BIGSERIAL PRIMARY KEY,
    kode_pesanan VARCHAR(50) NOT NULL UNIQUE,
    tanggal TIMESTAMP WITH TIME ZONE NOT NULL,
    nama_pembeli VARCHAR(255) NOT NULL,
    kontak_whatsapp VARCHAR(50) NOT NULL,
    metode_serah_terima VARCHAR(100) NOT NULL,
    total_belanja BIGINT NOT NULL CONSTRAINT chk_pesanan_total CHECK (total_belanja >= 0),
    nominal_donasi BIGINT NOT NULL DEFAULT 0 CONSTRAINT chk_pesanan_donasi CHECK (nominal_donasi >= 0),
    status_pesanan VARCHAR(30) NOT NULL DEFAULT 'menunggu_pembayaran' CONSTRAINT chk_pesanan_status CHECK (status_pesanan IN ('menunggu_pembayaran', 'dibayar', 'selesai', 'dibatalkan')),
    catatan TEXT,
    id_pencatat VARCHAR(36) REFERENCES profil_pengguna(id_pengguna) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pesanan_status ON pesanan(status_pesanan);
CREATE INDEX IF NOT EXISTS idx_pesanan_kode ON pesanan(kode_pesanan);

-- 4. Item Pesanan
CREATE TABLE IF NOT EXISTS item_pesanan (
    id_item BIGSERIAL PRIMARY KEY,
    id_pesanan BIGINT NOT NULL REFERENCES pesanan(id_pesanan) ON DELETE CASCADE,
    id_produk BIGINT NOT NULL REFERENCES produk(id_produk) ON DELETE RESTRICT,
    harga_saat_pesan BIGINT NOT NULL CONSTRAINT chk_item_harga CHECK (harga_saat_pesan > 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_item_pesanan_id_pesanan ON item_pesanan(id_pesanan);
CREATE INDEX IF NOT EXISTS idx_item_pesanan_id_produk ON item_pesanan(id_produk);

-- 5. Penyaluran Donasi
CREATE TABLE IF NOT EXISTS penyaluran_donasi (
    id_penyaluran BIGSERIAL PRIMARY KEY,
    tanggal TIMESTAMP WITH TIME ZONE NOT NULL,
    nominal BIGINT NOT NULL CONSTRAINT chk_penyaluran_nominal CHECK (nominal >= 0),
    keterangan TEXT NOT NULL,
    id_pencatat VARCHAR(36) NOT NULL REFERENCES profil_pengguna(id_pengguna) ON DELETE RESTRICT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_penyaluran_id_pencatat ON penyaluran_donasi(id_pencatat);

-- 6. Info Program
CREATE TABLE IF NOT EXISTS info_program (
    id_info BIGSERIAL PRIMARY KEY,
    nama_yayasan VARCHAR(255) NOT NULL,
    deskripsi_program TEXT NOT NULL,
    info_pembayaran TEXT NOT NULL,
    kontak_whatsapp_admin VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

