package group

import "sim-g-backend/helpers/pagination"

// SaveRequest dipakai untuk tambah dan ubah komunitas.
type SaveRequest struct {
	Nama      string `json:"nama" binding:"required,max=100"`
	Deskripsi string `json:"deskripsi" binding:"max=1000"`
}

// AnggotaRequest: tambah user ke komunitas.
type AnggotaRequest struct {
	UserID  uint `json:"id_user" binding:"required"`
	IsKetua bool `json:"is_ketua"`
}

// KetuaRequest: pointer supaya nilai false tetap lolos validasi required.
type KetuaRequest struct {
	IsKetua *bool `json:"is_ketua" binding:"required"`
}

// ListQuery: ?page=&limit=&sort=&q=
type ListQuery struct {
	pagination.Params
	Q string
}

type Response struct {
	ID        uint    `json:"id"`
	Nama      string  `json:"nama"`
	Deskripsi *string `json:"deskripsi"`
}

// ListResponse: daftar komunitas + info halaman (sama formatnya dengan /users).
type ListResponse struct {
	Items []Response      `json:"items"`
	Meta  pagination.Meta `json:"meta"`
}

type AnggotaResponse struct {
	UserID     uint   `json:"id_user"`
	KodeJemaat *string `json:"kode_jemaat"` // null kalau belum terverifikasi
	Nama       string `json:"nama"`
	IsKetua    bool   `json:"is_ketua"`
}

// DetailResponse = Response + daftar anggota (embedding, sama seperti Base di models).
type DetailResponse struct {
	Response
	Anggota []AnggotaResponse `json:"anggota"`
}
