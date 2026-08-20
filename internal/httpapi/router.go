package httpapi

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	Pool *pgxpool.Pool
}

func NewMux(h Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/watches", h.watchesHandler)
	mux.HandleFunc("/watches/{id}", h.watchesHandler)
	return mux
}
