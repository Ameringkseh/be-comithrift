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
	case errors.Is(err, services.ErrNominalMelebihiSaldo):
		return middleware.RespondError(c, fiber.StatusBadRequest, "NOMINAL_MELEBIHI_SALDO", err.Error())
	case errors.Is(err, services.ErrValidasi):
		return middleware.RespondError(c, fiber.StatusBadRequest, "VALIDASI_GAGAL", err.Error())
	default:
		return middleware.RespondError(c, fiber.StatusInternalServerError, "KESALAHAN_SERVER", "Terjadi kesalahan internal pada server.")
	}
}

// GetList menangani GET /api/produk (Publik) — Filter kategori & ukuran
// @Summary      Daftar Katalog Produk
// @Description  Mengambil daftar produk pakaian bekas yang tersedia untuk katalog publik dengan filter opsional
// @Tags         Produk (Publik)
// @Produce      json
// @Param        kategori  query     string  false  "Filter kategori produk (misal: Jaket, Kemeja, Kaos)"
// @Param        ukuran    query     string  false  "Filter ukuran produk (misal: S, M, L, XL)"
// @Param        status    query     string  false  "Filter status produk (default: tersedia)"
// @Success      200       {object}  ProdukListResponse
// @Failure      500       {object}  ErrorResponse
// @Router       /api/produk [get]
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

// GetByID menangani GET /api/produk/:id (Publik) — Detail produk
// @Summary      Detail Produk
// @Description  Mengambil rincian informasi satu produk berdasarkan ID produk
// @Tags         Produk (Publik)
// @Produce      json
// @Param        id   path      int  true  "ID Produk"
// @Success      200  {object}  ProdukSingleResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /api/produk/{id} [get]
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

// Create menangani POST /api/admin/produk (Relawan, Admin)
// @Summary      Tambah Produk Baru
// @Description  Menambahkan produk baru ke katalog oleh Relawan atau Admin
// @Tags         Produk (Admin/Relawan)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      CreateProdukRequest  true  "Data produk baru"
// @Success      201      {object}  ProdukSingleResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Router       /api/admin/produk [post]
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

// Update menangani PATCH /api/admin/produk/:id (Relawan, Admin)
// @Summary      Perbarui Data Produk
// @Description  Memperbarui informasi atau status produk oleh Relawan atau Admin
// @Tags         Produk (Admin/Relawan)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                  true  "ID Produk"
// @Param        request  body      UpdateProdukRequest  true  "Field produk yang akan diperbarui"
// @Success      200      {object}  ProdukSingleResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Router       /api/admin/produk/{id} [patch]
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

// Delete menangani DELETE /api/admin/produk/:id (Relawan, Admin)
// @Summary      Hapus Produk
// @Description  Menghapus produk dari katalog jika tidak sedang berada dalam pesanan aktif
// @Tags         Produk (Admin/Relawan)
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "ID Produk"
// @Success      200  {object}  DeleteSuccessResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /api/admin/produk/{id} [delete]
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

