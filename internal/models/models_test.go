package models_test

import (
	"testing"

	"be-comithrift/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestGORMItemPesananRelationship(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqlDB.Close()

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	// 1. Verifikasi model Produk TIDAK memiliki foreign key ke item_pesanan
	stmtProduk := &gorm.Statement{DB: db}
	if err := stmtProduk.Parse(&models.Produk{}); err != nil {
		t.Fatalf("failed to parse Produk: %v", err)
	}
	for _, rel := range stmtProduk.Schema.Relationships.Relations {
		if rel.FieldSchema.Table == "item_pesanan" {
			t.Fatalf("Produk tidak boleh memiliki relasi langsung ke item_pesanan: %v", rel.Name)
		}
	}

	// 2. Verifikasi model ItemPesanan: Produk adalah belongs_to dan FK ada di ItemPesanan (OwnPrimaryKey=false)
	stmtItem := &gorm.Statement{DB: db}
	if err := stmtItem.Parse(&models.ItemPesanan{}); err != nil {
		t.Fatalf("failed to parse ItemPesanan: %v", err)
	}

	foundProdukRel := false
	for _, rel := range stmtItem.Schema.Relationships.Relations {
		if rel.Name == "Produk" {
			foundProdukRel = true
			if rel.Type != schema.BelongsTo {
				t.Fatalf("Expected ItemPesanan.Produk to be belongs_to, got %s", rel.Type)
			}
			for _, ref := range rel.References {
				if ref.OwnPrimaryKey {
					t.Fatalf("ItemPesanan.Produk should have OwnPrimaryKey=false (FK in item_pesanan.id_produk)")
				}
				if ref.ForeignKey.DBName != "id_produk" {
					t.Fatalf("Expected foreign key column id_produk, got %s", ref.ForeignKey.DBName)
				}
			}
		}
	}

	if !foundProdukRel {
		t.Fatalf("ItemPesanan harus memiliki relasi Produk")
	}
}
