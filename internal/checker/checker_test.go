package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/requiem-glitch/personal-watcher/internal/watch"
)

func TestCheckerCheckHealthy(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	c := Checker{
		Client: server.Client(),
	}

	w := watch.Watch{
		ID:             123,
		URL:            server.URL,
		ExpectedStatus: http.StatusOK,
	}

	result := c.Check(context.Background(), w)
	if result.Err != nil {
		t.Fatalf("expected nil error, got %v", result.Err)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusOK,
			result.StatusCode,
		)
	}
	if !result.Healthy {
		t.Errorf("expected healthy=true")
	}
	if result.WatchID != w.ID {
		t.Errorf("expected watch id %d, got %d", w.ID, result.WatchID)
	}
}

func TestCheckerCheckUnhealthy(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)
	defer server.Close()

	c := Checker{
		Client: server.Client(),
	}

	w := watch.Watch{
		ID:             123,
		URL:            server.URL,
		ExpectedStatus: http.StatusOK,
	}

	result := c.Check(context.Background(), w)
	if result.Err != nil {
		t.Fatalf("expected nil error, got %v", result.Err)
	}
	if result.StatusCode != http.StatusInternalServerError {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			result.StatusCode,
		)
	}
	if result.Healthy {
		t.Errorf("expected healthy=false")
	}
	if result.WatchID != w.ID {
		t.Errorf("expected watch id %d, got %d", w.ID, result.WatchID)
	}
}

func TestCheckerCheckRequestError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	url := server.URL

	c := Checker{
		Client: server.Client(),
	}

	server.Close()

	w := watch.Watch{
		ID:             123,
		URL:            url,
		ExpectedStatus: http.StatusOK,
	}

	result := c.Check(context.Background(), w)
	if result.Err == nil {
		t.Fatalf("expected error, got nil")
	}
	if result.StatusCode != 0 {
		t.Errorf(
			"expected status %d, got %d",
			0,
			result.StatusCode,
		)
	}
	if result.Healthy {
		t.Errorf("expected healthy=false")
	}
	if result.WatchID != w.ID {
		t.Errorf("expected watch id %d, got %d", w.ID, result.WatchID)
	}

}
