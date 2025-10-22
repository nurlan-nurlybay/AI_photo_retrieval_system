-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS media (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT      NOT NULL,
    url          TEXT        NOT NULL,
    thumb_url    TEXT,
    mime_type    TEXT        NOT NULL,
    size_bytes   BIGINT      NOT NULL CHECK (size_bytes >= 0),
    checksum     TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Prevent dup uploads per user
    CONSTRAINT media_user_checksum_uniq UNIQUE (user_id, checksum)
);

CREATE INDEX IF NOT EXISTS idx_media_user_created_at
    ON media (user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_media_user_mime
    ON media (user_id, mime_type);

CREATE TABLE IF NOT EXISTS media_metadata (
    media_id            BIGINT PRIMARY KEY REFERENCES media(id) ON DELETE CASCADE,
    datetime_original   TIMESTAMPTZ,
    orientation         INT        NOT NULL DEFAULT 1,
    width               INT,
    height              INT,
    file_format         TEXT,              -- "jpeg","png","heic"
    camera_make         TEXT,
    camera_model        TEXT,
    software            TEXT
);

-- filter by shot date
CREATE INDEX IF NOT EXISTS idx_meta_datetime
    ON media_metadata (datetime_original);
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS media_metadata;
DROP TABLE IF EXISTS media;
-- +goose StatementEnd
