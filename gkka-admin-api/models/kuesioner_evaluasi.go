package models

import "time"

// KuesionerEvaluasi dibuat otomatis oleh backend untuk kebaktian yang perlu evaluasi.
type KuesionerEvaluasi struct {
	ID             uint       `gorm:"column:id_kuesioner;primaryKey"`
	JenisKuesioner string     `gorm:"column:jenis_kuesioner"` // MINGGUAN / TAHUNAN
	KegiatanID     *uint      `gorm:"column:id_kegiatan"`     // wajib untuk MINGGUAN
	Kegiatan       *Kegiatan  `gorm:"foreignKey:KegiatanID;references:ID"`
	Tahun          *int       `gorm:"column:tahun"` // wajib untuk TAHUNAN
	JudulForm      string     `gorm:"column:judul_form"`
	Tema           *string    `gorm:"column:tema"`      // salinan dari kegiatan
	Pembicara      *string    `gorm:"column:pembicara"` // salinan dari kegiatan
	DibukaPada     *time.Time `gorm:"column:dibuka_pada"`
	DitutupPada    *time.Time `gorm:"column:ditutup_pada"`
	KodeAkses      string     `gorm:"column:kode_akses"` // dibuat backend, 6 karakter, contoh "K7MP2X"
	Base
}

func (KuesionerEvaluasi) TableName() string { return "kuesioner_evaluasi" }
