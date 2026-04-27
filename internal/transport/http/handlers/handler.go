package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/netologist/ai-support-platform/internal/app/command"
	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/app/executor"
	"github.com/netologist/ai-support-platform/internal/app/query"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/transport"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var _ ServerInterface = (*Handlers)(nil)

func New(
	loginExecutor executor.LoginExecutor,
	createTicketExecutor executor.CreateTicketExecutor,
	updateTicketExecutor executor.UpdateTicketExecutor,
	listTicketsExecutor executor.ListTicketsExecutor,
	getTicketExecutor executor.GetTicketExecutor,
	ingestDocumentService executor.IngestDocumentExecutor,
	listDocumentsService executor.ListDocumentsExecutor,
	publicRateLimit int64,
	authenticatedRateLimit int64,
	rateLimitWindow time.Duration) *Handlers {
	return &Handlers{
		loginExecutor:          loginExecutor,
		createTicketExecutor:   createTicketExecutor,
		updateTicketExecutor:   updateTicketExecutor,
		getTicketExecutor:      getTicketExecutor,
		listTicketsExecutor:    listTicketsExecutor,
		publicRateLimit:        publicRateLimit,
		authenticatedRateLimit: authenticatedRateLimit,
		rateLimitWindow:        rateLimitWindow,
		ingestDocumentService:  ingestDocumentService,
		listDocumentsExecutor:  listDocumentsService,
	}
}

type Handlers struct {
	loginExecutor          executor.LoginExecutor
	createTicketExecutor   executor.CreateTicketExecutor
	updateTicketExecutor   executor.UpdateTicketExecutor
	getTicketExecutor      executor.GetTicketExecutor
	listTicketsExecutor    executor.ListTicketsExecutor
	ingestDocumentService  executor.IngestDocumentExecutor
	listDocumentsExecutor  executor.ListDocumentsExecutor
	publicRateLimit        int64
	authenticatedRateLimit int64
	rateLimitWindow        time.Duration
}

func (h *Handlers) GetHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var body LoginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Invalid request", fmt.Sprintf("decode request body: %v", err), nil)
		return
	}

	result, err := h.loginExecutor.Execute(r.Context(), command.LoginCommand{
		Email:    strings.ToLower(string(body.Email)),
		Password: body.Password,
		TenantID: body.TenantId,
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrInvalidCredentials):
			writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "email or password is invalid", nil)
		default:
			writeProblem(w, r, http.StatusInternalServerError, "Internal Server Error", "unexpected error", nil)
		}

		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		UserId:      result.Principal.UserID,
		Role:        result.Principal.Role,
		TenantId:    result.Principal.TenantID,
	})
}

func (h *Handlers) CreateTicket(w http.ResponseWriter, r *http.Request) {
	principal, ok := transport.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "missing authenticated principal", nil)
		return
	}

	var body CreateTicketRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Invalid request", fmt.Sprintf("decode request body: %v", err), nil)
		return
	}

	ticket, err := h.createTicketExecutor.Execute(r.Context(), command.CreateTicketCommand{
		Principal: principal,
		Subject:   body.Subject,
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrForbidden):
			writeProblem(w, r, http.StatusForbidden, "Forbidden", "you are not allowed to create tickets", nil)
		default:
			writeProblem(w, r, http.StatusInternalServerError, "Internal Server Error", "unexpected error", nil)
		}
		return
	}

	writeJSON(w, http.StatusCreated, toTicketResponse(ticket))
}

func (h *Handlers) UpdateTicket(w http.ResponseWriter, r *http.Request, ticketID openapi_types.UUID) {
	principal, ok := transport.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "missing authenticated principal", nil)
		return
	}

	var body UpdateTicketRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Invalid request", fmt.Sprintf("decode request body: %v", err), nil)
		return
	}

	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Validation failed", "ticketID must be a valid UUID", map[string][]string{"ticketid": {"must be a valid UUID"}})
		return
	}

	if body.Subject != nil {
		trimmed := strings.TrimSpace(*body.Subject)
		body.Subject = &trimmed
	}

	var bodyStatus *string
	if body.Status != nil {
		normalized := strings.ToLower(strings.TrimSpace(string(*body.Status)))
		bodyStatus = &normalized
	}

	ticket, err := h.updateTicketExecutor.Execute(r.Context(), command.UpdateTicketCommand{
		TicketID:  ticketID,
		Principal: principal,
		Subject:   body.Subject,
		Status:    bodyStatus,
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrInvalidArgument):
			writeProblem(w, r, http.StatusBadRequest, "Validation failed", "request body is invalid", nil)
		case errors.Is(err, apperrors.ErrForbidden):
			writeProblem(w, r, http.StatusForbidden, "Forbidden", "you are not allowed to update this ticket", nil)
		case errors.Is(err, apperrors.ErrNotFound):
			writeProblem(w, r, http.StatusNotFound, "Not Found", "ticket was not found", nil)
		default:
			writeProblem(w, r, http.StatusInternalServerError, "Internal Server Error", "unexpected error", nil)
		}
		return
	}

	writeJSON(w, http.StatusOK, toTicketResponse(ticket))
}

