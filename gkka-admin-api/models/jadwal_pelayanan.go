package models

// Status konfirmasi penugasan pelayanan (ERD R2).
const (
	PenugasanMenunggu      = "MENUNGGU"
	PenugasanTerkonfirmasi = "TERKONFIRMASI"
	PenugasanDitolak       = "DITOLAK"
)

// JadwalPelayanan = tabel jadwal_pelayanan: jemaat X bertugas dengan keahlian Y di kegiatan Z.
// Kombinasi kegiatan + user + keahlian tidak boleh dobel (UNIQUE KEY uk_jadwal_pelayanan).
type JadwalPelayanan struct {
	ID               uint     `gorm:"column:id_penjadwalan;primaryKey"`
	KegiatanID       uint     `gorm:"column:id_kegiatan"`
	Kegiatan         Kegiatan `gorm:"foreignKey:KegiatanID;references:ID"`
	UserID           uint     `gorm:"column:id_user"`
	User             User     `gorm:"foreignKey:UserID;references:ID"`
	KeahlianID       uint     `gorm:"column:id_keahlian"`
	Keahlian         Keahlian `gorm:"foreignKey:KeahlianID;references:ID"`
	StatusKonfirmasi string   `gorm:"column:status_konfirmasi"`
	Base
}

func (JadwalPelayanan) TableName() string { return "jadwal_pelayanan" }
