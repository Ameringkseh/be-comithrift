package handlers

import (
	"fmt"
	"time"

	"be-comithrift/internal/middleware"
	"be-comithrift/internal/services"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type DonasiHandler struct {
	service   services.DonasiService
	validator *validator.Validate
}

func NewDonasiHandler(service services.DonasiService) *DonasiHandler {
	return &DonasiHandler{
		service:   service,
		validator: validator.New(),
	}
}

// CreatePenyaluranRequest body request untuk POST /api/admin/donasi/penyaluran
type CreatePenyaluranRequest struct {
	Nominal    int64     `json:"nominal" validate:"required,gt=0"`
	Keterangan string    `json:"keterangan" validate:"required"`
	Tanggal    time.Time `json:"tanggal"`
}

// GetRingkasan menangani endpoint publik GET /api/donasi/ringkasan (F-10, T-13)
// Menampilkan akumulasi donasi terkumpul, donasi tersalurkan, saldo donasi, serta riwayat penyaluran
func (h *DonasiHandler) GetRingkasan(c *fiber.Ctx) error {
	ringkasan, err := h.service.GetRingkasan(c.Context())
	if err != nil {
		return MapServiceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": ringkasan,
	})
}

// CatatPenyaluran menangani endpoint admin POST /api/admin/donasi/penyaluran (F-09, P-07)
// Wajib memvalidasi nominal tidak melebihi saldo donasi tersedia (aturan #6 20-data-model.md)
func (h *DonasiHandler) CatatPenyaluran(c *fiber.Ctx) error {
	var req CreatePenyaluranRequest
	if err := c.BodyParser(&req); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest, "BODY_TIDAK_VALID", "Format request JSON tidak dapat dibaca.")
	}

	if err := h.validator.Struct(req); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest, "VALIDASI_GAGAL", fmt.Sprintf("Validasi input gagal: %v", err))
	}

	// id_pencatat diambil dari token Admin yang login (c.Locals("user_id")), BUKAN dari request body
	idAdmin, _ := c.Locals("user_id").(string)
	if idAdmin == "" {
		return middleware.RespondError(c, fiber.StatusUnauthorized, "TOKEN_TIDAK_VALID", "Identitas admin tidak ditemukan.")
	}

	input := services.CreatePenyaluranInput{
		Nominal:    req.Nominal,
		Keterangan: req.Keterangan,
		Tanggal:    req.Tanggal,
		IDPencatat: idAdmin,
	}

	penyaluran, err := h.service.CatatPenyaluran(c.Context(), input)
	if err != nil {
		return MapServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": penyaluran,
	})
}

