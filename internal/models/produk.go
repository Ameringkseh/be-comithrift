package models

import "time"

// Definisi status produk (enum: tersedia, dipesan, terjual)
const (
	StatusProdukTersedia = "tersedia"
	StatusProdukDipesan  = "dipesan"
	StatusProdukTerjual  = "terjual"
)

// Produk merepresentasikan entitas produk pakaian bekas
// Sumber: Tabel 3.3 proposal & .agents/rules/20-data-model.md
type Produk struct {
	IDProduk   uint      `gorm:"primaryKey;column:id_produk;autoIncrement" json:"id_produk"`
	NamaProduk string    `gorm:"column:nama_produk;type:varchar(255);not null" json:"nama_produk"`
	Harga      int64     `gorm:"column:harga;not null;check:harga > 0" json:"harga"`
	Deskripsi  string    `gorm:"column:deskripsi;type:text" json:"deskripsi"`
	Ukuran     string    `gorm:"column:ukuran;type:varchar(50);not null" json:"ukuran"`
	Kondisi    string    `gorm:"column:kondisi;type:varchar(100);not null" json:"kondisi"`
	Kategori   string    `gorm:"column:kategori;type:varchar(100);not null" json:"kategori"`
	Foto       string    `gorm:"column:foto;type:text" json:"foto"` // URL atau path storage Supabase
	Status     string    `gorm:"column:status;type:varchar(20);not null;default:'tersedia';check:status IN ('tersedia', 'dipesan', 'terjual')" json:"status"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName menentukan nama tabel di database
func (Produk) TableName() string {
	return "produk"
}

