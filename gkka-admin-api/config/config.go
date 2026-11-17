package config

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"

	// nilai contoh dari .env.example, jangan sampai kepakai beneran
	placeholderJWTSecret = "ganti-dengan-string-acak-panjang"
	minJWTSecretLength   = 32
)

// Config isinya diambil dari .env, nama variabel samain dengan .env.example
type Config struct {
	AppEnv      string `env:"APP_ENV" envDefault:"development"`
	AppPort     string `env:"APP_PORT" envDefault:"8080"`
	AppTimezone string `env:"APP_TIMEZONE" envDefault:"Asia/Jakarta"`

	DBHost     string `env:"DB_HOST" envDefault:"localhost"`
	DBPort     string `env:"DB_PORT" envDefault:"3306"`
	DBUser     string `env:"DB_USER" envDefault:"root"`
	DBPassword string `env:"DB_PASSWORD"`
	DBName     string `env:"DB_NAME" envDefault:"sim_g"`

	JWTSecret     string        `env:"JWT_SECRET,required,notEmpty"` // sengaja tanpa default
	JWTAccessTTL  time.Duration `env:"JWT_ACCESS_TTL" envDefault:"15m"`
	JWTRefreshTTL time.Duration `env:"JWT_REFRESH_TTL" envDefault:"168h"`

	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" envSeparator:"," envDefault:"http://localhost:3000"`
	TrustedProxies     []string `env:"TRUSTED_PROXIES" envSeparator:","` // IP reverse proxy (Nginx), kosong = konek langsung

	// Google Form: kosongkan form ID kalau belum mau dihubungkan ke Google
	GoogleCredentialsFile string `env:"GOOGLE_CREDENTIALS_FILE" envDefault:"service-account.json"`
	GoogleFormMingguan    string `env:"GOOGLE_FORM_ID_MINGGUAN"`
	GoogleFormTahunan     string `env:"GOOGLE_FORM_ID_TAHUNAN"`
}

// Load baca .env kalau ada. Env dari sistem tetap didahulukan.
func Load() (*Config, error) {
	_ = godotenv.Load()

	var c Config
	if err := env.Parse(&c); err != nil {
		return nil, fmt.Errorf("baca environment: %w", err)
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) validate() error {
	// Validasi APP_ENV, APP_TIMEZONE, JWT_SECRET, dan DB_PASSWORD.
	// APP_ENV harus development atau production.
	if c.AppEnv != EnvDevelopment && c.AppEnv != EnvProduction {
		return fmt.Errorf("APP_ENV harus %q atau %q, bukan %q", EnvDevelopment, EnvProduction, c.AppEnv)
	}
	if _, err := time.LoadLocation(c.AppTimezone); err != nil {
		return fmt.Errorf("APP_TIMEZONE tidak dikenal: %q", c.AppTimezone)
	}

	// JWT_SECRET dicek di dev juga, biar nggak kebiasaan pakai secret asal
	if c.JWTSecret == placeholderJWTSecret || len(c.JWTSecret) < minJWTSecretLength {
		return fmt.Errorf("JWT_SECRET wajib diganti dengan string acak minimal %d karakter (contoh: openssl rand -hex 32)", minJWTSecretLength)
	}

	for _, p := range c.TrustedProxies {
		if net.ParseIP(p) == nil {
			if _, _, err := net.ParseCIDR(p); err != nil {
				return fmt.Errorf("TRUSTED_PROXIES berisi IP/CIDR tidak valid: %q", p)
			}
		}
	}

	// form ID diambil dari URL editor: docs.google.com/forms/d/<ID>/edit
	for _, formID := range []string{c.GoogleFormMingguan, c.GoogleFormTahunan} {
		if strings.Contains(formID, "/") {
			return fmt.Errorf("GOOGLE_FORM_ID_* cukup diisi ID-nya saja, bukan URL lengkap: %q", formID)
		}
	}

	if c.IsProduction() {
		// Di production, DB_PASSWORD harus ada
		if c.DBPassword == "" {
			return errors.New("DB_PASSWORD wajib diisi di production")
		}
		// Di production jangan pakai root, bikin user khusus
		if c.DBUser == "root" {
			return errors.New("DB_USER tidak boleh root di production")
		}
		// Di production CORS cuma buat domain web admin
		for _, o := range c.CORSAllowedOrigins {
			if o == "*" || strings.Contains(o, "localhost") || strings.Contains(o, "127.0.0.1") {
				return fmt.Errorf("CORS_ALLOWED_ORIGINS tidak boleh berisi %q di production", o)
			}
		}
	}
	return nil
}

func (c *Config) IsProduction() bool { return c.AppEnv == EnvProduction }

// DSN = alamat koneksi MySQL, disusun FormatDSN() biar formatnya pasti benar.
func (c *Config) DSN() string {
	loc, _ := time.LoadLocation(c.AppTimezone) // sudah dicek di validate()
	m := mysql.NewConfig()
	m.User = c.DBUser
	m.Passwd = c.DBPassword
	m.Net = "tcp"
	m.Addr = c.DBHost + ":" + c.DBPort
	m.DBName = c.DBName
	m.ParseTime = true                                 // DATE/DATETIME langsung jadi time.Time
	m.Loc = loc                                        // waktu dari DB dianggap WIB
	m.Params = map[string]string{"charset": "utf8mb4"} // support emoji & semua karakter
	return m.FormatDSN()
}
