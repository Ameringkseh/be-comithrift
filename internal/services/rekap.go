package services

import (
	"context"
	"fmt"

	"be-comithrift/internal/models"

	"gorm.io/gorm"
)

// PesananRekap ringkasan jumlah pesanan per status
type PesananRekap struct {
	MenungguPembayaran int64 `json:"menunggu_pembayaran"`
	Dibayar            int64 `json:"dibayar"`
	Selesai            int64 `json:"selesai"`
	Dibatalkan         int64 `json:"dibatalkan"`
	Total              int64 `json:"total"`
}

// DonasiRekap ringkasan keuangan donasi
type DonasiRekap struct {
	TotalTerkumpul   int64 `json:"total_terkumpul"`
	TotalTersalurkan int64 `json:"total_tersalurkan"`
	SaldoDonasi      int64 `json:"saldo_donasi"`
}

// ProdukRekap ringkasan ketersediaan inventaris produk
type ProdukRekap struct {
	Tersedia int64 `json:"tersedia"`
	Dipesan  int64 `json:"dipesan"`
	Terjual  int64 `json:"terjual"`
	Total    int64 `json:"total"`
}

// RekapResponse agregasi rekapitulasi untuk admin
// Sumber: Tabel 3.7 proposal (GET /api/admin/rekap), F-11, T-14
type RekapResponse struct {
	Pesanan        PesananRekap `json:"pesanan"`
	TotalPenjualan int64        `json:"total_penjualan"`
	Donasi         DonasiRekap  `json:"donasi"`
	Produk         ProdukRekap  `json:"produk"`
}

type RekapService interface {
	GetRekap(ctx context.Context) (*RekapResponse, error)
}

type rekapService struct {
	db *gorm.DB
}

func NewRekapService(db *gorm.DB) RekapService {
	return &rekapService{db: db}
}

// GetRekap mengumpulkan statistik agregasi pesanan, penjualan, donasi, dan produk untuk admin
func (s *rekapService) GetRekap(ctx context.Context) (*RekapResponse, error) {
	// 1. Rekap pesanan per status
	type statusCount struct {
		StatusPesanan string
		Count         int64
	}
	var pesananCounts []statusCount
	if err := s.db.WithContext(ctx).
		Model(&models.Pesanan{}).
		Select("status_pesanan, count(*) as count").
		Group("status_pesanan").
		Scan(&pesananCounts).Error; err != nil {
		return nil, fmt.Errorf("gagal menghitung rekap status pesanan: %w", err)
	}

	var pesananRekap PesananRekap
	for _, sc := range pesananCounts {
		switch sc.StatusPesanan {
		case models.StatusPesananMenungguPembayaran:
			pesananRekap.MenungguPembayaran = sc.Count
		case models.StatusPesananDibayar:
			pesananRekap.Dibayar = sc.Count
		case models.StatusPesananSelesai:
			pesananRekap.Selesai = sc.Count
		case models.StatusPesananDibatalkan:
			pesananRekap.Dibatalkan = sc.Count
		}
		pesananRekap.Total += sc.Count
	}

	// 2. Total Penjualan (jumlah total_belanja pesanan berstatus dibayar atau selesai)
	var totalPenjualan int64
	if err := s.db.WithContext(ctx).
		Model(&models.Pesanan{}).
		Where("status_pesanan IN (?)", []string{models.StatusPesananDibayar, models.StatusPesananSelesai}).
		Select("COALESCE(SUM(total_belanja), 0)").
		Scan(&totalPenjualan).Error; err != nil {
		return nil, fmt.Errorf("gagal menghitung total penjualan: %w", err)
	}

	// 3. Donasi (terkumpul dari pesanan dibayar/selesai, tersalurkan dari penyaluran_donasi)
	var totalDonasiTerkumpul int64
	if err := s.db.WithContext(ctx).
		Model(&models.Pesanan{}).
		Where("status_pesanan IN (?)", []string{models.StatusPesananDibayar, models.StatusPesananSelesai}).
		Select("COALESCE(SUM(nominal_donasi), 0)").
		Scan(&totalDonasiTerkumpul).Error; err != nil {
		return nil, fmt.Errorf("gagal menghitung total donasi terkumpul: %w", err)
	}

	var totalDonasiTersalurkan int64
	if err := s.db.WithContext(ctx).
		Model(&models.PenyaluranDonasi{}).
		Select("COALESCE(SUM(nominal), 0)").
		Scan(&totalDonasiTersalurkan).Error; err != nil {
		return nil, fmt.Errorf("gagal menghitung total donasi tersalurkan: %w", err)
	}

	saldoDonasi := totalDonasiTerkumpul - totalDonasiTersalurkan
	if saldoDonasi < 0 {
		saldoDonasi = 0
	}

	donasiRekap := DonasiRekap{
		TotalTerkumpul:   totalDonasiTerkumpul,
		TotalTersalurkan: totalDonasiTersalurkan,
		SaldoDonasi:      saldoDonasi,
	}

	// 4. Rekap status produk
	type prodStatusCount struct {
		Status string
		Count  int64
	}
	var prodCounts []prodStatusCount
	if err := s.db.WithContext(ctx).
		Model(&models.Produk{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&prodCounts).Error; err != nil {
		return nil, fmt.Errorf("gagal menghitung rekap produk: %w", err)
	}

	var produkRekap ProdukRekap
	for _, pc := range prodCounts {
		switch pc.Status {
		case models.StatusProdukTersedia:
			produkRekap.Tersedia = pc.Count
		case models.StatusProdukDipesan:
			produkRekap.Dipesan = pc.Count
		case models.StatusProdukTerjual:
			produkRekap.Terjual = pc.Count
		}
		produkRekap.Total += pc.Count
	}

	return &RekapResponse{
		Pesanan:        pesananRekap,
		TotalPenjualan: totalPenjualan,
		Donasi:         donasiRekap,
		Produk:         produkRekap,
	}, nil
}
