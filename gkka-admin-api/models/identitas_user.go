package models

import "time"

// IdentitasUser = tabel identitas_user (1:1 dengan user). Semua kolom boleh kosong.
type IdentitasUser struct {
	UserID             uint       `gorm:"column:id_user;primaryKey;autoIncrement:false"`
	NIK                *string    `gorm:"column:nik"` // hanya Super Admin
	NamaPanggilan      *string    `gorm:"column:nama_panggilan"`
	JenisKelamin       *string    `gorm:"column:jenis_kelamin"`
	TempatLahir        *string    `gorm:"column:tempat_lahir"`
	TanggalLahir       *time.Time `gorm:"column:tanggal_lahir"`
	AlamatKTP          *string    `gorm:"column:alamat_ktp"`
	RtRw               *string    `gorm:"column:rt_rw"`
	Kelurahan          *string    `gorm:"column:kelurahan"`
	Kecamatan          *string    `gorm:"column:kecamatan"`
	Kota               *string    `gorm:"column:kota"`
	Provinsi           *string    `gorm:"column:provinsi"`
	AlamatDomisili     *string    `gorm:"column:alamat_domisili"`
	Kewarganegaraan    *string    `gorm:"column:kewarganegaraan"`
	StatusPernikahan   *string    `gorm:"column:status_pernikahan"`
	TanggalPernikahan  *time.Time `gorm:"column:tanggal_pernikahan"`
	PendidikanTerakhir *string    `gorm:"column:pendidikan_terakhir"`
	Pekerjaan          *string    `gorm:"column:pekerjaan"`
	NamaAyah           *string    `gorm:"column:nama_ayah"`
	NamaIbu            *string    `gorm:"column:nama_ibu"`
	StatusGereja       *string    `gorm:"column:status_gereja"` // JEMAAT_GKKA / SIMPATISAN / TAMU
	SudahBaptis        bool       `gorm:"column:sudah_baptis"`
	TanggalBaptis      *time.Time `gorm:"column:tanggal_baptis"`
	GerejaBaptis       *string    `gorm:"column:gereja_baptis"`
	SudahSidi          bool       `gorm:"column:sudah_sidi"`
	TanggalSidi        *time.Time `gorm:"column:tanggal_sidi"`
	GerejaSidi         *string    `gorm:"column:gereja_sidi"`
	GerejaAsal         *string    `gorm:"column:gereja_asal"`
	TanggalBergabung   *time.Time `gorm:"column:tanggal_bergabung"`
	TanggalKeluar      *time.Time `gorm:"column:tanggal_keluar"`
	AlasanKeluar       *string    `gorm:"column:alasan_keluar"`
	Base
}

func (IdentitasUser) TableName() string { return "identitas_user" }
