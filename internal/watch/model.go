package watch

import "time"

type CreateParams struct {
	URL            string
	ExpectedStatus int
	IntervalSec    int
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