func (h *Handlers) GetTicket(w http.ResponseWriter, r *http.Request, ticketID openapi_types.UUID) {
	principal, ok := transport.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "missing authenticated principal", nil)
		return
	}

	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Validation failed", "ticketID must be a valid UUID", map[string][]string{"ticketid": {"must be a valid UUID"}})
		return
	}

	ticket, err := h.getTicketExecutor.Execute(r.Context(), query.GetTicketQuery{
		TicketID:   ticketID,
		Principal:  principal,
		Resource:   "tickets",
		ActionName: "read",
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrForbidden):
			writeProblem(w, r, http.StatusForbidden, "Forbidden", "you are not allowed to access this ticket", nil)
		case errors.Is(err, apperrors.ErrNotFound):
			writeProblem(w, r, http.StatusNotFound, "Not Found", "ticket was not found", nil)
		default:
			writeProblem(w, r, http.StatusInternalServerError, "Internal Server Error", "unexpected error", nil)
		}

		return
	}

	writeJSON(w, http.StatusOK, toTicketResponse(ticket))
}

func (h *Handlers) ListTickets(w http.ResponseWriter, r *http.Request) {
	principal, ok := transport.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "missing authenticated principal", nil)
		return
	}

	tickets, err := h.listTicketsExecutor.Execute(r.Context(), query.ListTicketsQuery{Principal: principal})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrForbidden):
			writeProblem(w, r, http.StatusForbidden, "Forbidden", "you are not allowed to list tickets", nil)
		default:
			writeProblem(w, r, http.StatusInternalServerError, "Internal Server Error", "unexpected error", nil)
		}
		return
	}

	responses := make([]TicketResponse, 0, len(tickets))
	for _, ticket := range tickets {
		responses = append(responses, toTicketResponse(ticket))
	}

	writeJSON(w, http.StatusOK, responses)
}

func (h *Handlers) ListDocuments(w http.ResponseWriter, r *http.Request) {
	principal, ok := transport.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "missing authenticated principal", nil)
		return
	}

	documents, err := h.listDocumentsExecutor.Execute(r.Context(), query.ListDocumentsQuery{Principal: principal})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrForbidden):
			writeProblem(w, r, http.StatusForbidden, "Forbidden", "you are not allowed to list documents", nil)
		default:
			writeProblem(w, r, http.StatusInternalServerError, "Internal Server Error", "unexpected error", nil)
		}
		return
	}

	responses := make([]DocumentResponse, 0, len(documents))
	for _, document := range documents {
		responses = append(responses, toDocumentResponse(document))
	}

	writeJSON(w, http.StatusOK, responses)

}

func (h *Handlers) IngestDocument(w http.ResponseWriter, r *http.Request) {
	principal, ok := transport.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "missing authenticated principal", nil)
		return
	}

	var body IngestDocumentRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Invalid request", fmt.Sprintf("decode request body: %v", err), nil)
		return
	}

	document, err := h.ingestDocumentService.Execute(r.Context(), command.IngestDocumentCommand{
		Principal: principal,
		Title:     body.Title,
		SourceURI: body.SourceUri,
		Content:   body.Content,
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrForbidden):
			writeProblem(w, r, http.StatusForbidden, "Forbidden", "you are not allowed to ingest documents", nil)
		default:
			writeProblem(w, r, http.StatusInternalServerError, "Internal Server Error", "unexpected error", nil)
		}
		return
	}

	writeJSON(w, http.StatusCreated, toDocumentResponse(document))
}

func toTicketResponse(ticket entity.Ticket) TicketResponse {
	return TicketResponse{
		Id:               ticket.ID,
		Subject:          ticket.Subject,
		Status:           ticket.Status,
		TenantId:         openapi_types.UUID(ticket.TenantID),
		CreatedByUserId:  openapi_types.UUID(ticket.CreatedByUserID),
		AssignedToUserId: toAssignedToUserID(ticket.AssignedToUserID),
	}
}

func toAssignedToUserID(id *uuid.UUID) *openapi_types.UUID {
	if id == nil {
		return nil
	}
	u := openapi_types.UUID(*id)
	return &u
}

func toDocumentResponse(document entity.KnowledgeDocument) DocumentResponse {
	return DocumentResponse{
		Id:              openapi_types.UUID(document.ID),
		TenantId:        openapi_types.UUID(document.TenantID),
		Title:           document.Title,
		SourceUri:       document.SourceURI,
		CreatedByUserId: openapi_types.UUID(document.CreatedByUserID),
		CreatedAt:       document.CreatedAt,
	}
}
