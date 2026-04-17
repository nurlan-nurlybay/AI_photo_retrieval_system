-- +goose Up
-- +goose StatementBegin

-- The embeddings table was created with media_id as the sole PRIMARY KEY,
-- but UpsertEmbedding and ClaimForProcessing both use ON CONFLICT (media_id, model),
-- which requires a composite unique constraint. Fix that by promoting the PK to the
-- composite (media_id, model). This also enables one row per model per image, which
-- is required now that we have both siglip and qwen models.
--
-- Note: the FOREIGN KEY on media_id survives this change because it is a separate
-- constraint from the primary key. If existing rows have duplicate (media_id, model)
-- pairs, this migration will fail — deduplicate first with:
--   DELETE FROM embeddings a USING embeddings b
--     WHERE a.ctid > b.ctid AND a.media_id = b.media_id AND a.model = b.model;

ALTER TABLE embeddings DROP CONSTRAINT embeddings_pkey;
ALTER TABLE embeddings ADD PRIMARY KEY (media_id, model);

-- ClaimForProcessing sets status = 'processing', but the original CHECK only
-- allowed pending/in_index/failed. Add 'processing' so the INSERT doesn't violate it.
ALTER TABLE embeddings DROP CONSTRAINT embeddings_status_check;
ALTER TABLE embeddings ADD CONSTRAINT embeddings_status_check
    CHECK (status IN ('pending', 'processing', 'in_index', 'failed'));

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE embeddings DROP CONSTRAINT embeddings_pkey;
ALTER TABLE embeddings ADD PRIMARY KEY (media_id);

ALTER TABLE embeddings DROP CONSTRAINT embeddings_status_check;
ALTER TABLE embeddings ADD CONSTRAINT embeddings_status_check
    CHECK (status IN ('pending', 'in_index', 'failed'));

-- +goose StatementEnd
