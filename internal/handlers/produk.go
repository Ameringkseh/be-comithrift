package handlers

import (
	"errors"
	"fmt"
	"strconv"

	"be-comithrift/internal/middleware"
	"be-comithrift/internal/models"
	"be-comithrift/internal/services"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type ProdukHandler struct {
	service   services.ProdukService
	validator *validator.Validate
}

func NewProdukHandler(service services.ProdukService) *ProdukHandler {
	return &ProdukHandler{
		service:   service,
		validator: validator.New(),
	}
}

// Request struct untuk POST /api/admin/produk
type CreateProdukRequest struct {
	NamaProduk string `json:"nama_produk" validate:"required"`
	Harga      int64  `json:"harga" validate:"required,gt=0"`
	Deskripsi  string `json:"deskripsi"`
	Ukuran     string `json:"ukuran" validate:"required"`
	Kondisi    string `json:"kondisi" validate:"required"`
	Kategori   string `json:"kategori" validate:"required"`
	Foto       string `json:"foto"` // URL / path Supabase Storage yang sudah diunggah klien
	Status     string `json:"status" validate:"omitempty,oneof=tersedia dipesan terjual"`
}

// Request struct untuk PATCH /api/admin/produk/:id
type UpdateProdukRequest struct {
	NamaProduk *string `json:"nama_produk"`
	Harga      *int64  `json:"harga" validate:"omitempty,gt=0"`
	Deskripsi  *string `json:"deskripsi"`
	Ukuran     *string `json:"ukuran"`
	Kondisi    *string `json:"kondisi"`
	Kategori   *string `json:"kategori"`
	Foto       *string `json:"foto"`
	Status     *string `json:"status" validate:"omitempty,oneof=tersedia dipesan terjual"`
}

// MapServiceError memetakan domain service error ke response HTTP sesuai 40-api-conventions.md
func MapServiceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, services.ErrNotFound):
		return middleware.RespondError(c, fiber.StatusNotFound, "TIDAK_DITEMUKAN", "Data tidak ditemukan.")
	case errors.Is(err, services.ErrProdukTidakTersedia):
		return middleware.RespondError(c, fiber.StatusBadRequest, "PRODUK_TIDAK_TERSEDIA", "Produk sedang tidak tersedia.")
	case errors.Is(err, services.ErrProdukSedangDipesan):
		return middleware.RespondError(c, fiber.StatusBadRequest, "PRODUK_SEDANG_DIPESAN", "Produk sedang berada dalam pesanan aktif dan tidak dapat dihapus.")
	case errors.Is(err, services.ErrTransisiTidakValid):
		return middleware.RespondError(c, fiber.StatusBadRequest, "TRANSISI_TIDAK_VALID", err.Error())
	case errors.Is(err, services.ErrValidasi):
		return middleware.RespondError(c, fiber.StatusBadRequest, "VALIDASI_GAGAL", err.Error())
	default:
		return middleware.RespondError(c, fiber.StatusInternalServerError, "KESALAHAN_SERVER", "Terjadi kesalahan internal pada server.")
	}
}

// GET /api/produk (Publik) — Filter kategori & ukuran
func (h *ProdukHandler) GetList(c *fiber.Ctx) error {
	filter := services.ProdukFilter{
		Kategori: c.Query("kategori"),
		Ukuran:   c.Query("ukuran"),
		Status:   c.Query("status"), // jika kosong, service default ke 'tersedia'
	}

	list, err := h.service.GetList(c.Context(), filter)
	if err != nil {
		return MapServiceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": list,
		"meta": fiber.Map{
			"total": len(list),
		},
	})
}

// GET /api/produk/:id (Publik) — Detail produk
func (h *ProdukHandler) GetByID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest, "ID_TIDAK_VALID", "Parameter ID harus berupa angka.")
	}

	produk, err := h.service.GetByID(c.Context(), uint(id))
	if err != nil {
		return MapServiceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": produk,
	})
}

// POST /api/admin/produk (Relawan, Admin)
func (h *ProdukHandler) Create(c *fiber.Ctx) error {
	var req CreateProdukRequest
	if err := c.BodyParser(&req); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest, "BODY_TIDAK_VALID", "Format request tidak dapat dibaca.")
	}

	if err := h.validator.Struct(req); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest, "VALIDASI_GAGAL", fmt.Sprintf("Validasi input gagal: %v", err))
	}

	produk := &models.Produk{
		NamaProduk: req.NamaProduk,
		Harga:      req.Harga,
		Deskripsi:  req.Deskripsi,
		Ukuran:     req.Ukuran,
		Kondisi:    req.Kondisi,
		Kategori:   req.Kategori,
		Foto:       req.Foto,
		Status:     req.Status,
	}

	created, err := h.service.Create(c.Context(), produk)
	if err != nil {
		return MapServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": created,
	})
}

// PATCH /api/admin/produk/:id (Relawan, Admin)
func (h *ProdukHandler) Update(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest, "ID_TIDAK_VALID", "Parameter ID harus berupa angka.")
	}

	var req UpdateProdukRequest
	if err := c.BodyParser(&req); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest, "BODY_TIDAK_VALID", "Format request tidak dapat dibaca.")
	}

	if err := h.validator.Struct(req); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest, "VALIDASI_GAGAL", fmt.Sprintf("Validasi input gagal: %v", err))
	}

	input := services.UpdateProdukInput{
		NamaProduk: req.NamaProduk,
		Harga:      req.Harga,
		Deskripsi:  req.Deskripsi,
		Ukuran:     req.Ukuran,
		Kondisi:    req.Kondisi,
		Kategori:   req.Kategori,
		Foto:       req.Foto,
		Status:     req.Status,
	}

	updated, err := h.service.Update(c.Context(), uint(id), input)
	if err != nil {
		return MapServiceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": updated,
	})
}

// DELETE /api/admin/produk/:id (Relawan, Admin)
func (h *ProdukHandler) Delete(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest, "ID_TIDAK_VALID", "Parameter ID harus berupa angka.")
	}

	if err := h.service.Delete(c.Context(), uint(id)); err != nil {
		return MapServiceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": fiber.Map{
			"message": "Produk berhasil dihapus.",
		},
	})
}

