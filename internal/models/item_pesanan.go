package models

import "time"

// ItemPesanan merepresentasikan detail produk yang dibeli dalam satu pesanan
// Sumber: Tabel 3.3 proposal & .agents/rules/20-data-model.md
// HargaSaatPesan adalah snapshot harga produk pada waktu transaksi dibuat.
type ItemPesanan struct {
	IDItem         uint      `gorm:"primaryKey;column:id_item;autoIncrement" json:"id_item"`
	IDPesanan      uint      `gorm:"column:id_pesanan;not null;index" json:"id_pesanan"`
	IDProduk       uint      `gorm:"column:id_produk;not null;index" json:"id_produk"`
	HargaSaatPesan int64     `gorm:"column:harga_saat_pesan;not null;check:harga_saat_pesan > 0" json:"harga_saat_pesan"`
	Produk         *Produk   `gorm:"foreignKey:IDProduk;references:IDProduk" json:"produk,omitempty"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName menentukan nama tabel di database
func (ItemPesanan) TableName() string {
	return "item_pesanan"
}

