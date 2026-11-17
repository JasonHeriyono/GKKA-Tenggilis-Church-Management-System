package database

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"sim-g-backend/models"
)

const minSeedPasswordLen = 12

var roleDescriptions = map[string]string{
	models.RoleGuest:      "Guest: pengunjung yang belum terdaftar sebagai jemaat.",
	models.RoleJemaat:     "Jemaat gereja: melihat jadwal dan riwayat sendiri, mengisi kuesioner.",
	models.RoleMajelis:    "Majelis: melihat laporan dan dashboard evaluasi (hanya baca).",
	models.RoleSuperAdmin: "Super Admin: mengelola seluruh data dan akun.",
}

// Seed dijalankan oleh `go run . seed`. Idempoten: aman dijalankan berulang.
//   - membuat role default (Jemaat, Majelis, Super Admin) dan keahlian sistem Usher bila belum ada
//   - membuat akun Super Admin pertama bila belum ada Super Admin sama sekali,
//     dari env SEED_ADMIN_EMAIL dan SEED_ADMIN_PASSWORD (opsional: SEED_ADMIN_KODE, SEED_ADMIN_NAMA)
func Seed(db *gorm.DB, log *slog.Logger) error {
	return db.Transaction(func(tx *gorm.DB) error {
		roleIDs := map[string]uint{}
		for _, name := range models.AllRoles() {
			desc := roleDescriptions[name]
			role := models.Role{Nama: name, Deskripsi: &desc}
			if err := tx.Where("nama_role = ?", name).FirstOrCreate(&role).Error; err != nil {
				return fmt.Errorf("seed role %q: %w", name, err)
			}
			roleIDs[name] = role.ID
		}
		log.Info("role default siap", "jumlah", len(roleIDs))

		// keahlian bawaan sistem: Usher (tidak bisa diubah/dihapus lewat API)
		usher := models.Keahlian{Nama: "Usher", IsSistem: true}
		if err := tx.Where("nama_keahlian = ?", usher.Nama).FirstOrCreate(&usher).Error; err != nil {
			return fmt.Errorf("seed keahlian %q: %w", usher.Nama, err)
		}

		var admins int64
		if err := tx.Model(&models.User{}).Where("id_role = ?", roleIDs[models.RoleSuperAdmin]).Count(&admins).Error; err != nil {
			return err
		}
		if admins > 0 {
			log.Info("Super Admin sudah ada, dilewati")
			return nil
		}

		email := strings.TrimSpace(os.Getenv("SEED_ADMIN_EMAIL"))
		password := os.Getenv("SEED_ADMIN_PASSWORD")
		if email == "" || password == "" {
			return errors.New("isi SEED_ADMIN_EMAIL dan SEED_ADMIN_PASSWORD di .env untuk membuat Super Admin pertama")
		}
		if len(password) < minSeedPasswordLen || len(password) > 72 {
			return fmt.Errorf("SEED_ADMIN_PASSWORD harus 12 sampai 72 karakter")
		}
		kode := envOr("SEED_ADMIN_KODE", "ADMIN001")
		nama := envOr("SEED_ADMIN_NAMA", "Super Admin")

		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		h := string(hash)
		admin := models.User{
			KodeJemaat:   &kode,
			Nama:         nama,
			Email:        &email,
			PasswordHash: &h,
			RoleID:       roleIDs[models.RoleSuperAdmin],
			IsAktif:      true,
			// admin pertama dianggap jemaat resmi
			StatusVerifikasi: models.Terverifikasi,
		}
		if err := tx.Omit("Role").Create(&admin).Error; err != nil {
			return fmt.Errorf("buat Super Admin: %w", err)
		}
		log.Info("Super Admin pertama dibuat", "kode_jemaat", kode, "email", email)
		return nil
	})
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
