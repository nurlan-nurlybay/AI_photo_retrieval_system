-- +goose Up
-- +goose StatementBegin
CREATE TABLE embeddings (
  media_id   BIGINT PRIMARY KEY REFERENCES media(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL, 
  model      TEXT   NOT NULL,              -- e.g. "open_clip:ViT-L/14@336px"
  vec_bytes  BYTEA  NOT NULL,              -- raw float32[] vector data
  status     TEXT   NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','in_index','failed')),
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (octet_length(vec_bytes) % 4 = 0)
);

CREATE INDEX idx_embeddings_user_status ON embeddings (user_id, status, updated_at);
CREATE INDEX idx_embeddings_status ON embeddings (status, updated_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_embeddings_user_status;
DROP INDEX IF EXISTS idx_embeddings_status;
DROP TABLE IF EXISTS embeddings;
-- +goose StatementEnd
