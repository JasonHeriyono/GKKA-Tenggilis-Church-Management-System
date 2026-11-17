package models

type Komunitas struct {
	ID        uint    `gorm:"column:id_komunitas;primaryKey"`
	Nama      string  `gorm:"column:nama_komunitas"`
	Deskripsi *string `gorm:"column:deskripsi"`
	Base
}

func (Komunitas) TableName() string { return "komunitas" }

type UserKomunitas struct {
	ID          uint `gorm:"column:id_user_komunitas;primaryKey"`
	UserID      uint `gorm:"column:id_user"`
	User        User `gorm:"foreignKey:UserID;references:ID"`
	KomunitasID uint `gorm:"column:id_komunitas"`
	IsKetua     bool `gorm:"column:is_ketua"`
	Base
}

func (UserKomunitas) TableName() string { return "user_komunitas" }
