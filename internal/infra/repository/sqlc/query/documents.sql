-- name: CreateDocument :one
INSERT INTO knowledge_documents (
	id,
	tenant_id,
	title,
	source_uri,
	created_by_user_id,
	created_at
) VALUES (
	$1, $2, $3, $4, $5, $6
)
RETURNING id, tenant_id, title, source_uri, created_by_user_id, created_at;

-- name: GetDocumentByID :one
SELECT id, tenant_id, title, source_uri, created_by_user_id, created_at
FROM knowledge_documents
WHERE id = $1
LIMIT 1;

-- name: ListDocumentsByTenant :many
SELECT id, tenant_id, title, source_uri, created_by_user_id, created_at
FROM knowledge_documents
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: DeleteDocument :exec
DELETE FROM knowledge_documents
WHERE id = $1;

-- name: InsertDocumentChunk :exec
INSERT INTO document_chunks (
	id,
	document_id,
	tenant_id,
	chunk_index,
	content,
	embedding_model,
	embedding,
	created_at
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8
);

-- name: SearchSimilarChunks :many
SELECT id, document_id, tenant_id, chunk_index, content, embedding_model, embedding, created_at
FROM document_chunks
WHERE tenant_id = $1
ORDER BY embedding <=> $2
LIMIT $3;
