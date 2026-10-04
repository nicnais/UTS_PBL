package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"laundry-api/app/model"
)

type Store interface {
	UserByEmail(context.Context, string) (model.User, error)
	UserByID(context.Context, int64) (model.User, error)
	CreateUser(context.Context, string, string, string, string) (model.User, error)
	SaveToken(context.Context, int64, string, time.Time) error
	RotateToken(context.Context, string, string, time.Time) (model.User, error)
	RevokeToken(context.Context, string) error
	ListServices(context.Context) ([]model.LaundryService, error)
	CreateService(context.Context, model.CreateServiceRequest) (model.LaundryService, error)
	PatchService(context.Context, int64, model.PatchServiceRequest) (model.LaundryService, error)
	CreateOrder(context.Context, int64, model.CreateOrderRequest) (model.Order, error)
	GetOrder(context.Context, int64) (model.Order, error)
	ListOrders(context.Context, model.OrderFilter) ([]model.Order, error)
	UpdateOrder(context.Context, int64, func(model.Order, int64) (model.OrderChange, error)) (model.Order, error)
}

type Postgres struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

var _ Store = (*Postgres)(nil)

func databaseError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23505" || pgErr.Code == "23503") {
		return model.ErrConflict
	}
	return err
}
