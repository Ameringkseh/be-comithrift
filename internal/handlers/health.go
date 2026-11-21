package handlers

import (
	"be-comithrift/internal/db"

	"github.com/gofiber/fiber/v2"
)

// HealthCheckHandler menangani endpoint GET /healthz
// @Summary      Health Check Service
// @Description  Memeriksa status keaktifan server REST API dan konektivitas basis data Supabase
// @Tags         System
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Router       /healthz [get]
func HealthCheckHandler(c *fiber.Ctx) error {
	status := "ok"
	dbStatus := "connected"

	if db.DB == nil {
		dbStatus = "not initialized"
	} else {
		sqlDB, err := db.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			dbStatus = "disconnected"
			status = "degraded"
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": fiber.Map{
			"status":   status,
			"database": dbStatus,
		},
	})
}

