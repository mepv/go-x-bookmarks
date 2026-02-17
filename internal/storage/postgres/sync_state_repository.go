package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type SyncStateRepository struct {
	db *sql.DB
}

func NewSyncStateRepository(db *sql.DB) *SyncStateRepository {
	return &SyncStateRepository{db: db}
}

func (r *SyncStateRepository) GetByUserID(ctx context.Context, userID int64) (*UserSyncState, error) {
	const q = `
		SELECT user_id, last_bookmark_cursor, last_seen_tweet_id, last_sync_at,
		       last_success_at, last_error, updated_at
		FROM user_sync_state
		WHERE user_id = $1
	`

	var rec UserSyncState
	err := r.db.QueryRowContext(ctx, q, userID).Scan(
		&rec.UserID,
		&rec.LastBookmarkCursor,
		&rec.LastSeenTweetID,
		&rec.LastSyncAt,
		&rec.LastSuccessfulSync,
		&rec.LastError,
		&rec.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get sync state: %w", err)
	}

	return &rec, nil
}

func (r *SyncStateRepository) Upsert(ctx context.Context, state *UserSyncState) (*UserSyncState, error) {
	const q = `
		INSERT INTO user_sync_state (
			user_id, last_bookmark_cursor, last_seen_tweet_id, last_sync_at,
			last_success_at, last_error
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			last_bookmark_cursor = EXCLUDED.last_bookmark_cursor,
			last_seen_tweet_id = EXCLUDED.last_seen_tweet_id,
			last_sync_at = EXCLUDED.last_sync_at,
			last_success_at = EXCLUDED.last_success_at,
			last_error = EXCLUDED.last_error,
			updated_at = NOW()
		RETURNING user_id, last_bookmark_cursor, last_seen_tweet_id, last_sync_at,
		          last_success_at, last_error, updated_at
	`

	out := UserSyncState{}
	err := r.db.QueryRowContext(
		ctx,
		q,
		state.UserID,
		state.LastBookmarkCursor,
		state.LastSeenTweetID,
		state.LastSyncAt,
		state.LastSuccessfulSync,
		state.LastError,
	).Scan(
		&out.UserID,
		&out.LastBookmarkCursor,
		&out.LastSeenTweetID,
		&out.LastSyncAt,
		&out.LastSuccessfulSync,
		&out.LastError,
		&out.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert sync state: %w", err)
	}

	return &out, nil
}

func (r *SyncStateRepository) MarkSyncAttempt(ctx context.Context, userID int64, now time.Time, errMsg *string) error {
	const q = `
		INSERT INTO user_sync_state (user_id, last_sync_at, last_error)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET
			last_sync_at = EXCLUDED.last_sync_at,
			last_error = EXCLUDED.last_error,
			updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, q, userID, now, errMsg)
	if err != nil {
		return fmt.Errorf("failed to mark sync attempt: %w", err)
	}
	return nil
}
