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

type CreateWatchRequest struct {
	URL            string `json:"url"`
	ExpectedStatus int    `json:"expected_status"`
	IntervalSec    int    `json:"interval_seconds"`
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
	default:
		{
			//resp.Header().Set("Allow", "POST")
			resp.WriteHeader(http.StatusMethodNotAllowed)
			resp.Write([]byte(`{"error":"method not allowed"}`))
			return
		}
	}
}
