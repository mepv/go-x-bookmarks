package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type APIUsageRepository struct {
	db *sql.DB
}

func NewAPIUsageRepository(db *sql.DB) *APIUsageRepository {
	return &APIUsageRepository{db: db}
}

func (r *APIUsageRepository) GetByUserAndDay(ctx context.Context, userID int64, day time.Time) (*APIUsageRecord, error) {
	const q = `
		SELECT id, user_id, usage_day, x_calls_count, x_estimated_cost_units,
		       notion_writes_count, created_at, updated_at
		FROM api_usage
		WHERE user_id = $1 AND usage_day = $2
	`

	var rec APIUsageRecord
	err := r.db.QueryRowContext(ctx, q, userID, day.Format("2006-01-02")).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.UsageDay,
		&rec.XCallsCount,
		&rec.XEstimatedCostUnit,
		&rec.NotionWritesCount,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get usage record: %w", err)
	}

	return &rec, nil
}

func (r *APIUsageRepository) Upsert(ctx context.Context, rec *APIUsageRecord) (*APIUsageRecord, error) {
	const q = `
		INSERT INTO api_usage (
			user_id, usage_day, x_calls_count, x_estimated_cost_units, notion_writes_count
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, usage_day) DO UPDATE SET
			x_calls_count = EXCLUDED.x_calls_count,
			x_estimated_cost_units = EXCLUDED.x_estimated_cost_units,
			notion_writes_count = EXCLUDED.notion_writes_count,
			updated_at = NOW()
		RETURNING id, user_id, usage_day, x_calls_count, x_estimated_cost_units,
		          notion_writes_count, created_at, updated_at
	`

	out := APIUsageRecord{}
	err := r.db.QueryRowContext(
		ctx,
		q,
		rec.UserID,
		rec.UsageDay.Format("2006-01-02"),
		rec.XCallsCount,
		rec.XEstimatedCostUnit,
		rec.NotionWritesCount,
	).Scan(
		&out.ID,
		&out.UserID,
		&out.UsageDay,
		&out.XCallsCount,
		&out.XEstimatedCostUnit,
		&out.NotionWritesCount,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert usage record: %w", err)
	}

	return &out, nil
}

func (r *APIUsageRepository) IncrementUsage(ctx context.Context, userID int64, day time.Time, xCalls int32, xCost float64, notionWrites int32) (*APIUsageRecord, error) {
	const q = `
		INSERT INTO api_usage (
			user_id, usage_day, x_calls_count, x_estimated_cost_units, notion_writes_count
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, usage_day) DO UPDATE SET
			x_calls_count = api_usage.x_calls_count + EXCLUDED.x_calls_count,
			x_estimated_cost_units = api_usage.x_estimated_cost_units + EXCLUDED.x_estimated_cost_units,
			notion_writes_count = api_usage.notion_writes_count + EXCLUDED.notion_writes_count,
			updated_at = NOW()
		RETURNING id, user_id, usage_day, x_calls_count, x_estimated_cost_units,
		          notion_writes_count, created_at, updated_at
	`

	out := APIUsageRecord{}
	err := r.db.QueryRowContext(
		ctx,
		q,
		userID,
		day.Format("2006-01-02"),
		xCalls,
		xCost,
		notionWrites,
	).Scan(
		&out.ID,
		&out.UserID,
		&out.UsageDay,
		&out.XCallsCount,
		&out.XEstimatedCostUnit,
		&out.NotionWritesCount,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to increment usage: %w", err)
	}

	return &out, nil
}
