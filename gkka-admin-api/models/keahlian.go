package models

// Keahlian = tabel keahlian (skill: Pemusik, Singer, Usher, dst).
// IsSistem true = bawaan sistem (Usher), tidak boleh diubah atau dihapus lewat API.
type Keahlian struct {
	ID       uint   `gorm:"column:id_keahlian;primaryKey"`
	Nama     string `gorm:"column:nama_keahlian"`
	IsSistem bool   `gorm:"column:is_sistem"`
	Base
}

func (Keahlian) TableName() string { return "keahlian" }

// UserKeahlian = tabel user_keahlian (jemaat punya keahlian apa saja).
type UserKeahlian struct {
	ID         uint `gorm:"column:id_user_keahlian;primaryKey"`
	UserID     uint `gorm:"column:id_user"`
	User       User `gorm:"foreignKey:UserID;references:ID"`
	KeahlianID uint `gorm:"column:id_keahlian"`
	// Sumber: ADMIN (diberikan sekretariat, default) / TRAINING (lulus pelatihan). Jemaat tidak memilih sendiri.
	Sumber string `gorm:"column:sumber;default:ADMIN"`
	Base
}

func (UserKeahlian) TableName() string { return "user_keahlian" }
