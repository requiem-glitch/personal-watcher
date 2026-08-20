package httpapi

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	"net/http"

	"github.com/jackc/pgx/v5"
)

type CreateWatchRequest struct {
	URL            string `json:"url"`
	ExpectedStatus int    `json:"expected_status"`
	IntervalSec    int    `json:"interval_seconds"`
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

func (h Handler) createWatchHandler(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "application/json")

	switch req.Method {
	case http.MethodPost:
		{
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
			encoder := json.NewEncoder(resp)
			err = encoder.Encode(currentReq)
			if err != nil {
				log.Printf("encode response: %v", err)
				resp.WriteHeader(http.StatusInternalServerError) //useless
				return
			}

		}
	case http.MethodGet:
		{
			/*/watches*/
			if req.URL.Path == "/watches" {
				getting, err := h.Pool.Query(
					req.Context(),
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
					resp.WriteHeader(http.StatusInternalServerError)
					return
				}
				defer getting.Close()
				rows := []Watch{}

				encoder := json.NewEncoder(resp)
				for getting.Next() {
					var currentRow Watch
					err = getting.Scan( // err???
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
						resp.WriteHeader(http.StatusInternalServerError)
						return
					}
					rows = append(rows, currentRow)

				}
				if getting.Err() != nil {
					log.Printf("iterate rows: %v", getting.Err())
					resp.WriteHeader(http.StatusInternalServerError)
					return
				}
				resp.WriteHeader(http.StatusOK)
				err = encoder.Encode(rows) // set statusOk auto
				if err != nil {
					log.Printf("encode rows: %v", err)
					resp.WriteHeader(http.StatusInternalServerError) //useless
					return
				}
				return
			} else {
				/*/watches/{id}*/
				rowId, err := strconv.ParseInt(req.PathValue("id"), 10, 64) //cause ID is BIGINT
				if err != nil {
					log.Printf("convert ascii to int: %v", err)
					resp.WriteHeader(http.StatusBadRequest)
					resp.Write([]byte(`{"error":"ID must be numeric"}`))
					return
				}
				if rowId <= 0 {
					log.Printf("convert ascii to row: row ID less than 1")
					resp.WriteHeader(http.StatusBadRequest)
					resp.Write([]byte(`{"error":"ID must be greater than 0"}`))
					return
				}
				getting := h.Pool.QueryRow(
					req.Context(),
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
					rowId,
				)
				var currentRow Watch
				err = getting.Scan( // err???
					&currentRow.ID,
					&currentRow.URL,
					&currentRow.ExpectedStatus,
					&currentRow.IntervalSec,
					&currentRow.Enabled,
					&currentRow.CreatedAt,
					&currentRow.UpdatedAt,
				)
				if err == pgx.ErrNoRows {
					log.Printf("scan row: %v", err)
					resp.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					log.Printf("scan row: %v", err)
					resp.WriteHeader(http.StatusInternalServerError)
					return
				}
				resp.WriteHeader(http.StatusOK)
				encoder := json.NewEncoder(resp)
				err = encoder.Encode(currentRow)
				if err != nil {
					log.Printf("encode row: %v", err)
					resp.WriteHeader(http.StatusInternalServerError) //useless
					return
				}
				return
			}

		}
	case http.MethodDelete:
		{

		}
	default:
		{
			//resp.Header().Set("Allow", "POST")
			resp.WriteHeader(http.StatusMethodNotAllowed)
			resp.Write([]byte(`{"error":"method not allowed"}`))
			return
		}
	}
}
