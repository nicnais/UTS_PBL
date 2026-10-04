package middleware

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"laundry-api/app/model"
	"laundry-api/app/repository"
	"laundry-api/helper"
)

func RequireAuth(store repository.Store, jwt *helper.JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		parts := strings.Fields(c.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return helper.Error(401, "AUTH_REQUIRED", "Gunakan Authorization: Bearer <access_token>")
		}
		id, err := jwt.Verify(parts[1])
		if err != nil {
			return err
		}
		ctx, cancel := helper.DBContext(c)
		defer cancel()
		// Role dan permission dibaca melalui interface repository, sehingga perubahan
		// hak akses berlaku pada request berikutnya tanpa menunggu JWT kedaluwarsa.
		user, err := store.UserByID(ctx, id)
		if errors.Is(err, model.ErrNotFound) {
			return helper.Error(401, "INVALID_TOKEN", "Pengguna token tidak ditemukan")
		}
		if err != nil {
			return err
		} // Fail closed saat database gagal.
		c.Locals("user", user)
		c.Set("Cache-Control", "no-store")
		return c.Next()
	}
}
func RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !helper.CurrentUser(c).Can(permission) {
			return helper.Error(403, "FORBIDDEN", "Anda tidak memiliki izin untuk tindakan ini")
		}
		return c.Next()
	}
}
func AuthLimiter(max int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max: max, Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string { return c.IP() },
		LimitReached: func(c *fiber.Ctx) error {
			c.Set("Retry-After", "60")
			return helper.Error(429, "RATE_LIMITED", "Terlalu banyak percobaan; tunggu satu menit")
		},
	})
}