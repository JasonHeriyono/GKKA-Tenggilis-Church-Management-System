package database

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sim-g-backend/helpers/debuginfo"
)

// queryLogger (khusus dev) mencatat SETIAP query lengkap dengan nilainya + request_id,
// supaya query di log bisa dicocokkan dengan request yang memanggilnya.
type queryLogger struct {
	log *slog.Logger
}

func (l queryLogger) LogMode(logger.LogLevel) logger.Interface { return l }

func (l queryLogger) Info(ctx context.Context, pesan string, data ...any) {
	l.log.InfoContext(ctx, pesan, "data", data)
}

func (l queryLogger) Warn(ctx context.Context, pesan string, data ...any) {
	l.log.WarnContext(ctx, pesan, "data", data)
}

func (l queryLogger) Error(ctx context.Context, pesan string, data ...any) {
	l.log.ErrorContext(ctx, pesan, "data", data)
}

// Trace dipanggil GORM setelah setiap query selesai.
func (l queryLogger) Trace(ctx context.Context, mulai time.Time, ambilSQL func() (string, int64), err error) {
	sql, rows := ambilSQL()
	durasiMs := float64(time.Since(mulai).Microseconds()) / 1000

	query := debuginfo.Query{SQL: sql, Rows: rows, DurasiMs: durasiMs}
	level := slog.LevelDebug
	// "record not found" itu normal (mis. cek data ada atau tidak), jadi bukan error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		query.Error = err.Error()
		level = slog.LevelError
	}
	debuginfo.TambahQuery(ctx, query)

	l.log.Log(ctx, level, "query",
		"request_id", debuginfo.RequestID(ctx),
		"sql", sql,
		"rows", rows,
		"durasi_ms", durasiMs,
		"error", query.Error,
	)
}
