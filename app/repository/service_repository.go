package repository

import (
	"context"

	"laundry-api/app/model"

	"github.com/jackc/pgx/v5"
)

const serviceFields = `id,name,price_per_kg,is_active,created_at`

func scanService(row pgx.Row) (model.LaundryService, error) {
	var s model.LaundryService
	err := row.Scan(&s.ID, &s.Name, &s.PricePerKg, &s.IsActive, &s.CreatedAt)
	return s, databaseError(err)
}
func (r *Postgres) ListServices(ctx context.Context) ([]model.LaundryService, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+serviceFields+` FROM services WHERE is_active=TRUE ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.LaundryService{}
	for rows.Next() {
		s, err := scanService(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
func (r *Postgres) CreateService(ctx context.Context, req model.CreateServiceRequest) (model.LaundryService, error) {
	return scanService(r.pool.QueryRow(ctx, `INSERT INTO services(name,price_per_kg) VALUES($1,$2) RETURNING `+serviceFields, req.Name, req.PricePerKg))
}
func (r *Postgres) PatchService(ctx context.Context, id int64, req model.PatchServiceRequest) (model.LaundryService, error) {
	return scanService(r.pool.QueryRow(ctx, `UPDATE services SET name=COALESCE($2,name),
 price_per_kg=COALESCE($3,price_per_kg),is_active=COALESCE($4,is_active) WHERE id=$1 RETURNING `+serviceFields,
		id, req.Name, req.PricePerKg, req.IsActive))
}
