package services

import (
	"context"
	"fmt"
	"strings"
	"time"

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

// CreatePenyaluranInput data masukan untuk mencatat penyaluran donasi (F-09, P-07)
type CreatePenyaluranInput struct {
	Nominal    int64     `json:"nominal"`
	Keterangan string    `json:"keterangan"`
	Tanggal    time.Time `json:"tanggal"`
	IDPencatat string    `json:"id_pencatat"`
}

type DonasiService interface {
	GetRingkasan(ctx context.Context) (*RingkasanDonasi, error)
	CatatPenyaluran(ctx context.Context, input CreatePenyaluranInput) (*models.PenyaluranDonasi, error)
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

// CatatPenyaluran mencatat penyaluran donasi oleh admin (F-09, P-07)
// Wajib memvalidasi nominal tidak melebihi saldo donasi yang belum tersalurkan (aturan #6 20-data-model.md)
func (s *donasiService) CatatPenyaluran(ctx context.Context, input CreatePenyaluranInput) (*models.PenyaluranDonasi, error) {
	if input.Nominal <= 0 {
		return nil, fmt.Errorf("%w: nominal penyaluran harus lebih dari 0", ErrValidasi)
	}
	if strings.TrimSpace(input.Keterangan) == "" {
		return nil, fmt.Errorf("%w: keterangan penyaluran tidak boleh kosong", ErrValidasi)
	}
	if strings.TrimSpace(input.IDPencatat) == "" {
		return nil, fmt.Errorf("%w: identitas pencatat wajib ada", ErrValidasi)
	}
	if input.Tanggal.IsZero() {
		input.Tanggal = time.Now()
	}

	var penyaluran models.PenyaluranDonasi

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Hitung total donasi terkumpul dari pesanan dibayar/selesai
		var totalTerkumpul int64
		if err := tx.Model(&models.Pesanan{}).
			Where("status_pesanan IN (?)", []string{models.StatusPesananDibayar, models.StatusPesananSelesai}).
			Select("COALESCE(SUM(nominal_donasi), 0)").
			Scan(&totalTerkumpul).Error; err != nil {
			return fmt.Errorf("gagal menghitung total donasi terkumpul: %w", err)
		}

		// 2. Hitung total donasi yang sudah tersalurkan
		var totalTersalurkan int64
		if err := tx.Model(&models.PenyaluranDonasi{}).
			Select("COALESCE(SUM(nominal), 0)").
			Scan(&totalTersalurkan).Error; err != nil {
			return fmt.Errorf("gagal menghitung total donasi tersalurkan: %w", err)
		}

		// 3. Validasi aturan #6: Nominal tidak boleh melebihi saldo belum tersalurkan
		saldoTersedia := totalTerkumpul - totalTersalurkan
		if input.Nominal > saldoTersedia {
			return fmt.Errorf("%w: nominal penyaluran (%d) melebihi saldo donasi tersedia (%d)",
				ErrNominalMelebihiSaldo, input.Nominal, saldoTersedia)
		}

		// 4. Simpan catatan penyaluran donasi
		penyaluran = models.PenyaluranDonasi{
			Tanggal:    input.Tanggal,
			Nominal:    input.Nominal,
			Keterangan: strings.TrimSpace(input.Keterangan),
			IDPencatat: input.IDPencatat,
		}

		if err := tx.Create(&penyaluran).Error; err != nil {
			return fmt.Errorf("gagal menyimpan data penyaluran donasi: %w", err)
		}

		// 5. Preload data pencatat
		var result models.PenyaluranDonasi
		if err := tx.Preload("Pencatat").First(&result, "id_penyaluran = ?", penyaluran.IDPenyaluran).Error; err != nil {
			return fmt.Errorf("gagal memuat data pencatat: %w", err)
		}
		penyaluran = result

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &penyaluran, nil
}

