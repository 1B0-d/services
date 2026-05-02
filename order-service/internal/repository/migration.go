package repository

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(ctx context.Context, db *pgxpool.Pool, path string) error {
	sql, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read migration file %s: %w", path, err)
	}

	if _, err := db.Exec(ctx, string(sql)); err != nil {
		return fmt.Errorf("run migration file %s: %w", path, err)
	}

	return nil
}
