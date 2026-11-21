package handlers

import (
	"be-comithrift/internal/services"

	"github.com/gofiber/fiber/v2"
)

type RekapHandler struct {
	service services.RekapService
}

func NewRekapHandler(service services.RekapService) *RekapHandler {
	return &RekapHandler{
		service: service,
	}
}

// GetRekap menangani endpoint admin GET /api/admin/rekap (F-11, T-14)
// Mengembalikan agregasi statistik jumlah pesanan per status, total penjualan, total donasi, dan inventaris produk
// @Summary      Rekapitulasi Data Penjualan & Donasi
// @Description  Mengambil statistik rekapitulasi jumlah pesanan per status, total penjualan, donasi, dan inventaris produk
// @Tags         Rekapitulasi (Admin)
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  RekapDataResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/admin/rekap [get]
func (h *RekapHandler) GetRekap(c *fiber.Ctx) error {
	rekap, err := h.service.GetRekap(c.Context())
	if err != nil {
		return MapServiceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": rekap,
	})
}
