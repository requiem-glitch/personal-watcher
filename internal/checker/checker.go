package checker

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/requiem-glitch/personal-watcher/internal/watch"
)

type Checker struct {
	Client *http.Client
}

type Result struct {
	WatchID    int64
	StatusCode int
	Duration   time.Duration
	Err        error
}

func (c Checker) Check(ctx context.Context, w watch.Watch) Result {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.URL, nil)
	if err != nil {
		log.Printf("NewRequestWithContext: %v", err)
		return Result{WatchID: w.ID, Err: err}
	}
	startTime := time.Now()

	resp, err := c.Client.Do(req)
	duration := time.Since(startTime)
	if err != nil {
		log.Printf("Client.Do: %v", err)
		return Result{WatchID: w.ID, Duration: duration, Err: err}
	}
	defer resp.Body.Close()
	return Result{
		WatchID:    w.ID,
		StatusCode: resp.StatusCode,
		Duration:   duration,
		Err:        nil,
	}
}
