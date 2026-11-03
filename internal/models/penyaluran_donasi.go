package models

import "time"

// PenyaluranDonasi merepresentasikan catatan penyaluran donasi ke yayasan
// Sumber: Tabel 3.3 proposal & .agents/rules/20-data-model.md
type PenyaluranDonasi struct {
	IDPenyaluran uint            `gorm:"primaryKey;column:id_penyaluran;autoIncrement" json:"id_penyaluran"`
	Tanggal      time.Time       `gorm:"column:tanggal;not null" json:"tanggal"`
	Nominal      int64           `gorm:"column:nominal;not null;check:nominal >= 0" json:"nominal"`
	Keterangan   string          `gorm:"column:keterangan;type:text;not null" json:"keterangan"`
	IDPencatat   string          `gorm:"column:id_pencatat;type:varchar(36);not null" json:"id_pencatat"`
	Pencatat     *ProfilPengguna `gorm:"foreignKey:IDPencatat;references:IDPengguna" json:"pencatat,omitempty"`
	CreatedAt    time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName menentukan nama tabel di database
func (PenyaluranDonasi) TableName() string {
	return "penyaluran_donasi"
}
