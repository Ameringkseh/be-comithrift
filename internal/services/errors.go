package services

import "errors"

var (
	ErrNotFound            = errors.New("data tidak ditemukan")
	ErrProdukTidakTersedia = errors.New("produk tidak tersedia")
	ErrValidasi            = errors.New("validasi data gagal")
	ErrProdukSedangDipesan = errors.New("produk sedang dalam pesanan aktif dan tidak dapat dihapus")
)

