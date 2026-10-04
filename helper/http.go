package helper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"laundry-api/app/model"
)

type APIError struct {
	Status        int
	Code, Message string
	Fields        map[string]string
}

func (e *APIError) Error() string { return e.Message }
func Error(status int, code, message string) error {
	return &APIError{Status: status, Code: code, Message: message}
}
func Validation(fields map[string]string) error {
	return &APIError{Status: 422, Code: "VALIDATION_ERROR", Message: "Data tidak valid", Fields: fields}
}
func BadState(message string) error { return Error(409, "INVALID_ORDER_STATE", message) }

func ErrorHandler(c *fiber.Ctx, err error) error {
	e := &APIError{Status: 500, Code: "INTERNAL_ERROR", Message: "Terjadi kesalahan pada server"}
	var apiErr *APIError
	var fiberErr *fiber.Error
	switch {
	case errors.As(err, &apiErr):
		e = apiErr
	case errors.Is(err, model.ErrNotFound):
		e = &APIError{Status: 404, Code: "NOT_FOUND", Message: "Data tidak ditemukan"}
	case errors.Is(err, model.ErrConflict):
		e = &APIError{Status: 409, Code: "CONFLICT", Message: "Data duplikat atau relasi data tidak valid"}
	case errors.Is(err, model.ErrInvalidToken):
		e = &APIError{Status: 401, Code: "INVALID_REFRESH_TOKEN", Message: "Refresh token tidak aktif"}
	case errors.As(err, &fiberErr):
		e = &APIError{Status: fiberErr.Code, Code: "HTTP_ERROR", Message: fiberErr.Message}
	}
	id := c.GetRespHeader("X-Request-ID")
	if e.Status >= 500 {
		log.Printf("request_id=%s error=%v", id, err)
	}
	if e.Status == 401 {
		c.Set("WWW-Authenticate", `Bearer realm="laundry-api"`)
	}
	body := fiber.Map{"success": false, "code": e.Code, "message": e.Message, "request_id": id}
	if len(e.Fields) > 0 {
		body["errors"] = e.Fields
	}
	return c.Status(e.Status).JSON(body)
}

func OK(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(fiber.Map{"success": true, "message": message, "data": data})
}

// Hanya field request yang dikenal diterima; role, harga total, dan pemilik
// tidak dapat disisipkan ke request yang tidak mengizinkannya.
func Decode(c *fiber.Ctx, target any) error {
	mediaType, _, err := mime.ParseMediaType(c.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return Error(415, "UNSUPPORTED_MEDIA_TYPE", "Content-Type harus application/json")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(c.Body(), &fields); err != nil || fields == nil {
		return Error(400, "INVALID_JSON", "Body harus berupa objek JSON")
	}
	for k, v := range fields {
		if bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return Validation(map[string]string{k: "Tidak boleh null; hapus field bila tidak ingin mengubahnya"})
		}
	}
	dec := json.NewDecoder(bytes.NewReader(c.Body()))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return Error(400, "INVALID_JSON", "Field atau tipe data JSON tidak sesuai kontrak endpoint")
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return Error(400, "INVALID_JSON", "Body hanya boleh memuat satu objek JSON")
	}
	return nil
}
func ID(c *fiber.Ctx) (int64, error) {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, Error(400, "INVALID_ID", "ID harus angka positif")
	}
	return id, nil
}
func DBContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}
func CurrentUser(c *fiber.Ctx) model.User { u, _ := c.Locals("user").(model.User); return u }
