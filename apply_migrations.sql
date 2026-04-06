-- Migration 0001: Add Media
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
    local_path      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status          TEXT        NOT NULL DEFAULT 'pending',

    CONSTRAINT media_user_checksum_uniq UNIQUE (user_id, checksum)
);

CREATE INDEX IF NOT EXISTS idx_media_user_created_at
    ON media (user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_media_user_mime
    ON media (user_id, mime_type);

CREATE INDEX IF NOT EXISTS idx_media_datetime
    ON media (datetime_original);

-- Migration 0002: Add Embeddings
CREATE TABLE IF NOT EXISTS embeddings (
  media_id   BIGINT REFERENCES media(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL, 
  model      TEXT   NOT NULL,
  vec_bytes  BYTEA  NOT NULL,
  status     TEXT   NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','in_index','failed')),
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (media_id, model),
  CHECK (octet_length(vec_bytes) % 4 = 0)
);

CREATE INDEX IF NOT EXISTS idx_embeddings_user_status ON embeddings (user_id, status, updated_at);
CREATE INDEX IF NOT EXISTS idx_embeddings_status ON embeddings (status, updated_at);
