package models

import "time"

// Status pengajuan verifikasi (tabel pendaftaran).
const (
	PendaftaranMenunggu  = "MENUNGGU"
	PendaftaranDisetujui = "DISETUJUI"
	PendaftaranDitolak   = "DITOLAK"
)

// Pendaftaran = tabel pendaftaran: pengajuan verifikasi dari jemaat yang sudah punya akun.
// DataDiajukan = teks JSON salinan identitas saat dikirim (tidak ikut berubah kalau identitas diedit).
type Pendaftaran struct {
	ID            uint       `gorm:"column:id_pendaftaran;primaryKey"`
	UserID        uint       `gorm:"column:id_user"`
	User          *User      `gorm:"foreignKey:UserID;references:ID"`
	DataDiajukan  string     `gorm:"column:data_diajukan"`
	Status        string     `gorm:"column:status"`
	UserCocokID   *uint      `gorm:"column:id_user_cocok"` // fitur gabung data, belum dipakai
	DiperiksaOleh *uint      `gorm:"column:diperiksa_oleh"`
	DiperiksaPada *time.Time `gorm:"column:diperiksa_pada"`
	CatatanAdmin  *string    `gorm:"column:catatan_admin"`
	Base
}

func (Pendaftaran) TableName() string { return "pendaftaran" }
