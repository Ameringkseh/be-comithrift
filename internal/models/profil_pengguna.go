package models

import "time"

// Definisi peran pengguna (enum: Admin, Relawan)
const (
	PeranAdmin   = "Admin"
	PeranRelawan = "Relawan"
)

// ProfilPengguna merepresentasikan profil admin atau relawan
// Sumber: Tabel 3.3 proposal & .agents/rules/20-data-model.md
// IDPengguna sama dengan ID akun Supabase Auth (UUID string).
// Tidak menyimpan kata sandi karena dikelola oleh Supabase Auth.
type ProfilPengguna struct {
	IDPengguna string    `gorm:"primaryKey;column:id_pengguna;type:varchar(36)" json:"id_pengguna"`
	Nama       string    `gorm:"column:nama;type:varchar(255);not null" json:"nama"`
	Peran      string    `gorm:"column:peran;type:varchar(20);not null;check:peran IN ('Admin', 'Relawan')" json:"peran"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName menentukan nama tabel di database
func (ProfilPengguna) TableName() string {
	return "profil_pengguna"
}

