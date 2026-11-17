package skill

import "sim-g-backend/helpers/pagination"

// SaveRequest dipakai untuk tambah dan ubah keahlian.
type SaveRequest struct {
	Nama string `json:"nama" binding:"required,max=100"`
}

// JemaatRequest: beri keahlian ini ke satu jemaat.
type JemaatRequest struct {
	UserID uint `json:"id_user" binding:"required"`
}

// ReplaceRequest: daftar LENGKAP keahlian milik satu jemaat (menggantikan yang lama).
// Daftar kosong ([]) berarti semua keahlian jemaat itu dilepas.
type ReplaceRequest struct {
	IDs []uint `json:"id_keahlian" binding:"required,max=50,dive,gt=0"`
}

// ListQuery: ?page=&limit=&sort=&q=
type ListQuery struct {
	pagination.Params
	Q string
}

type Response struct {
	ID       uint   `json:"id"`
	Nama     string `json:"nama"`
	IsSistem bool   `json:"is_sistem"`
}

// ListResponse: daftar keahlian + info halaman (sama formatnya dengan /komunitas dan /users).
type ListResponse struct {
	Items []Response      `json:"items"`
	Meta  pagination.Meta `json:"meta"`
}

type JemaatResponse struct {
	UserID     uint   `json:"id_user"`
	KodeJemaat *string `json:"kode_jemaat"` // null kalau belum terverifikasi
	Nama       string `json:"nama"`
}

// DetailResponse = Response + daftar jemaat yang punya keahlian ini.
type DetailResponse struct {
	Response
	Jemaat []JemaatResponse `json:"jemaat"`
}

// UserKeahlianResponse: keahlian milik satu jemaat.
type UserKeahlianResponse struct {
	UserID   uint       `json:"id_user"`
	Keahlian []Response `json:"keahlian"`
}
