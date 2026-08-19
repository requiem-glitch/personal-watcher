package httpapi

import (
	"encoding/json"
	"log"
	"time"

	"net/http"
)

type CreateWatchRequest struct {
	URL            string `json:"url"`
	ExpectedStatus int    `json:"expected_status"`
	IntervalSec    int    `json:"interval_seconds"`
}

type Watch struct {
	ID             int64
	URL            string
	ExpectedStatus int
	IntervalSec    int
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (h Handler) createWatchHandler(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "application/json")
	if req.Method != http.MethodPost {
		resp.Header().Set("Allow", "POST")
		resp.WriteHeader(http.StatusMethodNotAllowed)
		resp.Write([]byte(`{"error":"method not allowed"}`))
		return
	}

	var input CreateWatchRequest
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&input)
	if err != nil || (input.URL == "" || (input.ExpectedStatus < 100 || input.ExpectedStatus > 599) || input.IntervalSec <= 0) {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(`{"error":"cannot decode json or invalid json data"}`))
		return
	}
	//log.Printf("%+v", input)

	/*INSERTING DATA*/
	var currentReq Watch
	currentReq.URL = input.URL
	currentReq.ExpectedStatus = input.ExpectedStatus
	currentReq.IntervalSec = input.IntervalSec
	inserting := h.Pool.QueryRow(
		req.Context(),
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
	err = inserting.Scan(&currentReq.ID, &currentReq.Enabled, &currentReq.CreatedAt, &currentReq.UpdatedAt)
	if err != nil {
		log.Printf("insert watch: %v", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("%+v", currentReq)
	resp.WriteHeader(http.StatusCreated)
}
