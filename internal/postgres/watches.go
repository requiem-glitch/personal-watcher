package postgres

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/requiem-glitch/personal-watcher/internal/checker"
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

func (r Repository) SaveCheck(ctx context.Context, result checker.Result) error {
	var statusCode, errText any
	if result.StatusCode != 0 {
		statusCode = result.StatusCode
	} else {
		statusCode = nil
	}
	if result.Err != nil {
		errText = result.Err.Error()
	} else {
		errText = nil
	}
	_, err := r.Pool.Exec(
		ctx,
		`INSERT INTO checks (
			watch_id,
			status_code,
			duration_ms,
			error,
			healthy
		)
		VALUES ($1, $2, $3, $4, $5);`,
		result.WatchID,
		statusCode,
		result.Duration.Milliseconds(),
		errText,
		result.Healthy,
	)
	return err
}

func (r Repository) ListDueWatches(ctx context.Context) ([]watch.Watch, error) {
	rows, err := r.Pool.Query(
		ctx,
		`SELECT 
			watches.id, 
			watches.url, 
			watches.expected_status, 
			watches.interval_seconds, 
			watches.enabled, 
			watches.created_at, 
			watches.updated_at
		 FROM watches
		 LEFT JOIN checks ON watches.id = checks.watch_id
		 WHERE watches.enabled = TRUE
		 GROUP BY watches.id
		 HAVING
		 	MAX(checks.checked_at) IS NULL
			OR
			MAX(checks.checked_at)
				+ watches.interval_seconds * INTERVAL '1 second'
				<= NOW();`,
	)
	if err != nil {
		return []watch.Watch{}, err
	}
	defer rows.Close()
	readyToCheck := []watch.Watch{}
	for rows.Next() {
		var currentRow watch.Watch
		err = rows.Scan(
			&currentRow.ID,
			&currentRow.URL,
			&currentRow.ExpectedStatus,
			&currentRow.IntervalSec,
			&currentRow.Enabled,
			&currentRow.CreatedAt,
			&currentRow.UpdatedAt,
		)
		if err != nil {
			return []watch.Watch{}, err
		}
		readyToCheck = append(readyToCheck, currentRow)
	}
	if rows.Err() != nil {
		return []watch.Watch{}, rows.Err()
	}
	return readyToCheck, nil
}

func (r Repository) GetLastHealth(ctx context.Context, watchID int64) (healthy bool, found bool, err error) {
	row := r.Pool.QueryRow(
		ctx,
		`SELECT healthy
		 FROM checks
		 WHERE watch_id = $1
		 ORDER BY checked_at DESC
		 LIMIT 1;`,
		watchID,
	)
	err = row.Scan(&healthy)
	if err == pgx.ErrNoRows {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return healthy, true, nil
}

func (r Repository) ListChecks(ctx context.Context, watchID int64, limit, offset int) ([]watch.Check, error) {
	rows, err := r.Pool.Query(
		ctx,
		`SELECT
			id,
			watch_id,
			status_code,
			duration_ms,
			error,
			healthy,
			checked_at
		 FROM checks
		 WHERE watch_id = $1
		 ORDER BY checked_at DESC
		 LIMIT $2
		 OFFSET $3;`,
		watchID,
		limit,
		offset,
	)
	if err != nil {
		return []watch.Check{}, err
	}
	defer rows.Close()
	result := []watch.Check{}
	for rows.Next() {
		var row watch.Check
		err = rows.Scan(&row.ID, &row.WatchID, &row.StatusCode, &row.DurationMS, &row.Error, &row.Healthy, &row.CheckedAt)
		if err != nil {
			return []watch.Check{}, err
		}
		result = append(result, row)
	}
	if rows.Err() != nil {
		return []watch.Check{}, rows.Err()
	}
	return result, nil
}

func (r Repository) UpdateWatch(ctx context.Context, id int64, params watch.UpdateParams) (watch.Watch, error) {
	row := r.Pool.QueryRow(
		ctx,
		`UPDATE watches
		 SET
		 	expected_status = COALESCE($2, expected_status),
		 	interval_seconds = COALESCE($3, interval_seconds),
		 	enabled = COALESCE($4, enabled),
		 	updated_at = NOW()
		  WHERE id = $1
		  RETURNING
		  	id,
			url,
			expected_status,
			interval_seconds,
			enabled,
			created_at,
			updated_at;`,
		id,
		params.ExpectedStatus,
		params.IntervalSec,
		params.Enabled,
	)
	var result watch.Watch
	err := row.Scan(
		&result.ID,
		&result.URL,
		&result.ExpectedStatus,
		&result.IntervalSec,
		&result.Enabled,
		&result.CreatedAt,
		&result.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return watch.Watch{}, pgx.ErrNoRows
	}
	if err != nil {
		return watch.Watch{}, err
	}
	return result, nil

}
