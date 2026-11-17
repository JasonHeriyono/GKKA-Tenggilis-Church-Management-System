package models

import "time"

type Kegiatan struct {
	ID            uint       `gorm:"column:id_kegiatan;primaryKey"`
	Nama          string     `gorm:"column:nama_kegiatan"`
	Tanggal       time.Time  `gorm:"column:tanggal"`       // hanya tanggalnya yang dipakai
	WaktuMulai    string     `gorm:"column:waktu_mulai"`   // format "07:00:00" (kolom TIME)
	WaktuSelesai  *string    `gorm:"column:waktu_selesai"` // boleh kosong
	Tema          *string    `gorm:"column:tema"`
	NatsAlkitab   *string    `gorm:"column:nats_alkitab"`
	Pembicara     *string    `gorm:"column:pembicara"`
	Lokasi        *string    `gorm:"column:lokasi"`
	Deskripsi     *string    `gorm:"column:deskripsi"`
	KomunitasID   *uint      `gorm:"column:id_komunitas"`
	Komunitas     *Komunitas `gorm:"foreignKey:KomunitasID;references:ID"`
	PerluEvaluasi bool       `gorm:"column:perlu_evaluasi"`
	Base
}

func (Kegiatan) TableName() string { return "kegiatan" }
