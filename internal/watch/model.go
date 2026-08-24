package watch

import "time"

type CreateParams struct {
	URL            string
	ExpectedStatus int
	IntervalSec    int
}

type UpdateParams struct {
	ExpectedStatus *int
	IntervalSec    *int
	Enabled        *bool
}

type Watch struct {
	ID             int64     `json:"id"`
	URL            string    `json:"url"`
	ExpectedStatus int       `json:"expected_status"`
	IntervalSec    int       `json:"interval_seconds"`
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Check struct {
	ID         int64     `json:"id"`
	WatchID    int64     `json:"watch_id"`
	StatusCode *int      `json:"status_code"`
	DurationMS int64     `json:"duration_ms"`
	Error      *string   `json:"error"`
	Healthy    bool      `json:"healthy"`
	CheckedAt  time.Time `json:"checked_at"`
}
