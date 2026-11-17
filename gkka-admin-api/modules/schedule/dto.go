package schedule

import (
	"time"

	"sim-g-backend/helpers/pagination"
)

// ImportRow adalah satu baris kegiatan hasil baca sheet "Kegiatan" di template Excel.
type ImportRow struct {
	Line          int    `json:"line"` // nomor baris di Excel (baris header = 1)
	Date          string `json:"tanggal,omitempty"`
	WaktuMulai    string `json:"waktu_mulai,omitempty"`   // "07:30"
	WaktuSelesai  string `json:"waktu_selesai,omitempty"` // kosong kalau tidak diisi
	Activity      string `json:"nama_kegiatan"`
	JenisKegiatan string `json:"jenis_kegiatan,omitempty"` // KEBAKTIAN / TRAINING / EVENT, belum disimpan di database
	Tema          string `json:"tema,omitempty"`
	NatsAlkitab   string `json:"nats_alkitab,omitempty"`
	Pembicara     string `json:"pembicara,omitempty"`
	Lokasi        string `json:"lokasi,omitempty"`
	Deskripsi     string `json:"deskripsi,omitempty"`
	Komunitas     string `json:"komunitas,omitempty"`
	PerluEvaluasi bool   `json:"perlu_evaluasi"`

	// Start dan End dipakai di Go saja (deteksi bentrok), tidak dikirim ke client
	Start *time.Time `json:"-"`
	End   *time.Time `json:"-"` // nil kalau waktu_selesai kosong

	// Assignee dan Skill dari template lama, dipakai detectImportClashes
	Assignee string `json:"petugas,omitempty"`
	Skill    string `json:"skill,omitempty"`

	Errors map[string]string `json:"errors,omitempty"` // kolom -> pesan error
}

// Valid: baris tidak punya error validasi.
func (r ImportRow) Valid() bool { return len(r.Errors) == 0 }

// ClashInfo adalah satu peringatan bentrok, dilaporkan dengan nomor baris Excel.
type ClashInfo struct {
	Type    ClashType `json:"type"`
	Lines   []int     `json:"lines"`
	Message string    `json:"message"`
}

// PreviewResponse adalah hasil POST /schedules/import/preview. Tidak ada data yang disimpan.
type PreviewResponse struct {
	TotalRows   int         `json:"total_rows"`
	ValidRows   int         `json:"valid_rows"`
	InvalidRows int         `json:"invalid_rows"`
	CanCommit   bool        `json:"can_commit"` // true jika tidak ada baris yang error (bentrok hanya peringatan)
	Rows        []ImportRow `json:"rows"`
	Clashes     []ClashInfo `json:"clashes"`
}

// KegiatanRequest dipakai untuk tambah dan ubah kegiatan (input manual dari web admin).
type KegiatanRequest struct {
	Nama          string `json:"nama_kegiatan" binding:"required,max=150"`
	Tanggal       string `json:"tanggal" binding:"required"`     // contoh "2026-11-01"
	WaktuMulai    string `json:"waktu_mulai" binding:"required"` // contoh "07:00"
	WaktuSelesai  string `json:"waktu_selesai"`                  // boleh kosong
	Tema          string `json:"tema" binding:"max=255"`
	NatsAlkitab   string `json:"nats_alkitab" binding:"max=100"`
	Pembicara     string `json:"pembicara" binding:"max=100"`
	Lokasi        string `json:"lokasi" binding:"max=150"`
	Deskripsi     string `json:"deskripsi"`
	KomunitasID   *uint  `json:"id_komunitas"` // kosong = kegiatan umum
	PerluEvaluasi bool   `json:"perlu_evaluasi"`
}

// KegiatanQuery: ?dari=2026-11-01&sampai=2026-11-30&q=umum&page=1&limit=20
type KegiatanQuery struct {
	pagination.Params
	Dari   string
	Sampai string
	Q      string
}

type KegiatanResponse struct {
	ID            uint    `json:"id"`
	Nama          string  `json:"nama_kegiatan"`
	Tanggal       string  `json:"tanggal"`       // "2026-11-01"
	WaktuMulai    string  `json:"waktu_mulai"`   // "07:00"
	WaktuSelesai  *string `json:"waktu_selesai"` // null kalau tidak diisi
	Tema          *string `json:"tema"`
	NatsAlkitab   *string `json:"nats_alkitab"`
	Pembicara     *string `json:"pembicara"`
	Lokasi        *string `json:"lokasi"`
	Deskripsi     *string `json:"deskripsi"`
	KomunitasID   *uint   `json:"id_komunitas"`
	NamaKomunitas *string `json:"nama_komunitas"`
	PerluEvaluasi bool    `json:"perlu_evaluasi"`
}

type KegiatanListResponse struct {
	Items []KegiatanResponse `json:"items"`
	Meta  pagination.Meta    `json:"meta"`
}

// ImportSimpanResponse adalah hasil POST /schedules/import.
type ImportSimpanResponse struct {
	Ditambah        int `json:"ditambah"`
	Diperbarui      int `json:"diperbarui"`
	KuesionerDibuat int `json:"kuesioner_dibuat"` // kuesioner evaluasi yang otomatis dibuat untuk KEBAKTIAN
}

// PenugasanRequest dipakai admin untuk menugaskan jemaat ke sebuah kegiatan.
type PenugasanRequest struct {
	UserID     uint `json:"id_user" binding:"required"`
	KeahlianID uint `json:"id_keahlian" binding:"required"`
}

// PenugasanResponse: satu petugas pada satu kegiatan.
// Peringatan hanya terisi saat penugasan baru dibuat (mis. bentrok waktu); penugasan tetap tersimpan.
type PenugasanResponse struct {
	ID           uint     `json:"id_penjadwalan"`
	KegiatanID   uint     `json:"id_kegiatan"`
	UserID       uint     `json:"id_user"`
	KodeJemaat   *string  `json:"kode_jemaat"` // null kalau belum terverifikasi
	Nama         string   `json:"nama"`
	KeahlianID   uint     `json:"id_keahlian"`
	NamaKeahlian string   `json:"nama_keahlian"`
	Status       string   `json:"status_konfirmasi"`
	Peringatan   []string `json:"peringatan,omitempty"`
}

type PetugasListResponse struct {
	Items []PenugasanResponse `json:"items"`
}

// JadwalSayaQuery: ?dari=2026-11-01&sampai=2026-11-30&page=1&limit=20
type JadwalSayaQuery struct {
	pagination.Params
	Dari   string
	Sampai string
}

// JadwalSayaResponse: satu tugas pelayanan milik jemaat yang sedang login.
type JadwalSayaResponse struct {
	ID           uint    `json:"id_penjadwalan"`
	KegiatanID   uint    `json:"id_kegiatan"`
	NamaKegiatan string  `json:"nama_kegiatan"`
	Tanggal      string  `json:"tanggal"`       // "2026-11-01"
	WaktuMulai   string  `json:"waktu_mulai"`   // "07:00"
	WaktuSelesai *string `json:"waktu_selesai"` // null kalau tidak diisi
	Lokasi       *string `json:"lokasi"`
	KeahlianID   uint    `json:"id_keahlian"`
	NamaKeahlian string  `json:"nama_keahlian"`
	Status       string  `json:"status_konfirmasi"`
}

type JadwalSayaListResponse struct {
	Items []JadwalSayaResponse `json:"items"`
	Meta  pagination.Meta      `json:"meta"`
}
