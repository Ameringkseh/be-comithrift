package services_test

import (
	"crypto/sha512"
	"encoding/hex"
	"testing"

	"be-comithrift/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyWebhookNotification(t *testing.T) {
	serverKey := "SB-Mid-server-test-secret"
	svc := services.NewMidtransService(serverKey, false)

	orderID := "CMT-20261122-TEST"
	statusCode := "200"
	grossAmount := "95000.00"

	// Hitung signature valid sesuai formula Midtrans
	raw := orderID + statusCode + grossAmount + serverKey
	h := sha512.Sum512([]byte(raw))
	validSignature := hex.EncodeToString(h[:])

	t.Run("Valid signature", func(t *testing.T) {
		payload := map[string]interface{}{
			"order_id":           orderID,
			"status_code":        statusCode,
			"gross_amount":       grossAmount,
			"signature_key":      validSignature,
			"transaction_status": "settlement",
			"fraud_status":       "accept",
		}

		retOrderID, txStatus, fraudStatus, err := svc.VerifyWebhookNotification(payload)
		require.NoError(t, err)
		assert.Equal(t, orderID, retOrderID)
		assert.Equal(t, "settlement", txStatus)
		assert.Equal(t, "accept", fraudStatus)
	})

	t.Run("Invalid signature (tampered amount)", func(t *testing.T) {
		payload := map[string]interface{}{
			"order_id":           orderID,
			"status_code":        statusCode,
			"gross_amount":       "10000.00", // Tampered!
			"signature_key":      validSignature,
			"transaction_status": "settlement",
		}

		_, _, _, err := svc.VerifyWebhookNotification(payload)
		require.Error(t, err)
		assert.ErrorIs(t, err, services.ErrInvalidSignature)
	})

	t.Run("Invalid signature (tampered order ID)", func(t *testing.T) {
		payload := map[string]interface{}{
			"order_id":           "CMT-HACKED",
			"status_code":        statusCode,
			"gross_amount":       grossAmount,
			"signature_key":      validSignature,
			"transaction_status": "settlement",
		}

		_, _, _, err := svc.VerifyWebhookNotification(payload)
		require.Error(t, err)
		assert.ErrorIs(t, err, services.ErrInvalidSignature)
	})

	t.Run("Missing fields", func(t *testing.T) {
		payload := map[string]interface{}{
			"order_id": orderID,
		}

		_, _, _, err := svc.VerifyWebhookNotification(payload)
		require.Error(t, err)
		assert.ErrorIs(t, err, services.ErrValidasi)
	})
}
