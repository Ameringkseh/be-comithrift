package services

import (
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"be-comithrift/internal/models"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
)

// MidtransService mendefinisikan kontrak interaksi dengan payment gateway Midtrans
type MidtransService interface {
	CreateSnapTransaction(ctx context.Context, pesanan *models.Pesanan, paymentType ...string) (snapToken string, redirectURL string, err error)
	VerifyWebhookNotification(payload map[string]interface{}) (orderID string, transactionStatus string, fraudStatus string, err error)
}

type midtransService struct {
	serverKey string
	snap      snap.Client
}

// NewMidtransService menginisialisasi service Midtrans dengan environment Sandbox / Production
func NewMidtransService(serverKey string, isProduction bool) MidtransService {
	env := midtrans.Sandbox
	if isProduction {
		env = midtrans.Production
	}

	var s snap.Client
	s.New(serverKey, env)

	return &midtransService{
		serverKey: serverKey,
		snap:      s,
	}
}

// CreateSnapTransaction membuat transaksi Snap dan menghasilkan token pembayaran serta redirect URL
// Mendukung filter channel spesifik jika paymentType diberikan (misal: "qris", "va", "ewallet")
func (m *midtransService) CreateSnapTransaction(ctx context.Context, pesanan *models.Pesanan, paymentType ...string) (string, string, error) {
	if pesanan == nil {
		return "", "", fmt.Errorf("%w: pesanan tidak boleh nil", ErrValidasi)
	}

	grossAmount := pesanan.TotalBelanja + pesanan.NominalDonasi
	if grossAmount <= 0 {
		return "", "", fmt.Errorf("%w: total transaksi harus lebih besar dari 0", ErrValidasi)
	}

	var itemDetails []midtrans.ItemDetails
	for _, it := range pesanan.ItemPesanan {
		nama := fmt.Sprintf("Produk #%d", it.IDProduk)
		if it.Produk != nil && it.Produk.NamaProduk != "" {
			nama = it.Produk.NamaProduk
		}
		itemDetails = append(itemDetails, midtrans.ItemDetails{
			ID:    fmt.Sprintf("PRD-%d", it.IDProduk),
			Name:  nama,
			Price: it.HargaSaatPesan,
			Qty:   1,
		})
	}

	if pesanan.NominalDonasi > 0 {
		itemDetails = append(itemDetails, midtrans.ItemDetails{
			ID:    "DONASI-TEBAR-CINTA",
			Name:  "Donasi Program Tebar Cinta",
			Price: pesanan.NominalDonasi,
			Qty:   1,
		})
	}

	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  pesanan.KodePesanan,
			GrossAmt: grossAmount,
		},
		Items: &itemDetails,
		CustomerDetail: &midtrans.CustomerDetails{
			FName: pesanan.NamaPembeli,
			Phone: pesanan.KontakWhatsapp,
		},
	}

	// Filter enabled_payments jika pemohon menentukan preferensi metode bayar
	if len(paymentType) > 0 && strings.TrimSpace(paymentType[0]) != "" {
		pType := strings.ToLower(strings.TrimSpace(paymentType[0]))
		switch pType {
		case "qris":
			req.EnabledPayments = []snap.SnapPaymentType{
				snap.PaymentTypeGopay,
				snap.PaymentTypeShopeepay,
				"qris",
			}
		case "va", "transfer bank", "virtual account":
			req.EnabledPayments = []snap.SnapPaymentType{
				snap.PaymentTypeBCAVA,
				snap.PaymentTypeBNIVA,
				snap.PaymentTypeBRIVA,
				snap.PaymentTypePermataVA,
				snap.PaymentTypeEChannel,
				snap.PaymentTypeOtherVA,
			}
		case "ewallet", "e-wallet", "gopay":
			req.EnabledPayments = []snap.SnapPaymentType{
				snap.PaymentTypeGopay,
				snap.PaymentTypeShopeepay,
			}
		}
	}

	resp, mErr := m.snap.CreateTransaction(req)
	if mErr != nil {
		return "", "", fmt.Errorf("gagal membuat transaksi midtrans snap: %s", mErr.GetMessage())
	}

	return resp.Token, resp.RedirectURL, nil
}

// VerifyWebhookNotification memvalidasi signature SHA-512 dari webhook Midtrans
// Formula: SHA512(order_id + status_code + gross_amount + ServerKey)
func (m *midtransService) VerifyWebhookNotification(payload map[string]interface{}) (string, string, string, error) {
	orderID, _ := payload["order_id"].(string)
	statusCode, _ := payload["status_code"].(string)
	signatureKey, _ := payload["signature_key"].(string)
	transactionStatus, _ := payload["transaction_status"].(string)
	fraudStatus, _ := payload["fraud_status"].(string)

	var grossAmountStr string
	switch v := payload["gross_amount"].(type) {
	case string:
		grossAmountStr = v
	case float64:
		grossAmountStr = strconv.FormatFloat(v, 'f', 2, 64)
	default:
		grossAmountStr = fmt.Sprintf("%v", v)
	}

	if orderID == "" || statusCode == "" || grossAmountStr == "" || signatureKey == "" {
		return "", "", "", fmt.Errorf("%w: payload notifikasi tidak lengkap", ErrValidasi)
	}

	rawSign := orderID + statusCode + grossAmountStr + m.serverKey
	hash := sha512.Sum512([]byte(rawSign))
	calculatedSign := hex.EncodeToString(hash[:])

	if subtle.ConstantTimeCompare([]byte(strings.ToLower(signatureKey)), []byte(strings.ToLower(calculatedSign))) != 1 {
		return "", "", "", ErrInvalidSignature
	}

	return orderID, transactionStatus, fraudStatus, nil
}
