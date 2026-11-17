package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sim-g-backend/config"
	"sim-g-backend/database"
	"sim-g-backend/routes"
)

//      @title                                          SIM-G API
//      @version                                        1.0
//      @description                            Dokumentasi API backend SIM-G
//      @BasePath                                       /gkki/api
//      @securityDefinitions.apikey     BearerAuth
//      @in                                                     header
//      @name                                           Authorization
//      @description                            Isi dengan: Bearer <access_token>

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("gagal membaca konfigurasi", "error", err)
		os.Exit(1)
	}

	log := newLogger(cfg)
	slog.SetDefault(log)
	log.Info("konfigurasi dimuat", "env", cfg.AppEnv, "port", cfg.AppPort, "timezone", cfg.AppTimezone)

	db, err := database.Connect(cfg, log)
	if err != nil {
		log.Error("gagal terhubung ke database", "error", err)
		os.Exit(1)
	}
	defer database.Close(db)
	log.Info("database terhubung", "host", cfg.DBHost, "name", cfg.DBName)

	// go run . seed -> isi role default + Super Admin pertama, lalu selesai
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		if err := database.Seed(db, log); err != nil {
			log.Error("seed gagal", "error", err)
			os.Exit(1)
		}
		log.Info("seed selesai")
		return
	}

	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           routes.New(cfg, db, log),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second, // agak lama buat upload/export excel
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Info("server berjalan", "addr", "http://localhost:"+cfg.AppPort+routes.BasePath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server berhenti tiba-tiba", "error", err)
			os.Exit(1)
		}
	}()

	// tunggu Ctrl+C / docker stop, request yang lagi jalan diselesaiin dulu
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info("mematikan server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown tidak bersih", "error", err)
	}
	log.Info("server berhenti")
}

// dev pakai format teks biar gampang dibaca, prod pakai JSON
func newLogger(cfg *config.Config) *slog.Logger {
	if cfg.IsProduction() {
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}
