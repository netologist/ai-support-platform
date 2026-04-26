package executor

import (
	"github.com/netologist/ai-support-platform/internal/app/command"
	"github.com/netologist/ai-support-platform/internal/app/query"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

// Convenience aliases for concrete ticket command executors.
type IngestDocumentExecutor Executor[command.IngestDocumentCommand, entity.KnowledgeDocument]

type ListDocumentsExecutor Executor[query.ListDocumentsQuery, []entity.KnowledgeDocument]
type SemanticSearchExecutor Executor[query.SemanticSearchQuery, query.SemanticSearchResult]
type SuggestReplyExecutor Executor[query.SuggestReplyQuery, query.SuggestReplyResult]
