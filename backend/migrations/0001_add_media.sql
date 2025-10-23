-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS media (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT      NOT NULL,
    url             TEXT        NOT NULL,
    thumb_url       TEXT,
    mime_type       TEXT        NOT NULL,
    size_bytes      BIGINT      NOT NULL CHECK (size_bytes >= 0),
    checksum        TEXT        NOT NULL,
    datetime_original TIMESTAMPTZ,
    orientation     INT         NOT NULL DEFAULT 1,
    width           INT,
    height          INT,
    camera_make     TEXT,
    camera_model    TEXT,
    software        TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT media_user_checksum_uniq UNIQUE (user_id, checksum)
);

CREATE INDEX IF NOT EXISTS idx_media_user_created_at
    ON media (user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_media_user_mime
    ON media (user_id, mime_type);

CREATE INDEX IF NOT EXISTS idx_media_datetime
    ON media (datetime_original);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_media_datetime;
DROP INDEX IF EXISTS idx_media_user_mime;
DROP INDEX IF EXISTS idx_media_user_created_at;
DROP TABLE IF EXISTS media;
-- +goose StatementEnd
