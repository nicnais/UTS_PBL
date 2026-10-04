package config

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/jackc/pgx/v5/pgxpool"
	"laundry-api/app/repository"
	"laundry-api/app/service"
	"laundry-api/helper"
	"laundry-api/route"
)

func NewApp(pool *pgxpool.Pool, settings Settings) (*fiber.App, error) {
	store := repository.New(pool)
	jwt := helper.NewJWT(settings.Secret, settings.Issuer, settings.AccessTTL)
	auth, err := service.NewAuth(store, jwt, settings.RefreshTTL)
	if err != nil {
		return nil, err
	}
	laundry := service.NewLaundry(store)
	app := fiber.New(fiber.Config{AppName: "Laundry API", BodyLimit: 1024 * 1024, ErrorHandler: helper.ErrorHandler})
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{Format: "${time} ${method} ${path} ${status} ${latency}\n"}))
	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(cors.New(cors.Config{AllowOrigins: settings.AllowedOrigins, AllowMethods: "GET,POST,PATCH,OPTIONS", AllowHeaders: "Origin,Content-Type,Accept,Authorization"}))
	route.Register(app, pool, store, jwt, auth, laundry)
	app.Use(func(c *fiber.Ctx) error { return helper.Error(404, "NOT_FOUND", "Endpoint tidak ditemukan") })
	return app, nil
}
