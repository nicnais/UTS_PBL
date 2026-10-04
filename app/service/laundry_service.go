package service

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"laundry-api/app/model"
	"laundry-api/app/repository"
	"laundry-api/helper"
)

type LaundryService struct{ store repository.Store }

func NewLaundry(store repository.Store) *LaundryService { return &LaundryService{store: store} }
func (s *LaundryService) ListServices(c *fiber.Ctx) error {
	ctx, cancel := helper.DBContext(c)
	defer cancel()
	data, err := s.store.ListServices(ctx)
	if err != nil {
		return err
	}
	return helper.OK(c, 200, "Daftar layanan aktif", data)
}
func validateService(name *string, price *int64) error {
	fields := map[string]string{}
	if name != nil {
		*name = strings.TrimSpace(*name)
		n := utf8.RuneCountInString(*name)
		if n < 3 || n > 100 {
			fields["name"] = "Panjang nama 3 sampai 100 karakter"
		}
	}
	if price != nil && (*price < 1 || *price > 1000000000) {
		fields["price_per_kg"] = "Harga harus 1 sampai 1000000000 rupiah"
	}
	if len(fields) > 0 {
		return helper.Validation(fields)
	}
	return nil
}
func (s *LaundryService) CreateService(c *fiber.Ctx) error {
	var req model.CreateServiceRequest
	if err := helper.Decode(c, &req); err != nil {
		return err
	}
	if err := validateService(&req.Name, &req.PricePerKg); err != nil {
		return err
	}
	ctx, cancel := helper.DBContext(c)
	defer cancel()
	data, err := s.store.CreateService(ctx, req)
	if err != nil {
		return err
	}
	// Tidak memberi Location ke route detail layanan yang belum disediakan.
	return helper.OK(c, 201, "Layanan dibuat", data)
}
func (s *LaundryService) PatchService(c *fiber.Ctx) error {
	id, err := helper.ID(c)
	if err != nil {
		return err
	}
	var req model.PatchServiceRequest
	if err := helper.Decode(c, &req); err != nil {
		return err
	}
	if req.Name == nil && req.PricePerKg == nil && req.IsActive == nil {
		return helper.Validation(map[string]string{"body": "Minimal satu field harus dikirim"})
	}
	if err := validateService(req.Name, req.PricePerKg); err != nil {
		return err
	}
	ctx, cancel := helper.DBContext(c)
	defer cancel()
	data, err := s.store.PatchService(ctx, id, req)
	if err != nil {
		return err
	}
	return helper.OK(c, 200, "Layanan diperbarui", data)
}
func (s *LaundryService) CreateOrder(c *fiber.Ctx) error {
	var req model.CreateOrderRequest
	if err := helper.Decode(c, &req); err != nil {
		return err
	}
	if req.ServiceID < 1 {
		return helper.Validation(map[string]string{"service_id": "Harus berupa ID positif"})
	}
	if utf8.RuneCountInString(req.Notes) > 2000 {
		return helper.Validation(map[string]string{"notes": "Maksimal 2000 karakter"})
	}
	ctx, cancel := helper.DBContext(c)
	defer cancel()
	data, err := s.store.CreateOrder(ctx, helper.CurrentUser(c).ID, req)
	if err != nil {
		return err
	}
	c.Set("Location", "/api/v1/orders/"+strconv.FormatInt(data.ID, 10))
	return helper.OK(c, 201, "Pesanan dibuat", data)
}
func (s *LaundryService) GetOrder(c *fiber.Ctx) error {
	id, err := helper.ID(c)
	if err != nil {
		return err
	}
	ctx, cancel := helper.DBContext(c)
	defer cancel()
	data, err := s.store.GetOrder(ctx, id)
	if err != nil {
		return err
	}
	if !CanReadOrder(helper.CurrentUser(c), data) {
		return helper.Error(403, "FORBIDDEN", "Pesanan bukan milik Anda")
	}
	return helper.OK(c, 200, "Detail pesanan", data)
}
func (s *LaundryService) ListOrders(c *fiber.Ctx) error {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 || page > 100000 {
		return helper.Validation(map[string]string{"page": "Page harus 1 sampai 100000"})
	}
	limit, err := strconv.Atoi(c.Query("limit", "20"))
	if err != nil || limit < 1 || limit > 100 {
		return helper.Validation(map[string]string{"limit": "Limit harus 1 sampai 100"})
	}
	status := c.Query("status")
	if status != "" && !ValidStatus(status) {
		return helper.Validation(map[string]string{"status": "Status tidak dikenal"})
	}
	f := model.OrderFilter{Status: status, Limit: limit, Offset: (page - 1) * limit}
	user := helper.CurrentUser(c)
	if !user.Can("order:read:any") {
		f.CustomerID = &user.ID
	}
	ctx, cancel := helper.DBContext(c)
	defer cancel()
	data, err := s.store.ListOrders(ctx, f)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"success": true, "message": "Daftar pesanan", "data": data, "meta": fiber.Map{"page": page, "limit": limit, "count": len(data)}})
}
func (s *LaundryService) PatchOrder(c *fiber.Ctx) error {
	id, err := helper.ID(c)
	if err != nil {
		return err
	}
	var req model.PatchOrderRequest
	if err := helper.Decode(c, &req); err != nil {
		return err
	}
	ctx, cancel := helper.DBContext(c)
	defer cancel()
	data, err := s.store.UpdateOrder(ctx, id, func(o model.Order, rate int64) (model.OrderChange, error) { return DecideOrderChange(o, rate, req) })
	if err != nil {
		return err
	}
	return helper.OK(c, 200, "Pesanan diperbarui", data)
}
