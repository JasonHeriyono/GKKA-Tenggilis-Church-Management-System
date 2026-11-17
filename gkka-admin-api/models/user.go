package models

import "time"

// Status verifikasi akun jemaat (kolom user.status_verifikasi).
const (
	BelumTerverifikasi = "BELUM_TERVERIFIKASI" // baru signup / pengajuan ditolak
	MenungguVerifikasi = "MENUNGGU_VERIFIKASI" // sudah mengajukan, menunggu sekretariat
	Terverifikasi      = "TERVERIFIKASI"       // jemaat resmi, punya kode jemaat
)

type User struct {
	ID                  uint       `gorm:"column:id_user;primaryKey"`
	KodeJemaat          *string    `gorm:"column:kode_jemaat"` // kosong sampai terverifikasi
	Nama                string     `gorm:"column:nama"`
	Email               *string    `gorm:"column:email"`
	NoTelepon           *string    `gorm:"column:no_telepon"`
	PasswordHash        *string    `gorm:"column:password_hash"`
	RoleID              uint       `gorm:"column:id_role"`
	Role                Role       `gorm:"foreignKey:RoleID;references:ID"`
	StatusVerifikasi    string     `gorm:"column:status_verifikasi;default:BELUM_TERVERIFIKASI"`
	PersetujuanDataPada *time.Time `gorm:"column:persetujuan_data_pada"` // consent UU PDP
	IsAktif             bool       `gorm:"column:is_aktif"`
	// Identitas: data diri jemaat (tabel identitas_user). Boleh nil jika belum pernah diisi.
	Identitas *IdentitasUser `gorm:"foreignKey:UserID;references:ID"`
	Base
}

func (User) TableName() string { return "user" }
