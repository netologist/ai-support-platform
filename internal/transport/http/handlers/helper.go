package handlers

import (
	"encoding/json"
	"net/http"
)

type problemDetails struct {
	Type      string              `json:"type"`
	Title     string              `json:"title"`
	Status    int                 `json:"status"`
	Detail    string              `json:"detail,omitempty"`
	Errors    map[string][]string `json:"errors,omitempty"`
	RequestID string              `json:"request_id,omitempty"`
}

func writeJSON(writer http.ResponseWriter, statusCode int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

func writeProblem(writer http.ResponseWriter, request *http.Request, statusCode int, title string, detail string, validationErrors map[string][]string) {
	writeJSON(writer, statusCode, problemDetails{
		Type:      "about:blank",
		Title:     title,
		Status:    statusCode,
		Detail:    detail,
		Errors:    validationErrors,
		RequestID: request.Header.Get("X-Request-ID"),
	})
}
