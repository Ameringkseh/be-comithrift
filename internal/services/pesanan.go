package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"be-comithrift/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ItemPesananInput merepresentasikan item produk yang dibeli
type ItemPesananInput struct {
	IDProduk uint `json:"id_produk"`
}

// CreatePesananInput merepresentasikan payload input pembuatan pesanan
type CreatePesananInput struct {
	NamaPembeli       string             `json:"nama_pembeli"`
	KontakWhatsapp    string             `json:"kontak_whatsapp"`
	MetodeSerahTerima string             `json:"metode_serah_terima"`
	NominalDonasi     int64              `json:"nominal_donasi"`
	Catatan           string             `json:"catatan"`
	Items             []ItemPesananInput `json:"items"`
}

// PesananFilter parameter filter query untuk list pesanan admin
type PesananFilter struct {
	Status string `json:"status"`
}

type PesananService interface {
	Create(ctx context.Context, input CreatePesananInput) (*models.Pesanan, error)
	GetList(ctx context.Context, filter PesananFilter) ([]models.Pesanan, error)
}

type pesananService struct {
	db *gorm.DB
}

func NewPesananService(db *gorm.DB) PesananService {
	return &pesananService{db: db}
}

// generateKodePesanan menghasilkan kode pesanan unik (contoh: CMT-20261109-AB12)
func generateKodePesanan() string {
	dateStr := time.Now().Format("20060102")
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 4)
	for i := range b {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b[i] = chars[num.Int64()]
	}
	return fmt.Sprintf("CMT-%s-%s", dateStr, string(b))
}

// Create membuat pesanan secara atomik di dalam satu transaksi GORM
// Mengunci baris produk dengan SELECT ... FOR UPDATE untuk mencegah race condition (F-04, F-05, P-02, P-03)
func (s *pesananService) Create(ctx context.Context, req CreatePesananInput) (*models.Pesanan, error) {
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("%w: keranjang pesanan tidak boleh kosong", ErrValidasi)
	}

	// Cek duplikasi ID produk di dalam request yang sama
	seen := make(map[uint]bool)
	for _, it := range req.Items {
		if it.IDProduk == 0 {
			return nil, fmt.Errorf("%w: id_produk tidak valid", ErrValidasi)
		}
		if seen[it.IDProduk] {
			return nil, fmt.Errorf("%w: produk dengan ID %d duplikat di keranjang", ErrValidasi, it.IDProduk)
		}
		seen[it.IDProduk] = true
	}

	if strings.TrimSpace(req.NamaPembeli) == "" {
		return nil, fmt.Errorf("%w: nama_pembeli tidak boleh kosong", ErrValidasi)
	}
	if strings.TrimSpace(req.KontakWhatsapp) == "" {
		return nil, fmt.Errorf("%w: kontak_whatsapp tidak boleh kosong", ErrValidasi)
	}
	if strings.TrimSpace(req.MetodeSerahTerima) == "" {
		return nil, fmt.Errorf("%w: metode_serah_terima tidak boleh kosong", ErrValidasi)
	}
	if req.NominalDonasi < 0 {
		return nil, fmt.Errorf("%w: nominal_donasi harus >= 0", ErrValidasi)
	}

	var pesanan models.Pesanan
	now := time.Now()

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var itemPesananList []models.ItemPesanan
		var totalBelanja int64 = 0

		for _, it := range req.Items {
			var produk models.Produk

			// 1. SELECT ... FOR UPDATE: kunci baris produk ini sampai transaksi selesai
			// Request paralel untuk produk yang sama harus menunggu antrean transaksi
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&produk, "id_produk = ?", it.IDProduk).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return fmt.Errorf("%w: produk dengan ID %d tidak ditemukan", ErrNotFound, it.IDProduk)
				}
				return fmt.Errorf("gagal mengunci produk ID %d: %w", it.IDProduk, err)
			}

			// 2. Validasi ketersediaan: hanya produk berstatus 'tersedia' yang boleh dipesan
			if produk.Status != models.StatusProdukTersedia {
				return fmt.Errorf("%w: produk '%s' sedang tidak tersedia (status: %s)",
					ErrProdukTidakTersedia, produk.NamaProduk, produk.Status)
			}

			// 3. Snapshot harga saat pesan & hitung total dari harga di database (bukan input klien)
			itemPesananList = append(itemPesananList, models.ItemPesanan{
				IDProduk:       produk.IDProduk,
				HargaSaatPesan: produk.Harga,
			})
			totalBelanja += produk.Harga

			// 4. Update status produk menjadi 'dipesan' secara atomik
			if err := tx.Model(&produk).Update("status", models.StatusProdukDipesan).Error; err != nil {
				return fmt.Errorf("gagal memperbarui status produk '%s': %w", produk.NamaProduk, err)
			}
		}

		// 5. Simpan pesanan & relasi item_pesanan dalam satu transaksi
		pesanan = models.Pesanan{
			KodePesanan:       generateKodePesanan(),
			Tanggal:           now,
			NamaPembeli:       req.NamaPembeli,
			KontakWhatsapp:    req.KontakWhatsapp,
			MetodeSerahTerima: req.MetodeSerahTerima,
			TotalBelanja:      totalBelanja,
			NominalDonasi:     req.NominalDonasi,
			StatusPesanan:     models.StatusPesananMenungguPembayaran,
			Catatan:           req.Catatan,
			ItemPesanan:       itemPesananList,
		}

		if err := tx.Create(&pesanan).Error; err != nil {
			return fmt.Errorf("gagal menyimpan data pesanan: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &pesanan, nil
}

// GetList mengambil daftar pesanan untuk panel admin dengan opsi filter status (F-08)
func (s *pesananService) GetList(ctx context.Context, filter PesananFilter) ([]models.Pesanan, error) {
	if filter.Status != "" {
		validStatuses := map[string]bool{
			models.StatusPesananMenungguPembayaran: true,
			models.StatusPesananDibayar:            true,
			models.StatusPesananSelesai:            true,
			models.StatusPesananDibatalkan:         true,
		}
		if !validStatuses[filter.Status] {
			return nil, fmt.Errorf("%w: status pesanan '%s' tidak valid", ErrValidasi, filter.Status)
		}
	}

	var list []models.Pesanan
	query := s.db.WithContext(ctx).Model(&models.Pesanan{}).
		Preload("ItemPesanan.Produk").
		Preload("Pencatat").
		Order("tanggal DESC, id_pesanan DESC")

	if filter.Status != "" {
		query = query.Where("status_pesanan = ?", filter.Status)
	}

	if err := query.Find(&list).Error; err != nil {
		return nil, fmt.Errorf("gagal mengambil data pesanan: %w", err)
	}

	if list == nil {
		list = []models.Pesanan{}
	}

	return list, nil
}


