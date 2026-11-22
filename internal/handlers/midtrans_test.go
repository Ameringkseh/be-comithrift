package handlers_test

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"be-comithrift/internal/handlers"
	"be-comithrift/internal/models"
	"be-comithrift/internal/services"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupMidtransTestApp(t *testing.T) (*fiber.App, sqlmock.Sqlmock, string) {
	serverKey := "SB-Mid-server-handler-test"
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	require.NoError(t, err)

	midtransSvc := services.NewMidtransService(serverKey, false)
	pesananSvc := services.NewPesananService(gormDB, midtransSvc)
	midtransHandler := handlers.NewMidtransHandler(midtransSvc, pesananSvc)

	app := fiber.New()
	app.Post("/api/midtrans/notification", midtransHandler.HandleNotification)

	return app, mock, serverKey
}

func calculateTestSignature(orderID, statusCode, grossAmount, serverKey string) string {
	raw := orderID + statusCode + grossAmount + serverKey
	h := sha512.Sum512([]byte(raw))
	return hex.EncodeToString(h[:])
}

func TestMidtransHandler_HandleNotification(t *testing.T) {
	t.Run("Invalid signature rejected with 400", func(t *testing.T) {
		app, _, _ := setupMidtransTestApp(t)

		payload := map[string]interface{}{
			"order_id":           "CMT-20261122-1234",
			"status_code":        "200",
			"gross_amount":       "50000.00",
			"signature_key":      "invalid-signature",
			"transaction_status": "settlement",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/midtrans/notification", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var resBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&resBody)
		errDetail := resBody["error"].(map[string]interface{})
		assert.Equal(t, "SIGNATURE_TIDAK_VALID", errDetail["code"])
	})

	t.Run("Pending status returns 200 without DB update", func(t *testing.T) {
		app, _, serverKey := setupMidtransTestApp(t)

		orderID := "CMT-20261122-PEND"
		statusCode := "201"
		grossAmount := "75000.00"
		sig := calculateTestSignature(orderID, statusCode, grossAmount, serverKey)

		payload := map[string]interface{}{
			"order_id":           orderID,
			"status_code":        statusCode,
			"gross_amount":       grossAmount,
			"signature_key":      sig,
			"transaction_status": "pending",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/midtrans/notification", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var resBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&resBody)
		assert.Equal(t, "pending", resBody["status"])
	})

	t.Run("Settlement status updates order to dibayar and products to terjual", func(t *testing.T) {
		app, mock, serverKey := setupMidtransTestApp(t)

		orderID := "CMT-20261122-OK"
		statusCode := "200"
		grossAmount := "100000.00"
		sig := calculateTestSignature(orderID, statusCode, grossAmount, serverKey)

		mock.ExpectBegin()
		// 1. SELECT FOR UPDATE pesanan
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "pesanan" WHERE kode_pesanan = $1`)).
			WithArgs(orderID, 1).
			WillReturnRows(sqlmock.NewRows([]string{
				"id_pesanan", "kode_pesanan", "tanggal", "nama_pembeli", "kontak_whatsapp",
				"metode_serah_terima", "total_belanja", "nominal_donasi", "status_pesanan",
			}).AddRow(1, orderID, time.Now(), "Budi", "08123456789", "Kirim kurir", 100000, 0, models.StatusPesananMenungguPembayaran))

		// 2. UPDATE pesanan status -> dibayar
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "pesanan" SET "status_pesanan"=$1,"updated_at"=$2 WHERE "id_pesanan" = $3`)).
			WithArgs(models.StatusPesananDibayar, sqlmock.AnyArg(), 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 3. SELECT product IDs from item_pesanan
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "id_produk" FROM "item_pesanan" WHERE id_pesanan = $1`)).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id_produk"}).AddRow(10))

		// 4. UPDATE produk status -> terjual
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "produk" SET "status"=$1,"updated_at"=$2 WHERE id_produk IN ($3)`)).
			WithArgs(models.StatusProdukTerjual, sqlmock.AnyArg(), 10).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 5. Preload reload pesanan
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "pesanan" WHERE id_pesanan = $1`)).
			WithArgs(1, 1).
			WillReturnRows(sqlmock.NewRows([]string{
				"id_pesanan", "kode_pesanan", "tanggal", "nama_pembeli", "kontak_whatsapp",
				"metode_serah_terima", "total_belanja", "nominal_donasi", "status_pesanan",
			}).AddRow(1, orderID, time.Now(), "Budi", "08123456789", "Kirim kurir", 100000, 0, models.StatusPesananDibayar))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "item_pesanan" WHERE "item_pesanan"."id_pesanan" = $1`)).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id_item", "id_pesanan", "id_produk", "harga_saat_pesan"}).AddRow(1, 1, 10, 100000))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "produk" WHERE "produk"."id_produk" = $1`)).
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).AddRow(10, "Jaket Denim", 100000, models.StatusProdukTerjual))

		mock.ExpectCommit()

		payload := map[string]interface{}{
			"order_id":           orderID,
			"status_code":        statusCode,
			"gross_amount":       grossAmount,
			"signature_key":      sig,
			"transaction_status": "settlement",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/midtrans/notification", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var resBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&resBody)
		assert.Equal(t, "success", resBody["status"])

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Cancel status updates order to dibatalkan and products back to tersedia", func(t *testing.T) {
		app, mock, serverKey := setupMidtransTestApp(t)

		orderID := "CMT-20261122-CANCEL"
		statusCode := "200"
		grossAmount := "50000.00"
		sig := calculateTestSignature(orderID, statusCode, grossAmount, serverKey)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "pesanan" WHERE kode_pesanan = $1`)).
			WithArgs(orderID, 1).
			WillReturnRows(sqlmock.NewRows([]string{
				"id_pesanan", "kode_pesanan", "tanggal", "nama_pembeli", "kontak_whatsapp",
				"metode_serah_terima", "total_belanja", "nominal_donasi", "status_pesanan",
			}).AddRow(2, orderID, time.Now(), "Siti", "08123456788", "Ambil di toko", 50000, 0, models.StatusPesananMenungguPembayaran))

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "pesanan" SET "status_pesanan"=$1,"updated_at"=$2 WHERE "id_pesanan" = $3`)).
			WithArgs(models.StatusPesananDibatalkan, sqlmock.AnyArg(), 2).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "id_produk" FROM "item_pesanan" WHERE id_pesanan = $1`)).
			WithArgs(2).
			WillReturnRows(sqlmock.NewRows([]string{"id_produk"}).AddRow(5))

		// Produk dikembalikan ke status 'tersedia'
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "produk" SET "status"=$1,"updated_at"=$2 WHERE id_produk IN ($3)`)).
			WithArgs(models.StatusProdukTersedia, sqlmock.AnyArg(), 5).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "pesanan" WHERE id_pesanan = $1`)).
			WithArgs(2, 1).
			WillReturnRows(sqlmock.NewRows([]string{
				"id_pesanan", "kode_pesanan", "tanggal", "nama_pembeli", "kontak_whatsapp",
				"metode_serah_terima", "total_belanja", "nominal_donasi", "status_pesanan",
			}).AddRow(2, orderID, time.Now(), "Siti", "08123456788", "Ambil di toko", 50000, 0, models.StatusPesananDibatalkan))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "item_pesanan" WHERE "item_pesanan"."id_pesanan" = $1`)).
			WithArgs(2).
			WillReturnRows(sqlmock.NewRows([]string{"id_item", "id_pesanan", "id_produk", "harga_saat_pesan"}).AddRow(2, 2, 5, 50000))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "produk" WHERE "produk"."id_produk" = $1`)).
			WithArgs(5).
			WillReturnRows(sqlmock.NewRows([]string{"id_produk", "nama_produk", "harga", "status"}).AddRow(5, "Kaos Polos", 50000, models.StatusProdukTersedia))

		mock.ExpectCommit()

		payload := map[string]interface{}{
			"order_id":           orderID,
			"status_code":        statusCode,
			"gross_amount":       grossAmount,
			"signature_key":      sig,
			"transaction_status": "cancel",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/midtrans/notification", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var resBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&resBody)
		assert.Equal(t, "success", resBody["status"])

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Nonexistent order returns 404", func(t *testing.T) {
		app, mock, serverKey := setupMidtransTestApp(t)

		orderID := "CMT-NOTFOUND"
		statusCode := "200"
		grossAmount := "50000.00"
		sig := calculateTestSignature(orderID, statusCode, grossAmount, serverKey)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "pesanan" WHERE kode_pesanan = $1`)).
			WithArgs(orderID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id_pesanan"})) // No rows
		mock.ExpectRollback()

		payload := map[string]interface{}{
			"order_id":           orderID,
			"status_code":        statusCode,
			"gross_amount":       grossAmount,
			"signature_key":      sig,
			"transaction_status": "settlement",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/midtrans/notification", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var resBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&resBody)
		errDetail := resBody["error"].(map[string]interface{})
		assert.Equal(t, "TIDAK_DITEMUKAN", errDetail["code"])
	})
}
