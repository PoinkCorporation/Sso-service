package postgresql

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
)

type PGRepo struct {
	pool *pgxpool.Pool
}

func New(conStr string) (*PGRepo, error) {
	pool, err := pgxpool.Connect(context.Background(), conStr)
	if err != nil {
		return nil, err
	}

	return &PGRepo{pool: pool}, nil
}

func (r *PGRepo) Close() {
	r.pool.Close()
}
