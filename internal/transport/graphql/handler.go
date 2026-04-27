package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/app/query"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/transport"
)

type Dependencies struct {
	GetTicketService      query.GetTicketService
	ListTicketsService    query.ListTicketsService
	ListDocumentsService  query.ListDocumentsService
	SemanticSearchService query.SemanticSearchService
	SuggestReplyService   query.SuggestReplyService
}

type Handler struct {
	deps Dependencies
}

func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type graphQLResponse struct {
	Data   any          `json:"data,omitempty"`
	Errors []graphQLErr `json:"errors,omitempty"`
}

type graphQLErr struct {
	Message string `json:"message"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeGQLError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req graphQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeGQLError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	principal, ok := transport.PrincipalFromContext(r.Context())
	if !ok {
		writeGQLError(w, http.StatusUnauthorized, "missing authenticated principal")
		return
	}

	data := map[string]any{}

	if strings.Contains(req.Query, "ticket(") {
		id, ok := stringVariable(req.Variables, "id", "ticketId")
		if !ok {
			writeGQLError(w, http.StatusBadRequest, "missing ticket id variable")
			return
		}

		result, err := h.ResolveTicket(r.Context(), principal, id)
		if err != nil {
			writeGraphQLErrorForAppError(w, err)
			return
		}
		data["ticket"] = result
	}

	if strings.Contains(req.Query, "tickets") && !strings.Contains(req.Query, "ticket(") {
		result, err := h.ResolveTickets(r.Context(), principal)
		if err != nil {
			writeGraphQLErrorForAppError(w, err)
			return
		}
		data["tickets"] = result
	}

	if strings.Contains(req.Query, "documents") && !strings.Contains(req.Query, "searchDocuments") {
		result, err := h.ResolveDocuments(r.Context(), principal)
		if err != nil {
			writeGraphQLErrorForAppError(w, err)
			return
		}
		data["documents"] = result
	}

	if strings.Contains(req.Query, "suggestReply(") {
		id, ok := stringVariable(req.Variables, "ticketId", "id")
		if !ok {
			writeGQLError(w, http.StatusBadRequest, "missing ticketId variable")
			return
		}

		result, err := h.ResolveSuggestReply(r.Context(), principal, id)
		if err != nil {
			writeGraphQLErrorForAppError(w, err)
			return
		}
		data["suggestReply"] = result
	}

	if strings.Contains(req.Query, "searchDocuments(") {
		queryText, ok := stringVariable(req.Variables, "query", "text")
		if !ok {
			writeGQLError(w, http.StatusBadRequest, "missing query variable")
			return
		}

		limit := intVariable(req.Variables, 5, "limit")
		result, err := h.ResolveSearchDocuments(r.Context(), principal, queryText, limit)
		if err != nil {
			writeGraphQLErrorForAppError(w, err)
			return
		}
		data["searchDocuments"] = result
	}

	if len(data) == 0 {
		writeGQLError(w, http.StatusBadRequest, "unsupported query")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(graphQLResponse{Data: data})
}

func (h *Handler) ResolveTicket(ctx context.Context, principal entity.Principal, id string) (any, error) {
	ticketID, err := uuid.Parse(id)
	if err != nil {
		return nil, apperrors.ErrInvalidArgument
	}

	ticket, err := h.deps.GetTicketService.Execute(ctx, query.GetTicketQuery{
		TicketID:   ticketID,
		Principal:  principal,
		Resource:   "tickets",
		ActionName: "read",
	})
	if err != nil {
		return nil, err
	}

	return ticketToGQL(ticket), nil
}

func (h *Handler) ResolveTickets(ctx context.Context, principal entity.Principal) (any, error) {
	tickets, err := h.deps.ListTicketsService.Execute(ctx, query.ListTicketsQuery{Principal: principal})
	if err != nil {
		return nil, err
	}

	result := make([]map[string]any, 0, len(tickets))
	for _, t := range tickets {
		result = append(result, ticketToGQL(t))
	}

	return result, nil
}

func (h *Handler) ResolveDocuments(ctx context.Context, principal entity.Principal) (any, error) {
	documents, err := h.deps.ListDocumentsService.Execute(ctx, query.ListDocumentsQuery{Principal: principal})
	if err != nil {
		return nil, err
	}

	result := make([]map[string]any, 0, len(documents))
	for _, d := range documents {
		result = append(result, documentToGQL(d))
	}

	return result, nil
}

func (h *Handler) ResolveSuggestReply(ctx context.Context, principal entity.Principal, ticketID string) (any, error) {
	id, err := uuid.Parse(ticketID)
	if err != nil {
		return nil, apperrors.ErrInvalidArgument
	}

	result, err := h.deps.SuggestReplyService.Execute(ctx, query.SuggestReplyQuery{
		Principal: principal,
		TicketID:  id,
	})
	if err != nil {
		return nil, err
	}

	sources := make([]map[string]any, 0, len(result.Sources))
	for _, s := range result.Sources {
		sources = append(sources, chunkToGQL(s))
	}

	return map[string]any{
		"suggestedReply": result.SuggestedReply,
		"sources":        sources,
	}, nil
}

func (h *Handler) ResolveSearchDocuments(ctx context.Context, principal entity.Principal, queryText string, limit int) (any, error) {
	result, err := h.deps.SemanticSearchService.Execute(ctx, query.SemanticSearchQuery{
		Principal: principal,
		Text:      queryText,
		Limit:     limit,
	})
	if err != nil {
		return nil, err
	}

	chunks := make([]map[string]any, 0, len(result.Chunks))
	for _, c := range result.Chunks {
		chunks = append(chunks, chunkToGQL(c))
	}

	return chunks, nil
}

func ticketToGQL(t entity.Ticket) map[string]any {
	m := map[string]any{
		"id":              t.ID.String(),
		"tenantId":        t.TenantID.String(),
		"subject":         t.Subject,
		"status":          t.Status,
		"createdByUserId": t.CreatedByUserID.String(),
		"createdAt":       t.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if t.AssignedToUserID != nil {
		m["assignedToUserId"] = t.AssignedToUserID.String()
	}

	return m
}

func chunkToGQL(c entity.DocumentChunk) map[string]any {
	return map[string]any{
		"id":             c.ID.String(),
		"documentId":     c.DocumentID.String(),
		"chunkIndex":     c.ChunkIndex,
		"content":        c.Content,
		"embeddingModel": c.EmbeddingModel,
	}
}

func documentToGQL(d entity.KnowledgeDocument) map[string]any {
	return map[string]any{
		"id":              d.ID.String(),
		"tenantId":        d.TenantID.String(),
		"title":           d.Title,
		"sourceUri":       d.SourceURI,
		"createdByUserId": d.CreatedByUserID.String(),
		"createdAt":       d.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func writeGQLError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(graphQLResponse{
		Errors: []graphQLErr{{Message: message}},
	})
}

func writeGraphQLErrorForAppError(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		writeGQLError(w, http.StatusInternalServerError, "unexpected error")
	case strings.Contains(err.Error(), apperrors.ErrInvalidArgument.Error()):
		writeGQLError(w, http.StatusBadRequest, err.Error())
	case strings.Contains(err.Error(), apperrors.ErrForbidden.Error()):
		writeGQLError(w, http.StatusForbidden, err.Error())
	case strings.Contains(err.Error(), apperrors.ErrNotFound.Error()):
		writeGQLError(w, http.StatusNotFound, err.Error())
	default:
		writeGQLError(w, http.StatusInternalServerError, err.Error())
	}
}

func stringVariable(variables map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		value, ok := variables[key]
		if !ok {
			continue
		}

		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			return s, true
		}
	}

	return "", false
}

func intVariable(variables map[string]any, fallback int, keys ...string) int {
	for _, key := range keys {
		value, ok := variables[key]
		if !ok {
			continue
		}

		switch v := value.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}

	return fallback
}
