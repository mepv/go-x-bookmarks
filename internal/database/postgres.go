package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

const defaultConnectTimeout = 10 * time.Second

func ConnectPostgres(ctx context.Context, driverName, databaseURL string) (*sql.DB, error) {
	connectCtx, cancel := context.WithTimeout(ctx, defaultConnectTimeout)
	defer cancel()

	db, err := sql.Open(driverName, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open sql driver %q: %w", driverName, err)
	}

	if pingErr := db.PingContext(connectCtx); pingErr != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to ping postgres: %v (and failed to close db: %w)", pingErr, closeErr)
		}
		return nil, fmt.Errorf("failed to ping postgres: %w", pingErr)
	}

	return db, nil
}
