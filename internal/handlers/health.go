package handlers

import (
	"be-comithrift/internal/db"

	"github.com/gofiber/fiber/v2"
)

// HealthCheckHandler menangani endpoint GET /healthz
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

