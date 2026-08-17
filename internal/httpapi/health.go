package httpapi

import "net/http"

func healthHandler(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "application/json")
	if req.Method == http.MethodGet {
		resp.Write([]byte(`{"status":"ok"}`))
	} else {
		resp.Header().Set("Allow", "GET")
		resp.WriteHeader(http.StatusMethodNotAllowed)
		resp.Write([]byte(`{"error":"method not allowed"}`))
		return
	}

}
