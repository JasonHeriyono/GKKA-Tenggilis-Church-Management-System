package models

import "time"

// Status dan alasan respon evaluasi (tabel respon_evaluasi).
const (
	ResponValid   = "VALID"
	ResponInvalid = "INVALID"

	AlasanKodeSalah   = "KODE_SALAH"   // kode tidak cocok dengan kuesioner mana pun
	AlasanLewatWaktu  = "LEWAT_WAKTU"  // dikirim setelah form ditutup
	AlasanBelumDibuka = "BELUM_DIBUKA" // dikirim sebelum form dibuka
)

// ResponEvaluasi: satu kiriman Google Form yang sudah ditarik ke database.
type ResponEvaluasi struct {
	ID               uint              `gorm:"column:id_respon;primaryKey"`
	KuesionerID      *uint             `gorm:"column:id_kuesioner"` // nil kalau kode salah
	UserID           *uint             `gorm:"column:id_user"`      // nil = anonim
	KodeDimasukkan   *string           `gorm:"column:kode_dimasukkan"`
	StatusRespon     string            `gorm:"column:status_respon"`
	AlasanInvalid    *string           `gorm:"column:alasan_invalid"`
	WaktuSubmit      time.Time         `gorm:"column:waktu_submit"`
	GoogleResponseID string            `gorm:"column:google_response_id"`
	Jawaban          []JawabanEvaluasi `gorm:"foreignKey:ResponID;references:ID"`
	Base
}

func (ResponEvaluasi) TableName() string { return "respon_evaluasi" }

// JawabanEvaluasi: jawaban satu pertanyaan. CENTANG = satu baris per opsi yang dicentang.
type JawabanEvaluasi struct {
	ID           uint    `gorm:"column:id_jawaban;primaryKey"`
	ResponID     uint    `gorm:"column:id_respon"`
	PertanyaanID uint    `gorm:"column:id_pertanyaan"`
	OpsiID       *uint   `gorm:"column:id_opsi"`      // untuk tipe pilihan
	NilaiRating  *int    `gorm:"column:nilai_rating"` // 1-5 untuk RATING
	NilaiTeks    *string `gorm:"column:nilai_teks"`   // teks, atau label pilihan yang sudah tidak ada di database
	Base
}

func (JawabanEvaluasi) TableName() string { return "jawaban_evaluasi" }
