package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"be-comithrift/internal/models"

	"gorm.io/gorm"
)

// ProdukFilter parameter filter query untuk list produk katalog
type ProdukFilter struct {
	Kategori string
	Ukuran   string
	Status   string
}

// UpdateProdukInput field opsional untuk memperbarui data produk
type UpdateProdukInput struct {
	NamaProduk *string
	Harga      *int64
	Deskripsi  *string
	Ukuran     *string
	Kondisi    *string
	Kategori   *string
	Foto       *string
	Status     *string
}

type ProdukService interface {
	GetList(ctx context.Context, filter ProdukFilter) ([]models.Produk, error)
	GetByID(ctx context.Context, id uint) (*models.Produk, error)
	Create(ctx context.Context, produk *models.Produk) (*models.Produk, error)
	Update(ctx context.Context, id uint, input UpdateProdukInput) (*models.Produk, error)
	Delete(ctx context.Context, id uint) error
}

type produkService struct {
	db *gorm.DB
}

func NewProdukService(db *gorm.DB) ProdukService {
	return &produkService{db: db}
}

// GetList mengambil daftar produk publik (default status: tersedia jika tidak dispesifikasikan)
func (s *produkService) GetList(ctx context.Context, filter ProdukFilter) ([]models.Produk, error) {
	var list []models.Produk
	query := s.db.WithContext(ctx).Model(&models.Produk{})

	// Filter status: jika kosong, default ambil yang tersedia untuk katalog publik
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	} else {
		query = query.Where("status = ?", models.StatusProdukTersedia)
	}

	if filter.Kategori != "" {
		query = query.Where("LOWER(kategori) = ?", strings.ToLower(filter.Kategori))
	}

	if filter.Ukuran != "" {
		query = query.Where("LOWER(ukuran) = ?", strings.ToLower(filter.Ukuran))
	}

	// Urutkan produk terbaru terlebih dahulu
	if err := query.Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar produk: %w", err)
	}

	return list, nil
}

// GetByID mengambil detail satu produk
func (s *produkService) GetByID(ctx context.Context, id uint) (*models.Produk, error) {
	var produk models.Produk
	err := s.db.WithContext(ctx).First(&produk, "id_produk = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil detail produk: %w", err)
	}
	return &produk, nil
}

// Create menambahkan produk baru (Relawan / Admin)
func (s *produkService) Create(ctx context.Context, produk *models.Produk) (*models.Produk, error) {
	if produk.Harga <= 0 {
		return nil, fmt.Errorf("%w: harga harus lebih dari 0", ErrValidasi)
	}

	// Default status adalah tersedia jika kosong
	if produk.Status == "" {
		produk.Status = models.StatusProdukTersedia
	} else if produk.Status != models.StatusProdukTersedia &&
		produk.Status != models.StatusProdukDipesan &&
		produk.Status != models.StatusProdukTerjual {
		return nil, fmt.Errorf("%w: status produk tidak valid", ErrValidasi)
	}

	if err := s.db.WithContext(ctx).Create(produk).Error; err != nil {
		return nil, fmt.Errorf("gagal membuat produk: %w", err)
	}

	return produk, nil
}

// Update mengubah data produk atau status ketersediaan (Relawan / Admin)
func (s *produkService) Update(ctx context.Context, id uint, input UpdateProdukInput) (*models.Produk, error) {
	produk, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if input.NamaProduk != nil {
		if strings.TrimSpace(*input.NamaProduk) == "" {
			return nil, fmt.Errorf("%w: nama_produk tidak boleh kosong", ErrValidasi)
		}
		updates["nama_produk"] = *input.NamaProduk
	}

	if input.Harga != nil {
		if *input.Harga <= 0 {
			return nil, fmt.Errorf("%w: harga harus lebih besar dari 0", ErrValidasi)
		}
		updates["harga"] = *input.Harga
	}

	if input.Deskripsi != nil {
		updates["deskripsi"] = *input.Deskripsi
	}

	if input.Ukuran != nil {
		if strings.TrimSpace(*input.Ukuran) == "" {
			return nil, fmt.Errorf("%w: ukuran tidak boleh kosong", ErrValidasi)
		}
		updates["ukuran"] = *input.Ukuran
	}

	if input.Kondisi != nil {
		if strings.TrimSpace(*input.Kondisi) == "" {
			return nil, fmt.Errorf("%w: kondisi tidak boleh kosong", ErrValidasi)
		}
		updates["kondisi"] = *input.Kondisi
	}

	if input.Kategori != nil {
		if strings.TrimSpace(*input.Kategori) == "" {
			return nil, fmt.Errorf("%w: kategori tidak boleh kosong", ErrValidasi)
		}
		updates["kategori"] = *input.Kategori
	}

	if input.Foto != nil {
		updates["foto"] = *input.Foto
	}

	if input.Status != nil {
		st := *input.Status
		if st != models.StatusProdukTersedia && st != models.StatusProdukDipesan && st != models.StatusProdukTerjual {
			return nil, fmt.Errorf("%w: status harus 'tersedia', 'dipesan', atau 'terjual'", ErrValidasi)
		}
		updates["status"] = st
	}

	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(produk).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("gagal memperbarui produk: %w", err)
		}
	}

	return s.GetByID(ctx, id)
}

// Delete menghapus produk (Relawan / Admin)
func (s *produkService) Delete(ctx context.Context, id uint) error {
	produk, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Cek apakah produk sedang ada di pesanan aktif
	var activeItemCount int64
	err = s.db.WithContext(ctx).
		Table("item_pesanan").
		Joins("JOIN pesanan ON pesanan.id_pesanan = item_pesanan.id_pesanan").
		Where("item_pesanan.id_produk = ? AND pesanan.status_pesanan IN ('menunggu_pembayaran', 'dibayar')", id).
		Count(&activeItemCount).Error

	if err != nil {
		return fmt.Errorf("gagal memeriksa riwayat pesanan produk: %w", err)
	}

	if activeItemCount > 0 {
		return ErrProdukSedangDipesan
	}

	if err := s.db.WithContext(ctx).Delete(produk).Error; err != nil {
		return fmt.Errorf("gagal menghapus produk: %w", err)
	}

	return nil
}

