package models

const (
	//Tipe Form Evaluasi
	KuesionerMingguan = "MINGGUAN"
	KuesionerTahunan  = "TAHUNAN"

	//Tipe Jawaban Evaluasi
	TipeRating       = "RATING"
	TipeTeksSingkat  = "TEKS_SINGKAT"
	TipePilihanGanda = "PILIHAN_GANDA"
	TipeCentang      = "CENTANG"
	TipeDropdown     = "DROPDOWN"
	TipeParagraf     = "PARAGRAF"
)

type PertanyaanEvaluasi struct {
	ID               uint             `gorm:"column:id_pertanyaan;primaryKey"`
	JenisKuesioner   string           `gorm:"column:jenis_kuesioner"`
	Teks             string           `gorm:"column:teks_pertanyaan"`
	TipeJawaban      string           `gorm:"column:tipe_jawaban"`
	Urutan           int              `gorm:"column:urutan"`
	Wajib            bool             `gorm:"column:wajib"`
	IsAktif          bool             `gorm:"column:is_aktif"`
	GoogleItemID     *string          `gorm:"column:google_item_id"`
	GoogleQuestionID *string          `gorm:"column:google_question_id"`
	Opsi             []OpsiPertanyaan `gorm:"foreignKey:PertanyaanID;references:ID"`
	Base
}

func (PertanyaanEvaluasi) TableName() string {
	return "pertanyaan_evaluasi"
}
