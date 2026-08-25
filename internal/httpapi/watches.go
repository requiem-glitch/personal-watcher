package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/requiem-glitch/personal-watcher/internal/watch"
)

func parseWatchID(req *http.Request) (int64, error) {
	num, err := strconv.ParseInt(req.PathValue("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	return num, nil
}

func parseInt(str string) (int, error) {
	num, err := strconv.Atoi(str)
	if err != nil {
		return 0, err
	}
	return num, nil
}

type CreateWatchRequest struct {
	URL            string `json:"url"`
	ExpectedStatus int    `json:"expected_status"`
	IntervalSec    int    `json:"interval_seconds"`
}

type UpdateWatchRequest struct {
	ExpectedStatus *int  `json:"expected_status"`
	IntervalSec    *int  `json:"interval_seconds"`
	Enabled        *bool `json:"enabled"`
}

func (h Handler) watchesHandler(resp http.ResponseWriter, req *http.Request) {
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
			var currentReq watch.Watch
			inputParams := watch.CreateParams{
				URL:            input.URL,
				ExpectedStatus: input.ExpectedStatus,
				IntervalSec:    input.IntervalSec,
			}
			currentReq, err = h.Repo.CreateWatch(req.Context(), inputParams)

			if err != nil {
				log.Printf("CreateWatch: %v", err)
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.Printf("%+v", currentReq)

			resp.WriteHeader(http.StatusCreated)
			encoder := json.NewEncoder(resp)
			err = encoder.Encode(currentReq)
			if err != nil {
				log.Printf("encode response: %v", err)
				return
			}

		}
	case http.MethodGet:
		{
			/*/watches*/
			if req.URL.Path == "/watches" {
				rows, err := h.Repo.ListWatches(req.Context())
				if err != nil {
					log.Printf("ListWatches: %v", err)
					resp.WriteHeader(http.StatusInternalServerError)
					return
				}

				encoder := json.NewEncoder(resp)
				resp.WriteHeader(http.StatusOK)
				err = encoder.Encode(rows) // set statusOk auto
				if err != nil {
					log.Printf("encode rows: %v", err)
					return
				}
				return
			} else {
				/*/watches/{id}*/
				rowId, err := parseWatchID(req)
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

				currentRow, err := h.Repo.GetWatch(req.Context(), rowId)
				if err == pgx.ErrNoRows {
					log.Printf("scan row: %v", err)
					resp.WriteHeader(http.StatusNotFound)
					return
				} else if err != nil {
					log.Printf("scan row: %v", err)
					resp.WriteHeader(http.StatusInternalServerError)
					return
				}
				resp.WriteHeader(http.StatusOK)
				encoder := json.NewEncoder(resp)
				err = encoder.Encode(currentRow)
				if err != nil {
					log.Printf("encode row: %v", err)
					return
				}
				return
			}

		}
	case http.MethodDelete:
		{
			rowId, err := parseWatchID(req)
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

			deleting, err := h.Repo.DeleteWatch(req.Context(), rowId)
			if err != nil {
				log.Printf("DeleteWatch: %v", err)
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}
			if deleting == 0 {
				log.Printf("delete item: item not found")
				resp.WriteHeader(http.StatusNotFound)
				resp.Write([]byte(`{"error":"item not found"}`))
				return
			}
			log.Printf("delete item: 1 item deleted")
			resp.WriteHeader(http.StatusNoContent)
			return
		}
	case http.MethodPatch:
		decoder := json.NewDecoder(req.Body)
		var input UpdateWatchRequest
		err := decoder.Decode(&input)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte(`{"error":"cannot decode json or invalid json data"}`))
			return
		}
		if input.ExpectedStatus != nil && (*input.ExpectedStatus < 100 || *input.ExpectedStatus > 599) {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte(`{"error":"invalid ExpectedStatus"}`))
			return
		}
		if input.IntervalSec != nil && *input.IntervalSec <= 0 {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte(`{"error":"invalid IntervalSec"}`))
			return
		}
		if input.IntervalSec == nil && input.ExpectedStatus == nil && input.Enabled == nil {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte(`{"error":"nothing to update"}`))
			return
		}
		wid, err := parseWatchID(req)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte(`{"error":"watch_id must be numeric"}`))
			return
		}
		if wid <= 0 {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte(`{"error":"watch_id must be greater than 0"}`))
			return
		}
		params := watch.UpdateParams{
			ExpectedStatus: input.ExpectedStatus,
			IntervalSec:    input.IntervalSec,
			Enabled:        input.Enabled,
		}
		result, err := h.Repo.UpdateWatch(req.Context(), wid, params)
		if err == pgx.ErrNoRows {
			resp.WriteHeader(http.StatusNotFound)
			return
		}
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		encoder := json.NewEncoder(resp)
		resp.WriteHeader(http.StatusOK)
		err = encoder.Encode(result)
		if err != nil {
			log.Printf("encode updated watch: %v", err)
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

func (h Handler) watchChecksHandler(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "application/json")
	if req.Method != http.MethodGet {
		resp.Header().Set("Allow", "GET")
		resp.WriteHeader(http.StatusMethodNotAllowed)
		resp.Write([]byte(`{"error":"method not allowed"}`))
		return
	}
	wid, err := parseWatchID(req)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	if wid <= 0 {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(`{"error":"watch_id must be greater than 0"}`))
		return
	}

	values := req.URL.Query()
	limit, offset := 20, 0
	limitS, offsetS := values.Get("limit"), values.Get("offset")
	if limitS != "" {
		limit, err = parseInt(limitS)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte(`{"error":"limit must be numeric"}`))
			return
		}
		if limit <= 0 || limit > 100 {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte(`{"error":"limit must be in interval 1..100"}`))
			return
		}
	}
	if offsetS != "" {
		offset, err = parseInt(offsetS)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte(`{"error":"offset must be numeric"}`))
			return
		}
		if offset < 0 {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte(`{"error":"offset must be greater or equal 0"}`))
			return
		}
	}
	checks, err := h.Repo.ListChecks(req.Context(), wid, limit, offset)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	encoder := json.NewEncoder(resp)
	resp.WriteHeader(http.StatusOK)
	err = encoder.Encode(checks)
	if err != nil {
		log.Printf("encode checks: %v", err)
	}
}
