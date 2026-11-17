package models

import "time"

// Pengaturan: satu baris = satu pengaturan aplikasi (kunci -> nilai).
type Pengaturan struct {
	Kunci      string    `gorm:"column:kunci;primaryKey"`
	Nilai      string    `gorm:"column:nilai"`
	Deskripsi  *string   `gorm:"column:deskripsi"`
	DiubahOleh *uint     `gorm:"column:diubah_oleh"` // admin terakhir yang mengubah
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (Pengaturan) TableName() string { return "pengaturan" }
