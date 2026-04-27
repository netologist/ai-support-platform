package handlers

import (
	"encoding/json"
	"log/slog"
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
	if err := json.NewEncoder(writer).Encode(payload); err != nil {
		// JSON encoding only fails for non-serialisable types (chan, func, etc.).
		// Domain structs should never trigger this, but log it if they do.
		slog.Error("writeJSON encode failed", slog.Any("error", err))
	}
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
