CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS media (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),          -- or pass your own string ids
  device_id  text NOT NULL,
  url        text NOT NULL,
  thumb_url  text,                                                -- nullable for now
  created_at timestamptz NOT NULL DEFAULT now(),
  deleted    boolean NOT NULL DEFAULT false,

  CONSTRAINT url_nonempty CHECK (length(url) > 0)
);

CREATE INDEX IF NOT EXISTS idx_media_device_created   ON media (device_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_media_deleted_created  ON media (deleted, created_at DESC);
