-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE knowledge_documents (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    source_uri TEXT NOT NULL,
    created_by_user_id UUID NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE document_chunks (
    id UUID PRIMARY KEY,
    document_id UUID NOT NULL REFERENCES knowledge_documents (id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,
    content TEXT NOT NULL,
    embedding_model TEXT NOT NULL,
    embedding vector(1536),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX knowledge_documents_tenant_id_idx ON knowledge_documents (tenant_id);
CREATE INDEX document_chunks_document_id_idx ON document_chunks (document_id);
CREATE INDEX document_chunks_tenant_id_idx ON document_chunks (tenant_id);

-- +goose Down
DROP INDEX IF EXISTS document_chunks_tenant_id_idx;
DROP INDEX IF EXISTS document_chunks_document_id_idx;
DROP INDEX IF EXISTS knowledge_documents_tenant_id_idx;
DROP TABLE IF EXISTS document_chunks;
DROP TABLE IF EXISTS knowledge_documents;
DROP EXTENSION IF EXISTS vector;