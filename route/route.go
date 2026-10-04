package route

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"laundry-api/app/repository"
	"laundry-api/app/service"
	"laundry-api/helper"
	"laundry-api/middleware"
)

func Register(app *fiber.App, pool *pgxpool.Pool, store repository.Store, jwt *helper.JWTManager, auth *service.AuthService, laundry *service.LaundryService) {
	api := app.Group("/api/v1")
	api.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := helper.DBContext(c)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return helper.Error(503, "DATABASE_UNAVAILABLE", "Database belum tersedia")
		}
		return helper.OK(c, 200, "Laundry API berjalan", fiber.Map{"database": "connected"})
	})
	authentication := api.Group("/auth")
	authentication.Post("/register", middleware.AuthLimiter(10), auth.Register)
	authentication.Post("/login", middleware.AuthLimiter(5), auth.Login)
	authentication.Post("/refresh", middleware.AuthLimiter(30), auth.Refresh)
	authentication.Post("/logout", middleware.AuthLimiter(30), auth.Logout)
	authentication.Get("/me", middleware.RequireAuth(store, jwt), auth.Me)

	secured := api.Group("", middleware.RequireAuth(store, jwt))
	services := secured.Group("/services")
	services.Get("/", laundry.ListServices)
	services.Post("/", middleware.RequirePermission("service:create"), laundry.CreateService)
	services.Patch("/:id", middleware.RequirePermission("service:update"), laundry.PatchService)

	orders := secured.Group("/orders")
	orders.Post("/", middleware.RequirePermission("order:create"), laundry.CreateOrder)
	orders.Get("/", laundry.ListOrders)
	orders.Get("/:id", laundry.GetOrder)
	orders.Patch("/:id", middleware.RequirePermission("order:update:any"), laundry.PatchOrder)
}