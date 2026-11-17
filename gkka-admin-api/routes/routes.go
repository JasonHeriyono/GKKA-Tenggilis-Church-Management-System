package routes

import (
	"log/slog"
	"net/http"
	"sim-g-backend/config"
	_ "sim-g-backend/docs" // daftarkan hasil generate swag
	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/response"
	"sim-g-backend/middleware"
	"sim-g-backend/models"
	"sim-g-backend/modules/auth"
	"sim-g-backend/modules/evaluation"
	"sim-g-backend/modules/group"
	"sim-g-backend/modules/pendaftaran"
	"sim-g-backend/modules/pengaturan"
	"sim-g-backend/modules/schedule"
	"sim-g-backend/modules/skill"
	"sim-g-backend/modules/user"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// prefix sementara
const BasePath = "/gkki/api"

func New(cfg *config.Config, db *gorm.DB, log *slog.Logger) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.HandleMethodNotAllowed = true
	// tanpa ini X-Forwarded-For bisa dipalsukan buat ngakalin rate limit
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Error("TRUSTED_PROXIES tidak valid", "error", err)
	}
	r.Use(
		middleware.RequestID(),
		middleware.Logger(log),
		middleware.Recovery(log),
		middleware.CORS(cfg.CORSAllowedOrigins),
	)
	// dev: setiap response ikut membawa query SQL + detail error di field "debug"
	if !cfg.IsProduction() {
		r.Use(middleware.Debug())
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	r.NoRoute(func(c *gin.Context) { response.Error(c, apperror.NotFound("Endpoint tidak ditemukan")) })
	r.NoMethod(func(c *gin.Context) {
		response.Error(c, apperror.New(http.StatusMethodNotAllowed, "Method tidak diizinkan"))
	})

	loc, _ := time.LoadLocation(cfg.AppTimezone) // sudah dicek di config.validate()

	// autentikasi dan hak akses (RBAC)
	authSvc := auth.NewService(auth.NewRepository(db), cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	requireLogin := middleware.Auth(authSvc.ParseAccessToken)
	superAdmin := middleware.RequireRole(models.RoleSuperAdmin)
	majelisAtauAdmin := middleware.RequireRole(models.RoleMajelis, models.RoleSuperAdmin) // baca laporan saja

	// Google Form: kalau gagal disiapkan, server tetap jalan, hanya fitur sinkron Google yang mati
	googleForm, err := evaluation.NewGoogleForm(cfg.GoogleCredentialsFile, cfg.GoogleFormMingguan, cfg.GoogleFormTahunan)
	if err != nil {
		log.Warn("Google Form tidak bisa dihubungkan", "error", err)
		googleForm = nil
	}

	api := r.Group(BasePath)
	{
		api.GET("/health", healthHandler(cfg, db))

		auth.RegisterRoutes(api, auth.NewHandler(authSvc), requireLogin)

		// signup publik (langsung masuk lewat authSvc); /profil/saya dan /verifikasi untuk yang login;
		// /pendaftaran/* (pemeriksaan verifikasi) khusus Super Admin
		pendaftaran.RegisterRoutes(api, pendaftaran.NewHandler(pendaftaran.NewService(pendaftaran.NewRepository(db), authSvc)), requireLogin, superAdmin)

		// khusus Super Admin
		user.RegisterRoutes(api, user.NewHandler(user.NewService(user.NewRepository(db))), requireLogin, superAdmin)

		// evaluasi dibuat duluan: import kegiatan (schedule) memakainya untuk membuat kuesioner otomatis
		evaluationSvc := evaluation.NewService(evaluation.NewRepository(db), googleForm, loc)

		// GET daftar publik (tanpa login), selain itu khusus Super Admin
		schedule.RegisterRoutes(api, schedule.NewService(schedule.NewRepository(db), loc, evaluationSvc), requireLogin, superAdmin) // GET /kegiatan, /kegiatan/:id
		group.RegisterRoutes(api, group.NewHandler(group.NewService(group.NewRepository(db))), requireLogin, superAdmin)            // GET /komunitas
		skill.RegisterRoutes(api, skill.NewHandler(skill.NewService(skill.NewRepository(db))), requireLogin, superAdmin)            // GET /keahlian (form signup)

		// pengaturan admin (lama buka form, jam tutup, dll): khusus Super Admin
		pengaturan.RegisterRoutes(api, pengaturan.NewHandler(pengaturan.NewService(pengaturan.NewRepository(db))), requireLogin, superAdmin)

		// pertanyaan: GET untuk Majelis dan Super Admin, ubah data khusus Super Admin
		evaluation.RegisterRoutes(api, evaluation.NewHandler(evaluationSvc), requireLogin, majelisAtauAdmin, superAdmin)
	}

	return r
}

// buat cek server & db masih hidup
func healthHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err == nil {
			err = sqlDB.PingContext(c.Request.Context())
		}
		if err != nil {
			data := gin.H{"status": "error", "env": cfg.AppEnv, "db": "down"}
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, response.Body{Success: false, Data: data, Message: "Database tidak dapat dihubungi"})
			return
		}
		response.OK(c, gin.H{"status": "ok", "env": cfg.AppEnv, "db": "up"}, "")
	}
}
