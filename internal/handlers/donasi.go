package handlers

import (
	"be-comithrift/internal/services"

	"github.com/gofiber/fiber/v2"
)

type DonasiHandler struct {
	service services.DonasiService
}

func NewDonasiHandler(service services.DonasiService) *DonasiHandler {
	return &DonasiHandler{
		service: service,
	}
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
