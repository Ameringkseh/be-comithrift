package handlers

import (
	"be-comithrift/internal/models"
	"be-comithrift/internal/services"
)

// ErrorDetail rincian objek kesalahan
type ErrorDetail struct {
	Code    string `json:"code" example:"TIDAK_DITEMUKAN"`
	Message string `json:"message" example:"Data tidak ditemukan."`
}

// ErrorResponse pembungkus response error standar
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// HealthResponseData data status health check
type HealthResponseData struct {
	Status   string `json:"status" example:"ok"`
	Database string `json:"database" example:"connected"`
}

// HealthResponse pembungkus response health check
type HealthResponse struct {
	Data HealthResponseData `json:"data"`
}

// MetaTotal informasi pagination / total count
type MetaTotal struct {
	Total int `json:"total" example:"4"`
}

// ProdukSingleResponse pembungkus response produk tunggal
type ProdukSingleResponse struct {
	Data models.Produk `json:"data"`
}

// ProdukListResponse pembungkus response daftar produk
type ProdukListResponse struct {
	Data []models.Produk `json:"data"`
	Meta MetaTotal       `json:"meta"`
}

// DeleteSuccessData pesan sukses penghapusan
type DeleteSuccessData struct {
	Message string `json:"message" example:"Produk berhasil dihapus."`
}

// DeleteSuccessResponse pembungkus response sukses delete
type DeleteSuccessResponse struct {
	Data DeleteSuccessData `json:"data"`
}

// PesananSingleResponse pembungkus response pesanan tunggal
type PesananSingleResponse struct {
	Data models.Pesanan `json:"data"`
}

// PesananListResponse pembungkus response daftar pesanan
type PesananListResponse struct {
	Data []models.Pesanan `json:"data"`
	Meta MetaTotal        `json:"meta"`
}

// RingkasanDonasiResponse pembungkus response transparansi donasi
type RingkasanDonasiResponse struct {
	Data services.RingkasanDonasi `json:"data"`
}

// PenyaluranDonasiResponse pembungkus response penyaluran donasi
type PenyaluranDonasiResponse struct {
	Data models.PenyaluranDonasi `json:"data"`
}

// RekapDataResponse pembungkus response rekapitulasi data admin
type RekapDataResponse struct {
	Data services.RekapResponse `json:"data"`
}
