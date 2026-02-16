package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*UserRecord, error) {
	const q = `
		SELECT id, x_user_id, username, display_name, notion_workspace_id,
		       metadata_fetched_at, metadata_expires_at, last_successful_sync_at,
		       created_at, updated_at
		FROM users
		WHERE username = $1
	`

	var rec UserRecord
	err := r.db.QueryRowContext(ctx, q, username).Scan(
		&rec.ID,
		&rec.XUserID,
		&rec.Username,
		&rec.DisplayName,
		&rec.NotionWorkspaceID,
		&rec.MetadataFetchedAt,
		&rec.MetadataExpiresAt,
		&rec.LastSuccessfulSync,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return &rec, nil
}

func (r *UserRepository) Upsert(ctx context.Context, rec *UserRecord) (*UserRecord, error) {
	const q = `
		INSERT INTO users (
			x_user_id, username, display_name, notion_workspace_id,
			metadata_fetched_at, metadata_expires_at, last_successful_sync_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (x_user_id) DO UPDATE SET
			username = EXCLUDED.username,
			display_name = EXCLUDED.display_name,
			notion_workspace_id = EXCLUDED.notion_workspace_id,
			metadata_fetched_at = EXCLUDED.metadata_fetched_at,
			metadata_expires_at = EXCLUDED.metadata_expires_at,
			last_successful_sync_at = EXCLUDED.last_successful_sync_at,
			updated_at = NOW()
		RETURNING id, x_user_id, username, display_name, notion_workspace_id,
		          metadata_fetched_at, metadata_expires_at, last_successful_sync_at,
		          created_at, updated_at
	`

	out := UserRecord{}
	err := r.db.QueryRowContext(
		ctx,
		q,
		rec.XUserID,
		rec.Username,
		rec.DisplayName,
		rec.NotionWorkspaceID,
		rec.MetadataFetchedAt,
		rec.MetadataExpiresAt,
		rec.LastSuccessfulSync,
	).Scan(
		&out.ID,
		&out.XUserID,
		&out.Username,
		&out.DisplayName,
		&out.NotionWorkspaceID,
		&out.MetadataFetchedAt,
		&out.MetadataExpiresAt,
		&out.LastSuccessfulSync,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert user: %w", err)
	}

	return &out, nil
}

func (r *UserRepository) MarkSuccessfulSync(ctx context.Context, userID int64, syncedAt time.Time) error {
	const q = `
		UPDATE users
		SET last_successful_sync_at = $2, updated_at = NOW()
		WHERE id = $1
	`

	res, err := r.db.ExecContext(ctx, q, userID, syncedAt)
	if err != nil {
		return fmt.Errorf("failed to mark successful user sync: %w", err)
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}
	if affectedRows == 0 {
		return ErrNotFound
	}

	return nil
}
