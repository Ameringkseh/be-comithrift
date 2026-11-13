package handlers

import (
	"fmt"

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


