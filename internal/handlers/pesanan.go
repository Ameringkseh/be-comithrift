package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"be-comithrift/internal/middleware"
	"be-comithrift/internal/services"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type PesananHandler struct {
	service   services.PesananService
	validator *validator.Validate
}

func NewPesananHandler(service services.PesananService) *PesananHandler {
	return &PesananHandler{
		service:   service,
		validator: validator.New(),
	}
}

type ItemPesananReq struct {
	IDProduk uint `json:"id_produk" validate:"required,gt=0"`
}

// CreatePesananRequest payload request checkout publik
// Sumber: Tabel 3.7 proposal (POST /api/pesanan) & .agents/skills/gorm-atomic-order/SKILL.md
type CreatePesananRequest struct {
	NamaPembeli       string           `json:"nama_pembeli" validate:"required"`
	KontakWhatsapp    string           `json:"kontak_whatsapp" validate:"required"`
	MetodeSerahTerima string           `json:"metode_serah_terima" validate:"required"`
	NominalDonasi     int64            `json:"nominal_donasi" validate:"gte=0"`
	Catatan           string           `json:"catatan"`
	Items             []ItemPesananReq `json:"items" validate:"required,min=1,dive"`
}

// Create menangani endpoint publik POST /api/pesanan (F-02, F-03, F-04, F-05, P-02, P-03)
func (h *PesananHandler) Create(c *fiber.Ctx) error {
	var req CreatePesananRequest
	if err := c.BodyParser(&req); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest, "BODY_TIDAK_VALID", "Format request JSON tidak dapat dibaca.")
	}

	if err := h.validator.Struct(req); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest, "VALIDASI_GAGAL", fmt.Sprintf("Validasi input gagal: %v", err))
	}

	var itemsInput []services.ItemPesananInput
	for _, it := range req.Items {
		itemsInput = append(itemsInput, services.ItemPesananInput{
			IDProduk: it.IDProduk,
		})
	}

	input := services.CreatePesananInput{
		NamaPembeli:       req.NamaPembeli,
		KontakWhatsapp:    req.KontakWhatsapp,
		MetodeSerahTerima: req.MetodeSerahTerima,
		NominalDonasi:     req.NominalDonasi,
		Catatan:           req.Catatan,
		Items:             itemsInput,
	}

	pesanan, err := h.service.Create(c.Context(), input)
	if err != nil {
		return MapServiceError(c, err)
	}

	// Status 201 Created sesuai Tabel 4.2 skenario P-02
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": pesanan,
	})
}

// GetList menangani endpoint admin GET /api/admin/pesanan dengan filter status (F-08)
func (h *PesananHandler) GetList(c *fiber.Ctx) error {
	filter := services.PesananFilter{
		Status: c.Query("status"),
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

// UpdateStatusPesananRequest body request untuk PATCH /api/admin/pesanan/:id/status
type UpdateStatusPesananRequest struct {
	StatusPesanan string `json:"status_pesanan"`
	Status        string `json:"status"` // alias opsional
}

// UpdateStatus menangani endpoint admin PATCH /api/admin/pesanan/:id/status (F-08, P-06)
// Mengikuti aturan state machine & transaksi atomik dari .agents/skills/order-status-lifecycle/SKILL.md
func (h *PesananHandler) UpdateStatus(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest, "ID_TIDAK_VALID", "Parameter ID pesanan harus berupa angka.")
	}

	var req UpdateStatusPesananRequest
	if err := c.BodyParser(&req); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest, "BODY_TIDAK_VALID", "Format request JSON tidak dapat dibaca.")
	}

	statusBaru := strings.TrimSpace(req.StatusPesanan)
	if statusBaru == "" {
		statusBaru = strings.TrimSpace(req.Status)
	}
	if statusBaru == "" {
		return middleware.RespondError(c, fiber.StatusBadRequest, "VALIDASI_GAGAL", "Field status_pesanan wajib diisi.")
	}

	// id_pencatat diambil dari token Admin yang login (c.Locals("user_id")), BUKAN dari request body
	idAdmin, _ := c.Locals("user_id").(string)
	if idAdmin == "" {
		return middleware.RespondError(c, fiber.StatusUnauthorized, "TOKEN_TIDAK_VALID", "Identitas admin tidak ditemukan.")
	}

	pesanan, err := h.service.UpdateStatus(c.Context(), uint(id), statusBaru, idAdmin)
	if err != nil {
		return MapServiceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": pesanan,
	})
}



