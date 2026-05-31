package postgresql

import (
	"context"
	"errors"
	"fmt"
	"sso/internal/domain/models"
	"sso/internal/storage"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

func (repo *PGRepo) SaveUser(ctx context.Context, email string, passHash []byte) (int64, error) {
	const op = "storage.postgresql.SaveUser"
	var id int64

	err := repo.pool.QueryRow(
		ctx,
		`INSERT INTO users(email, pass_hash) VALUES($1, $2) RETURNING id;`,
		email,
		passHash,
	).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (repo *PGRepo) User(ctx context.Context, email string) (models.User, error) {
	const op = "storage.postgresql.User"
	var user models.User

	err := repo.pool.QueryRow(
		ctx,
		"SELECT id, email, pass_hash FROM users WHERE email = $1",
		email,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PassHash,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}
