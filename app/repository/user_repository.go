package repository

import (
	"context"

	"laundry-api/app/model"

	"github.com/jackc/pgx/v5"
)

const userSelect = `SELECT u.id,u.name,u.email,u.password_hash,r.name,u.created_at,
 COALESCE((SELECT array_agg(p.name) FROM role_permissions rp
 JOIN permissions p ON p.id=rp.permission_id WHERE rp.role_id=r.id), ARRAY[]::text[])
 FROM users u JOIN roles r ON r.id=u.role_id `

func scanUser(row pgx.Row) (model.User, error) {
	var u model.User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.Permissions)
	return u, databaseError(err)
}
func (r *Postgres) UserByEmail(ctx context.Context, email string) (model.User, error) {
	return scanUser(r.pool.QueryRow(ctx, userSelect+`WHERE LOWER(u.email)=LOWER($1)`, email))
}
func (r *Postgres) UserByID(ctx context.Context, id int64) (model.User, error) {
	return scanUser(r.pool.QueryRow(ctx, userSelect+`WHERE u.id=$1`, id))
}
func (r *Postgres) CreateUser(ctx context.Context, name, email, hash, role string) (model.User, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `INSERT INTO users(name,email,password_hash,role_id)
 VALUES($1,$2,$3,(SELECT id FROM roles WHERE name=$4)) RETURNING id`, name, email, hash, role).Scan(&id)
	if err != nil {
		return model.User{}, databaseError(err)
	}
	return r.UserByID(ctx, id)
}
