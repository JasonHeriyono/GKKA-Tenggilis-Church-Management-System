package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/debuginfo"
)

// Format response semua endpoint:
//
//	{ "success": true, "data": {...}, "message": "", "errors": null }
//
// Khusus APP_ENV=development ada tambahan "debug": request_id, detail error asli, dan semua query SQL.
type Body struct {
	Success bool               `json:"success"`
	Data    any                `json:"data"`
	Message string             `json:"message"`
	Errors  map[string]string  `json:"errors"`
	Debug   *debuginfo.Catatan `json:"debug,omitempty"`
}

func OK(c *gin.Context, data any, message string) {
	c.JSON(http.StatusOK, Body{Success: true, Data: data, Message: message, Debug: debuginfo.Salin(c.Request.Context(), nil)})
}

func Created(c *gin.Context, data any, message string) {
	c.JSON(http.StatusCreated, Body{Success: true, Data: data, Message: message, Debug: debuginfo.Salin(c.Request.Context(), nil)})
}

// Error 5xx dicatat ke log, client cuma dapat pesan umum (di dev, error aslinya ada di "debug").
func Error(c *gin.Context, err error) {
	appErr := apperror.From(err)
	if appErr.Status >= http.StatusInternalServerError {
		slog.ErrorContext(c.Request.Context(), "internal error",
			"request_id", c.GetString("request_id"),
			"path", c.Request.URL.Path,
			"error", err,
		)
	}
	c.AbortWithStatusJSON(appErr.Status, Body{
		Success: false,
		Message: appErr.Message,
		Errors:  appErr.Fields,
		Debug:   debuginfo.Salin(c.Request.Context(), err),
	})
}

// BindError dipanggil kalau ShouldBindJSON gagal.
// Gagal validasi / tipe data salah -> 422 per field, JSON rusak -> 400.
func BindError(c *gin.Context, err error) {
	// contoh: {"id_user": "abc"} padahal id_user harus angka
	var errTipe *json.UnmarshalTypeError
	if errors.As(err, &errTipe) && errTipe.Field != "" {
		fieldErr := apperror.Validation(map[string]string{errTipe.Field: "harus bertipe " + namaTipe(errTipe.Type.Kind())})
		fieldErr.Err = err
		Error(c, fieldErr)
		return
	}

	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		badRequest := apperror.BadRequest("Format request tidak valid")
		badRequest.Err = err // detail JSON rusak-nya kelihatan di "debug" saat dev
		Error(c, badRequest)
		return
	}
	fields := make(map[string]string, len(verrs))
	for _, fe := range verrs {
		fields[fe.Field()] = validationMessage(fe)
	}
	Error(c, apperror.Validation(fields))
}

// nama field di "errors" pakai nama json (email, full_name), bukan nama struct Go
func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(f reflect.StructField) string {
			name := strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
			if name == "" || name == "-" {
				return f.Name
			}
			return name
		})
	}
}

// namaTipe: nama tipe JSON yang gampang dipahami admin/frontend.
func namaTipe(kind reflect.Kind) string {
	switch kind {
	case reflect.Bool:
		return "boolean (true/false)"
	case reflect.String:
		return "teks"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Struct, reflect.Map:
		return "object"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "angka bulat"
	default:
		return "angka"
	}
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		return "minimal " + fe.Param()
	case "max":
		return "maksimal " + fe.Param()
	case "oneof":
		return "harus salah satu dari: " + fe.Param()
	default:
		return "tidak valid (" + fe.Tag() + ")"
	}
}
