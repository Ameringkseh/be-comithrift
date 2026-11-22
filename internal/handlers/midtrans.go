package handlers

import (
	"errors"
	"log"

	"be-comithrift/internal/middleware"
	"be-comithrift/internal/models"
	"be-comithrift/internal/services"

	"github.com/gofiber/fiber/v2"
)

type MidtransHandler struct {
	midtransService services.MidtransService
	pesananService  services.PesananService
}

func NewMidtransHandler(midtransService services.MidtransService, pesananService services.PesananService) *MidtransHandler {
	return &MidtransHandler{
		midtransService: midtransService,
		pesananService:  pesananService,
	}
}

// HandleNotification menangani webhook notifikasi dari Midtrans
// @Summary      Webhook Notifikasi Midtrans
// @Description  Menerima callback webhook status pembayaran dari Midtrans, memverifikasi signature SHA-512, dan memperbarui status pesanan menjadi dibayar atau dibatalkan secara otomatis
// @Tags         Midtrans
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /api/midtrans/notification [post]
func (h *MidtransHandler) HandleNotification(c *fiber.Ctx) error {
	var payload map[string]interface{}
	if err := c.BodyParser(&payload); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest, "BODY_TIDAK_VALID", "Format body webhook tidak valid.")
	}

	orderID, txStatus, fraudStatus, err := h.midtransService.VerifyWebhookNotification(payload)
	if err != nil {
		if errors.Is(err, services.ErrInvalidSignature) {
			return middleware.RespondError(c, fiber.StatusBadRequest, "SIGNATURE_TIDAK_VALID", "Signature key notifikasi Midtrans tidak valid.")
		}
		return middleware.RespondError(c, fiber.StatusBadRequest, "VALIDASI_GAGAL", err.Error())
	}

	log.Printf("[Midtrans Webhook] Menerima notifikasi untuk OrderID: %s, TransactionStatus: %s, FraudStatus: %s", orderID, txStatus, fraudStatus)

	var targetStatus string
	switch txStatus {
	case "capture":
		if fraudStatus == "challenge" {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status":  "challenge",
				"message": "Transaksi dalam status challenge FDS",
			})
		} else if fraudStatus == "accept" {
			targetStatus = models.StatusPesananDibayar
		}
	case "settlement":
		targetStatus = models.StatusPesananDibayar
	case "cancel", "deny", "expire":
		targetStatus = models.StatusPesananDibatalkan
	case "pending":
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "pending",
			"message": "Pesanan masih menunggu pembayaran",
		})
	default:
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "ignored",
			"message": "Status transaksi tidak memerlukan aksi lanjutan",
		})
	}

	if targetStatus != "" {
		_, err := h.pesananService.UpdateStatusByKode(c.Context(), orderID, targetStatus)
		if err != nil {
			if errors.Is(err, services.ErrNotFound) {
				return middleware.RespondError(c, fiber.StatusNotFound, "TIDAK_DITEMUKAN", "Data pesanan tidak ditemukan.")
			}
			return MapServiceError(c, err)
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Notifikasi pembayaran berhasil diproses",
	})
}
