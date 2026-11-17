package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sim-g-backend/config"
)

// Connect buka koneksi ke MySQL lalu ping.
// Dev: semua query masuk log + response (field "debug"). Prod: cuma error & query lambat.
func Connect(cfg *config.Config, log *slog.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{Logger: gormLogger(cfg, log)})
	if err != nil {
		return nil, fmt.Errorf("buka koneksi MySQL %s:%s/%s: %w", cfg.DBHost, cfg.DBPort, cfg.DBName, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping MySQL: %w", err)
	}
	return db, nil
}

func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func gormLogger(cfg *config.Config, log *slog.Logger) logger.Interface {
	log = log.With("component", "gorm")
	if !cfg.IsProduction() {
		return queryLogger{log: log}
	}
	return logger.NewSlogLogger(log, logger.Config{
		SlowThreshold:             500 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true, // isi parameter query (data jemaat) nggak ikut ke log
	})
}
