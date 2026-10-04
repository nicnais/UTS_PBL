package repository

import (
	"context"
	"errors"
	"time"

	"laundry-api/app/model"

	"github.com/jackc/pgx/v5"
)

func (r *Postgres) SaveToken(ctx context.Context, userID int64, hash string, expires time.Time) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO refresh_tokens(user_id,token_hash,expires_at) VALUES($1,$2,$3)`, userID, hash, expires)
	return databaseError(err)
}
func (r *Postgres) RotateToken(ctx context.Context, oldHash, newHash string, expires time.Time) (model.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.User{}, err
	}
	defer tx.Rollback(ctx)
	var userID int64
	// UPDATE mengambil row lock: dua request bersamaan tidak bisa memakai token yang sama.
	err = tx.QueryRow(ctx, `UPDATE refresh_tokens SET revoked_at=NOW()
 WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>NOW() RETURNING user_id`, oldHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, model.ErrInvalidToken
	}
	if err != nil {
		return model.User{}, err
	}
	user, err := scanUser(tx.QueryRow(ctx, userSelect+`WHERE u.id=$1`, userID))
	if err != nil {
		return user, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO refresh_tokens(user_id,token_hash,expires_at) VALUES($1,$2,$3)`, userID, newHash, expires)
	if err != nil {
		return user, err
	}
	return user, tx.Commit(ctx)
}
func (r *Postgres) RevokeToken(ctx context.Context, hash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=NOW() WHERE token_hash=$1 AND revoked_at IS NULL`, hash)
	return err
}
