package pendaftaran

import (
	"time"

	"sim-g-backend/helpers/identitas"
	"sim-g-backend/helpers/pagination"
)

// ---- sisi app jemaat ----

// SignupRequest: buat akun baru. Setelah berhasil jemaat langsung masuk (status BELUM_TERVERIFIKASI).
// Email atau no. telepon wajib salah satu (dipakai untuk login).
type SignupRequest struct {
	Nama            string `json:"nama" binding:"required,max=100"`
	Email           string `json:"email" binding:"omitempty,email,max=100"`
	NoTelepon       string `json:"no_telepon" binding:"omitempty,max=20"`
	Password        string `json:"password" binding:"required,min=8,max=72"`
	JenisKelamin    string `json:"jenis_kelamin" binding:"required,oneof=L P"`
	TanggalLahir    string `json:"tanggal_lahir" binding:"required,datetime=2006-01-02"`
	StatusGereja    string `json:"status_gereja" binding:"required,oneof=JEMAAT_GKKA SIMPATISAN TAMU"`
	PersetujuanData bool   `json:"persetujuan_data" binding:"required"` // consent UU PDP, harus true
}

// ProfilRequest: jemaat mengubah data dirinya sendiri (form lengkap).
// NIK dan data keanggotaan (tanggal bergabung/keluar) diabaikan, hanya admin yang boleh mengubah.
type ProfilRequest struct {
	Nama      string          `json:"nama" binding:"required,max=100"`
	Email     string          `json:"email" binding:"omitempty,email,max=100"`
	NoTelepon string          `json:"no_telepon" binding:"omitempty,max=20"`
	Identitas identitas.Isian `json:"identitas"`
}

// PengajuanRingkas: status pengajuan verifikasi terakhir, untuk ditampilkan di app.
type PengajuanRingkas struct {
	ID           uint      `json:"id"`
	Status       string    `json:"status"`        // MENUNGGU / DISETUJUI / DITOLAK
	CatatanAdmin *string   `json:"catatan_admin"` // alasan kalau ditolak
	DiajukanPada time.Time `json:"diajukan_pada"`
}

// ProfilResponse: GET /profil/saya. NIK tidak ikut dikirim ke app.
type ProfilResponse struct {
	ID                uint                  `json:"id"`
	KodeJemaat        *string               `json:"kode_jemaat"`
	Nama              string                `json:"nama"`
	Email             *string               `json:"email"`
	NoTelepon         *string               `json:"no_telepon"`
	StatusVerifikasi  string                `json:"status_verifikasi"`
	Identitas         identitas.Isian       `json:"identitas"`
	Kelengkapan       identitas.Kelengkapan `json:"kelengkapan"`        // untuk banner "Lengkapi data diri 2/5"
	PengajuanTerakhir *PengajuanRingkas     `json:"pengajuan_terakhir"` // null kalau belum pernah mengajukan
	// KolomTerkunci: field yang tampil read-only di app (data inti setelah TERVERIFIKASI). Kosong = semua bisa diubah.
	KolomTerkunci []string `json:"kolom_terkunci"`
}

// ---- sisi Super Admin ----

// ListQuery: ?page=&limit=&sort=&status=&q=
type ListQuery struct {
	pagination.Params
	Status string
	Q      string
}

type ListItem struct {
	ID            uint       `json:"id"`
	Status        string     `json:"status"`
	UserID        uint       `json:"id_user"`
	Nama          string     `json:"nama"`
	Email         *string    `json:"email"`
	NoTelepon     *string    `json:"no_telepon"`
	DiajukanPada  time.Time  `json:"diajukan_pada"`
	DiperiksaPada *time.Time `json:"diperiksa_pada"`
}

type ListResponse struct {
	Items []ListItem      `json:"items"`
	Meta  pagination.Meta `json:"meta"`
}

// dataDiajukan: isi kolom data_diajukan (JSON), salinan data saat jemaat mengajukan.
type dataDiajukan struct {
	Nama      string          `json:"nama"`
	Email     string          `json:"email"`
	NoTelepon string          `json:"no_telepon"`
	Identitas identitas.Isian `json:"identitas"`
}

// DetailResponse: isi pengajuan untuk diperiksa admin. Password tidak pernah ditampilkan.
type DetailResponse struct {
	ListItem
	DataDiajukan     dataDiajukan `json:"data_diajukan"`
	StatusVerifikasi string       `json:"status_verifikasi"` // status akun sekarang
	KodeJemaat       *string      `json:"kode_jemaat"`
	DiperiksaOleh    *uint        `json:"diperiksa_oleh"`
	CatatanAdmin     *string      `json:"catatan_admin"`
}

type SetujuiRequest struct {
	CatatanAdmin string `json:"catatan_admin" binding:"omitempty,max=500"`
}

type TolakRequest struct {
	CatatanAdmin string `json:"catatan_admin" binding:"required,max=500"`
}

type ProsesResponse struct {
	ID         uint    `json:"id"`
	Status     string  `json:"status"`
	UserID     uint    `json:"id_user"`
	KodeJemaat *string `json:"kode_jemaat"`
}
