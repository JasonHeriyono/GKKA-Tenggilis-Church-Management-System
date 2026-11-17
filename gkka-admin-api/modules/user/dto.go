package user

import (
	"sim-g-backend/helpers/identitas"
	"sim-g-backend/helpers/pagination"
)

// DataRequest: data jemaat yang diisi admin, dipakai untuk tambah dan edit.
type DataRequest struct {
	KodeJemaat string          `json:"kode_jemaat" binding:"max=20"` // kosong = dibuat otomatis (tambah) / tidak diubah (edit)
	Nama       string          `json:"nama" binding:"required,max=100"`
	Email      string          `json:"email" binding:"omitempty,email,max=100"`
	NoTelepon  string          `json:"no_telepon" binding:"omitempty,max=20"`
	Identitas  identitas.Isian `json:"identitas"`
}

// CreateRequest: Super Admin menambah jemaat secara manual. Jemaat yang ditambah admin langsung TERVERIFIKASI.
// Password boleh kosong (mis. anak); akun bisa login setelah password diisi lewat PUT /users/:id/password.
type CreateRequest struct {
	DataRequest
	Role     string `json:"role" binding:"required,oneof=Jemaat Majelis 'Super Admin'"`
	Password string `json:"password" binding:"omitempty,min=8,max=72"`
}

// UpdateRequest: edit data jemaat (role dan password punya endpoint sendiri).
type UpdateRequest = DataRequest

type RoleRequest struct {
	Role string `json:"role" binding:"required,oneof=Jemaat Majelis 'Super Admin'"`
}

type StatusRequest struct {
	IsAktif *bool `json:"is_aktif" binding:"required"`
}

type PasswordRequest struct {
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// ListQuery: ?page=&limit=&sort=&q=&role=&status_verifikasi=
type ListQuery struct {
	pagination.Params
	Q                string
	Role             string
	StatusVerifikasi string
}

type Response struct {
	ID               uint            `json:"id"`
	KodeJemaat       *string         `json:"kode_jemaat"` // null kalau belum terverifikasi
	Nama             string          `json:"nama"`
	Email            *string         `json:"email"`
	NoTelepon        *string         `json:"no_telepon"`
	Role             string          `json:"role"`
	StatusVerifikasi string          `json:"status_verifikasi"`
	IsAktif          bool            `json:"is_aktif"`
	AccountActivated bool            `json:"account_activated"` // sudah punya password
	Identitas        identitas.Isian `json:"identitas"`
}

type ListResponse struct {
	Items []Response      `json:"items"`
	Meta  pagination.Meta `json:"meta"`
}
