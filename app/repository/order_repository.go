package repository

import (
	"context"

	"laundry-api/app/model"

	"github.com/jackc/pgx/v5"
)

const orderFields = `id,customer_id,service_id,weight_kg::text,price_per_kg_snapshot,total_price,status,payment_status,notes,created_at,updated_at`

func scanOrder(row pgx.Row) (model.Order, error) {
	var o model.Order
	err := row.Scan(&o.ID, &o.CustomerID, &o.ServiceID, &o.WeightKg, &o.PricePerKgSnapshot, &o.TotalPrice,
		&o.Status, &o.PaymentStatus, &o.Notes, &o.CreatedAt, &o.UpdatedAt)
	return o, databaseError(err)
}
func (r *Postgres) CreateOrder(ctx context.Context, userID int64, req model.CreateOrderRequest) (model.Order, error) {
	return scanOrder(r.pool.QueryRow(ctx, `INSERT INTO orders(customer_id,service_id,notes)
 SELECT $1,id,$3 FROM services WHERE id=$2 AND is_active=TRUE RETURNING `+orderFields, userID, req.ServiceID, req.Notes))
}
func (r *Postgres) GetOrder(ctx context.Context, id int64) (model.Order, error) {
	return scanOrder(r.pool.QueryRow(ctx, `SELECT `+orderFields+` FROM orders WHERE id=$1`, id))
}
func (r *Postgres) ListOrders(ctx context.Context, f model.OrderFilter) ([]model.Order, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+orderFields+` FROM orders
 WHERE ($1::bigint IS NULL OR customer_id=$1) AND ($2::text='' OR status=$2)
 ORDER BY created_at DESC,id DESC LIMIT $3 OFFSET $4`, f.CustomerID, f.Status, f.Limit, f.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
func (r *Postgres) UpdateOrder(ctx context.Context, id int64, decide func(model.Order, int64) (model.OrderChange, error)) (model.Order, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Order{}, err
	}
	defer tx.Rollback(ctx)
	// Baca, validasi aturan bisnis, dan tulis dilakukan dalam transaksi yang sama.
	current, err := scanOrder(tx.QueryRow(ctx, `SELECT `+orderFields+` FROM orders WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return model.Order{}, err
	}
	var rate int64
	if current.PricePerKgSnapshot != nil {
		rate = *current.PricePerKgSnapshot
	} else {
		err = tx.QueryRow(ctx, `SELECT price_per_kg FROM services WHERE id=$1`, current.ServiceID).Scan(&rate)
		if err != nil {
			return model.Order{}, databaseError(err)
		}
	}
	change, err := decide(current, rate)
	if err != nil {
		return model.Order{}, err
	}
	updated, err := scanOrder(tx.QueryRow(ctx, `UPDATE orders SET weight_kg=$2::numeric,
 price_per_kg_snapshot=$3,total_price=$4,status=$5,payment_status=$6,updated_at=NOW()
 WHERE id=$1 RETURNING `+orderFields, id, change.WeightKg, change.PricePerKgSnapshot, change.TotalPrice, change.Status, change.PaymentStatus))
	if err != nil {
		return model.Order{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return model.Order{}, err
	}
	return updated, nil
}
