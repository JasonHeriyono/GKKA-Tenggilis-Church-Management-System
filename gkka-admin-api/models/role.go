package models

const (
	RoleGuest      = "Guest"
	RoleJemaat     = "Jemaat"
	RoleMajelis    = "Majelis"
	RoleSuperAdmin = "Super Admin"
)

type Role struct {
	ID        uint    `gorm:"column:id_role;primaryKey"`
	Nama      string  `gorm:"column:nama_role"`
	Deskripsi *string `gorm:"column:deskripsi"`
	Base
}

func AllRoles() []string {
	return []string{
		RoleGuest,
		RoleJemaat,
		RoleMajelis,
		RoleSuperAdmin,
	}
}

func (Role) TableName() string { return "role" }
