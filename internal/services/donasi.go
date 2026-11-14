package services

import (
	"context"
	"fmt"

	"be-comithrift/internal/models"

	"gorm.io/gorm"
)

// RingkasanDonasi merepresentasikan ringkasan akumulasi donasi untuk publik & program Tebar Cinta
// Sumber: Tabel 3.7 proposal (GET /api/donasi/ringkasan), F-10, T-13, & .agents/rules/20-data-model.md
type RingkasanDonasi struct {
	TotalTerkumpul    int64                    `json:"total_terkumpul"`
	TotalTersalurkan  int64                    `json:"total_tersalurkan"`
	SaldoDonasi       int64                    `json:"saldo_donasi"`
	RiwayatPenyaluran []models.PenyaluranDonasi `json:"riwayat_penyaluran"`
	InfoProgram       *models.InfoProgram      `json:"info_program,omitempty"`
}

type DonasiService interface {
	GetRingkasan(ctx context.Context) (*RingkasanDonasi, error)
}

type donasiService struct {
	db *gorm.DB
}

func NewDonasiService(db *gorm.DB) DonasiService {
	return &donasiService{db: db}
}

// GetRingkasan menghitung total donasi terkumpul, tersalurkan, saldo, serta riwayat penyaluran
// Aturan:
// 1. Donasi terkumpul = jumlah nominal_donasi dari pesanan 'dibayar' atau 'selesai'
// 2. Donasi tersalurkan = jumlah nominal dari tabel penyaluran_donasi
// 3. Saldo donasi = donasi terkumpul - donasi tersalurkan
func (s *donasiService) GetRingkasan(ctx context.Context) (*RingkasanDonasi, error) {
	// 1. Total Terkumpul: jumlah nominal_donasi pada pesanan berstatus 'dibayar' atau 'selesai'
	var totalTerkumpul int64
	err := s.db.WithContext(ctx).
		Model(&models.Pesanan{}).
		Where("status_pesanan IN (?)", []string{models.StatusPesananDibayar, models.StatusPesananSelesai}).
		Select("COALESCE(SUM(nominal_donasi), 0)").
		Scan(&totalTerkumpul).Error
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung total donasi terkumpul: %w", err)
	}

	// 2. Total Tersalurkan: jumlah nominal pada tabel penyaluran_donasi
	var totalTersalurkan int64
	err = s.db.WithContext(ctx).
		Model(&models.PenyaluranDonasi{}).
		Select("COALESCE(SUM(nominal), 0)").
		Scan(&totalTersalurkan).Error
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung total donasi tersalurkan: %w", err)
	}

	// 3. Saldo donasi tersedia (belum tersalurkan)
	saldoDonasi := totalTerkumpul - totalTersalurkan
	if saldoDonasi < 0 {
		saldoDonasi = 0
	}

	// 4. Riwayat penyaluran donasi
	var riwayat []models.PenyaluranDonasi
	err = s.db.WithContext(ctx).
		Preload("Pencatat").
		Order("tanggal DESC, id_penyaluran DESC").
		Find(&riwayat).Error
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil riwayat penyaluran donasi: %w", err)
	}
	if riwayat == nil {
		riwayat = []models.PenyaluranDonasi{}
	}

	// 5. Info Program Tebar Cinta (jika tersedia di basis data)
	var info models.InfoProgram
	var infoPtr *models.InfoProgram
	if err := s.db.WithContext(ctx).First(&info).Error; err == nil {
		infoPtr = &info
	}

	return &RingkasanDonasi{
		TotalTerkumpul:    totalTerkumpul,
		TotalTersalurkan:  totalTersalurkan,
		SaldoDonasi:       saldoDonasi,
		RiwayatPenyaluran: riwayat,
		InfoProgram:       infoPtr,
	}, nil
}
