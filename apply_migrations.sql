CREATE TABLE IF NOT EXISTS media (
    id                BIGSERIAL PRIMARY KEY,
    user_id           BIGINT      NOT NULL,
    url               TEXT        NOT NULL,
    thumb_url         TEXT,
    mime_type         TEXT        NOT NULL,
    size_bytes        BIGINT      NOT NULL CHECK (size_bytes >= 0),
    checksum          TEXT        NOT NULL,
    datetime_original TIMESTAMPTZ,
    orientation       INT         NOT NULL DEFAULT 1,
    width             INT,
    height            INT,
    camera_make       TEXT,
    camera_model      TEXT,
    software          TEXT,
    local_path        TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status            TEXT        NOT NULL DEFAULT 'pending',
    CONSTRAINT media_user_checksum_uniq UNIQUE (user_id, checksum)
);

CREATE TABLE IF NOT EXISTS embeddings (
  media_id   BIGINT REFERENCES media(id) ON DELETE CASCADE,
  user_id    BIGINT NOT NULL,
  model      TEXT   NOT NULL,
  vec_bytes  BYTEA,
  status     TEXT   NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'fast_encoded', 'slow_encoded', 'in_index', 'failed')),
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (media_id, model),
  CHECK (vec_bytes IS NULL OR octet_length(vec_bytes) % 4 = 0)
);
