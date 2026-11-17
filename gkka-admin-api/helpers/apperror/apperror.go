package apperror

import (
	"errors"
	"net/http"
)

// AppError dikembalikan dari service, nanti handler tinggal lempar ke response.Error.
type AppError struct {
	Status  int
	Message string
	Fields  map[string]string // error per field, buat validasi
	Err     error             // cuma buat log, nggak dikirim ke client
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func New(status int, message string) *AppError {
	return &AppError{Status: status, Message: message}
}

func BadRequest(message string) *AppError {
	return New(http.StatusBadRequest, message)
}
func Unauthorized(message string) *AppError {
	return New(http.StatusUnauthorized, message)
}
func Forbidden(message string) *AppError {
	return New(http.StatusForbidden, message)
}
func NotFound(message string) *AppError {
	return New(http.StatusNotFound, message)
}
func Conflict(message string) *AppError {
	return New(http.StatusConflict, message)
}

func Validation(fields map[string]string) *AppError {
	return &AppError{Status: http.StatusUnprocessableEntity, Message: "Data tidak valid", Fields: fields}
}

// Internal buat error yang nggak terduga, client cuma dapat pesan umum.
func Internal(err error) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Message: "Terjadi kesalahan pada server", Err: err}
}

// error biasa (bukan AppError) dianggap 500
func From(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal(err)
}
