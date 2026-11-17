package models

type OpsiPertanyaan struct {
	ID           uint   `gorm:"column:id_opsi;primaryKey"`
	PertanyaanID uint   `gorm:"column:id_pertanyaan"`
	Label        string `gorm:"column:label"`
	Urutan       int    `gorm:"column:urutan"`
	Base
}

func (OpsiPertanyaan) TableName() string {
	return "opsi_pertanyaan"
}
