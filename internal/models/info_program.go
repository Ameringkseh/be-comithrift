package models

import "time"

// InfoProgram merepresentasikan konfigurasi informasi program donasi Tebar Cinta
// Sumber: Tabel 3.3 proposal & .agents/rules/20-data-model.md
// Tabel ini umumnya hanya berisi satu baris data konfigurasi.
type InfoProgram struct {
	IDInfo              uint      `gorm:"primaryKey;column:id_info;autoIncrement" json:"id_info"`
	NamaYayasan         string    `gorm:"column:nama_yayasan;type:varchar(255);not null" json:"nama_yayasan"`
	DeskripsiProgram    string    `gorm:"column:deskripsi_program;type:text;not null" json:"deskripsi_program"`
	InfoPembayaran      string    `gorm:"column:info_pembayaran;type:text;not null" json:"info_pembayaran"`
	KontakWhatsappAdmin string    `gorm:"column:kontak_whatsapp_admin;type:varchar(50);not null" json:"kontak_whatsapp_admin"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName menentukan nama tabel di database
func (InfoProgram) TableName() string {
	return "info_program"
}

