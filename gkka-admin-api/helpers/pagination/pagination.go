package pagination

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"sim-g-backend/helpers/apperror"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
	MaxPage      = 100000 // page di atas ini ditolak, biar (page-1)*limit tidak overflow jadi minus
)

// Params dibaca dari query string: ?page=1&limit=20&sort=-nama
type Params struct {
	Page   int
	Limit  int
	Offset int
	Sort   string
}

// Meta dikirim bersama data pada response berhalaman.
type Meta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// FromQuery membaca page, limit, dan sort. Page bukan angka / kurang dari 1 / terlalu besar -> 400.
func FromQuery(c *gin.Context) (Params, error) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 || page > MaxPage {
		return Params{}, apperror.BadRequest("Parameter page harus angka 1 sampai " + strconv.Itoa(MaxPage))
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(DefaultLimit)))
	if limit < 1 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	return Params{Page: page, Limit: limit, Offset: (page - 1) * limit, Sort: c.Query("sort")}, nil
}

// Order mengubah ?sort= menjadi klausa ORDER BY yang aman: hanya kolom di allowed yang
// diterima ("-nama" = DESC), selain itu dipakai fallback. Mencegah SQL injection lewat sort.
func (p Params) Order(allowed []string, fallback string) string {
	field := strings.TrimPrefix(p.Sort, "-")
	if field == "" {
		return fallback
	}
	for _, a := range allowed {
		if a == field {
			if strings.HasPrefix(p.Sort, "-") {
				return field + " DESC"
			}
			return field + " ASC"
		}
	}
	return fallback
}

func NewMeta(p Params, total int64) Meta {
	pages := 0
	if p.Limit > 0 {
		pages = int((total + int64(p.Limit) - 1) / int64(p.Limit))
	}
	return Meta{Page: p.Page, Limit: p.Limit, Total: total, TotalPages: pages}
}

// CekHalaman: page melebihi halaman terakhir (meta.total_pages) -> 404.
// Kalau data kosong, page 1 tetap boleh (hasilnya list kosong).
func CekHalaman(p Params, total int64) error {
	halamanTerakhir := max(NewMeta(p, total).TotalPages, 1)
	if p.Page > halamanTerakhir {
		return apperror.NotFound("Halaman " + strconv.Itoa(p.Page) + " tidak ada, halaman terakhir " + strconv.Itoa(halamanTerakhir))
	}
	return nil
}
