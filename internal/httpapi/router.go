package httpapi

import (
	"net/http"

	"github.com/requiem-glitch/personal-watcher/internal/postgres"
)

type Handler struct {
	Repo postgres.Repository
}

func NewMux(h Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/watches", h.watchesHandler)
	mux.HandleFunc("/watches/{id}", h.watchesHandler)
	return mux
}
