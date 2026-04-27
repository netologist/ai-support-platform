package graphql

import (
	"encoding/json"
	"net/http"

	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"

	"github.com/netologist/ai-support-platform/internal/app/query"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	appTransport "github.com/netologist/ai-support-platform/internal/transport"
	"github.com/netologist/ai-support-platform/internal/transport/graphql/model"
)

type Dependencies struct {
	GetTicketService      query.GetTicketService
	ListTicketsService    query.ListTicketsService
	ListDocumentsService  query.ListDocumentsService
	SemanticSearchService query.SemanticSearchService
	SuggestReplyService   query.SuggestReplyService
}

// NewHandler returns an http.Handler backed by the gqlgen execution engine.
// It wraps the gqlgen server with a thin middleware that rejects requests
// that have no authenticated principal in the context (defense-in-depth;
// the router-level auth middleware already enforces this before the handler
// is reached in production).
func NewHandler(deps Dependencies) http.Handler {
	resolver := &Resolver{deps: deps}
	schema := NewExecutableSchema(Config{Resolvers: resolver})

	srv := gqlhandler.New(schema)
	srv.AddTransport(transport.POST{})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := appTransport.PrincipalFromContext(r.Context()); !ok {
			writeGQLError(w, http.StatusUnauthorized, "missing authenticated principal")
			return
		}
		srv.ServeHTTP(w, r)
	})
}

// --- gqlgen response helpers ------------------------------------------------

type graphQLResponse struct {
	Data   any          `json:"data,omitempty"`
	Errors []graphQLErr `json:"errors,omitempty"`
}

type graphQLErr struct {
	Message string `json:"message"`
}

func writeGQLError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(graphQLResponse{
		Errors: []graphQLErr{{Message: message}},
	})
}

// --- domain → gqlgen model converters ----------------------------------------

func domainTicketToModel(t entity.Ticket) *model.Ticket {
	m := &model.Ticket{
		ID:              t.ID.String(),
		TenantID:        t.TenantID.String(),
		Subject:         t.Subject,
		Status:          t.Status,
		CreatedByUserID: t.CreatedByUserID.String(),
		CreatedAt:       t.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if t.AssignedToUserID != nil {
		s := t.AssignedToUserID.String()
		m.AssignedToUserID = &s
	}
	return m
}

func domainDocumentToModel(d entity.KnowledgeDocument) *model.Document {
	return &model.Document{
		ID:              d.ID.String(),
		TenantID:        d.TenantID.String(),
		Title:           d.Title,
		SourceURI:       d.SourceURI,
		CreatedByUserID: d.CreatedByUserID.String(),
		CreatedAt:       d.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func domainChunkToModel(c entity.DocumentChunk) *model.DocumentChunk {
	return &model.DocumentChunk{
		ID:             c.ID.String(),
		DocumentID:     c.DocumentID.String(),
		ChunkIndex:     int32(c.ChunkIndex),
		Content:        c.Content,
		EmbeddingModel: c.EmbeddingModel,
	}
}

