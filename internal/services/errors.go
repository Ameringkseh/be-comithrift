package services

import "errors"

var (
	ErrNotFound            = errors.New("data tidak ditemukan")
	ErrProdukTidakTersedia = errors.New("produk tidak tersedia")
	ErrValidasi            = errors.New("validasi data gagal")
	ErrProdukSedangDipesan  = errors.New("produk sedang dalam pesanan aktif dan tidak dapat dihapus")
	ErrTransisiTidakValid   = errors.New("transisi status pesanan tidak diizinkan")
	ErrNominalMelebihiSaldo = errors.New("nominal penyaluran melebihi saldo donasi yang tersedia")
	ErrInvalidSignature     = errors.New("signature notifikasi midtrans tidak valid")
)

