package postgresql

import (
	"context"
	"errors"
	"fmt"
	"sso/internal/domain/models"
	"sso/internal/storage"

	"github.com/jackc/pgx/v4"
)

func (repo *PGRepo) App(ctx context.Context, id int) (models.App, error) {
	const op = "storage.postgresql.App"

	var app models.App

	err := repo.pool.QueryRow(
		ctx,
		"SELECT id, name, secret FROM apps WHERE id = $1",
		id,
	).Scan(
		&app.ID,
		&app.Name,
		&app.Secret,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
		}

		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}
