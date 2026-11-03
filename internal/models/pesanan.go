package models

import "time"

// Definisi status pesanan (enum: menunggu_pembayaran, dibayar, selesai, dibatalkan)
const (
	StatusPesananMenungguPembayaran = "menunggu_pembayaran"
	StatusPesananDibayar            = "dibayar"
	StatusPesananSelesai            = "selesai"
	StatusPesananDibatalkan         = "dibatalkan"
)

// Pesanan merepresentasikan transaksi pembelian pakaian dan donasi
// Sumber: Tabel 3.3 proposal & .agents/rules/20-data-model.md
type Pesanan struct {
	IDPesanan         uint            `gorm:"primaryKey;column:id_pesanan;autoIncrement" json:"id_pesanan"`
	KodePesanan       string          `gorm:"column:kode_pesanan;type:varchar(50);uniqueIndex;not null" json:"kode_pesanan"`
	Tanggal           time.Time       `gorm:"column:tanggal;not null" json:"tanggal"`
	NamaPembeli       string          `gorm:"column:nama_pembeli;type:varchar(255);not null" json:"nama_pembeli"`
	KontakWhatsapp    string          `gorm:"column:kontak_whatsapp;type:varchar(50);not null" json:"kontak_whatsapp"`
	MetodeSerahTerima string          `gorm:"column:metode_serah_terima;type:varchar(100);not null" json:"metode_serah_terima"`
	TotalBelanja      int64           `gorm:"column:total_belanja;not null;check:total_belanja >= 0" json:"total_belanja"`
	NominalDonasi     int64           `gorm:"column:nominal_donasi;not null;default:0;check:nominal_donasi >= 0" json:"nominal_donasi"`
	StatusPesanan     string          `gorm:"column:status_pesanan;type:varchar(30);not null;default:'menunggu_pembayaran';check:status_pesanan IN ('menunggu_pembayaran', 'dibayar', 'selesai', 'dibatalkan')" json:"status_pesanan"`
	Catatan           string          `gorm:"column:catatan;type:text" json:"catatan"`
	IDPencatat        *string         `gorm:"column:id_pencatat;type:varchar(36)" json:"id_pencatat,omitempty"`
	Pencatat          *ProfilPengguna `gorm:"foreignKey:IDPencatat;references:IDPengguna" json:"pencatat,omitempty"`
	ItemPesanan       []ItemPesanan   `gorm:"foreignKey:IDPesanan;references:IDPesanan" json:"item_pesanan,omitempty"`
	CreatedAt         time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName menentukan nama tabel di database
func (Pesanan) TableName() string {
	return "pesanan"
}

