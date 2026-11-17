package debuginfo

import (
	"context"
	"sync"
)

// Catatan hanya ada kalau middleware.Debug dipasang (APP_ENV=development), jadi production aman.

// Query: satu query SQL yang dijalankan selama request (SQL-nya sudah berisi nilai, siap di-copy ke MySQL).
type Query struct {
	SQL      string  `json:"sql"`
	Rows     int64   `json:"rows"`
	DurasiMs float64 `json:"durasi_ms"`
	Error    string  `json:"error,omitempty"`
}

// Catatan: info debug milik satu request, dikirim di field "debug" response (khusus dev).
type Catatan struct {
	RequestID string  `json:"request_id"`
	Error     string  `json:"error,omitempty"`
	Queries   []Query `json:"queries"`
	mu        sync.Mutex
}

type ctxKey struct{}

// Mulai menempelkan catatan kosong ke context request (dipanggil middleware.Debug).
func Mulai(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, &Catatan{RequestID: requestID, Queries: []Query{}})
}

// Dari mengambil catatan dari context; nil kalau mode debug mati.
func Dari(ctx context.Context) *Catatan {
	catatan, _ := ctx.Value(ctxKey{}).(*Catatan)
	return catatan
}

// RequestID dari context, kosong kalau mode debug mati.
func RequestID(ctx context.Context) string {
	if catatan := Dari(ctx); catatan != nil {
		return catatan.RequestID
	}
	return ""
}

// TambahQuery dipanggil logger GORM setiap ada query. Pakai mutex karena query bisa jalan bersamaan.
func TambahQuery(ctx context.Context, query Query) {
	catatan := Dari(ctx)
	if catatan == nil {
		return
	}
	catatan.mu.Lock()
	defer catatan.mu.Unlock()
	catatan.Queries = append(catatan.Queries, query)
}

// Salin membuat salinan catatan untuk dikirim ke client (nil kalau mode debug mati).
func Salin(ctx context.Context, err error) *Catatan {
	catatan := Dari(ctx)
	if catatan == nil {
		return nil
	}
	catatan.mu.Lock()
	defer catatan.mu.Unlock()
	salinan := &Catatan{RequestID: catatan.RequestID, Queries: append([]Query{}, catatan.Queries...)}
	if err != nil {
		salinan.Error = err.Error()
	}
	return salinan
}
