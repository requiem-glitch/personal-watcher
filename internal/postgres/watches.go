package postgres

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/requiem-glitch/personal-watcher/internal/watch"
)

type Repository struct {
	Pool *pgxpool.Pool
}

func (r Repository) CreateWatch(ctx context.Context, input watch.CreateParams) (watch.Watch, error) {
	var currentReq watch.Watch
	inserting := r.Pool.QueryRow(
		ctx,
		`INSERT INTO watches (
			url,
			expected_status,
			interval_seconds
		)
		VALUES ($1, $2, $3)	
		RETURNING id, enabled, created_at, updated_at;`,
		input.URL,
		input.ExpectedStatus,
		input.IntervalSec,
	)
	currentReq.URL = input.URL
	currentReq.ExpectedStatus = input.ExpectedStatus
	currentReq.IntervalSec = input.IntervalSec
	err := inserting.Scan(&currentReq.ID, &currentReq.Enabled, &currentReq.CreatedAt, &currentReq.UpdatedAt)
	if err != nil {
		log.Printf("insert watch: %v", err)
		return watch.Watch{}, err
	}
	return currentReq, nil
}

func (r Repository) ListWatches(ctx context.Context) ([]watch.Watch, error) {
	getting, err := r.Pool.Query(
		ctx,
		`SELECT
			id,
			url,
			expected_status,
			interval_seconds,
			enabled,
			created_at,
			updated_at
		FROM watches;`,
	)
	if err != nil {
		log.Printf("get request: %v", err)
		return []watch.Watch{}, err
	}
	defer getting.Close()

	rows := []watch.Watch{}

	for getting.Next() {
		var currentRow watch.Watch
		err = getting.Scan(
			&currentRow.ID,
			&currentRow.URL,
			&currentRow.ExpectedStatus,
			&currentRow.IntervalSec,
			&currentRow.Enabled,
			&currentRow.CreatedAt,
			&currentRow.UpdatedAt,
		)
		if err != nil {
			log.Printf("scan row: %v", err)
			return []watch.Watch{}, err
		}
		rows = append(rows, currentRow)

	}
	if getting.Err() != nil {
		log.Printf("iterate rows: %v", getting.Err())
		return []watch.Watch{}, getting.Err()
	}
	return rows, nil
}

func (r Repository) GetWatch(ctx context.Context, id int64) (watch.Watch, error) {
	getting := r.Pool.QueryRow(
		ctx,
		`SELECT
			id,
			url,
			expected_status,
			interval_seconds,
			enabled,
			created_at,
			updated_at
		FROM watches
		WHERE id = $1;`,
		id,
	)
	var currentRow watch.Watch
	err := getting.Scan(
		&currentRow.ID,
		&currentRow.URL,
		&currentRow.ExpectedStatus,
		&currentRow.IntervalSec,
		&currentRow.Enabled,
		&currentRow.CreatedAt,
		&currentRow.UpdatedAt,
	)
	if err != nil {
		return watch.Watch{}, err
	}
	return currentRow, nil
}

func (r Repository) DeleteWatch(ctx context.Context, id int64) (int64, error) {
	deleting, err := r.Pool.Exec(
		ctx,
		`DELETE
		FROM watches
		WHERE id = $1;`,
		id,
	)
	if err != nil {
		log.Printf("delete row: %v", err)
		return 0, err
	}
	return deleting.RowsAffected(), nil
}
