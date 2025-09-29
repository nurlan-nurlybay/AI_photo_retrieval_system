-- id BIGSERIAL if you need int64 for FAISS; or UUID if you prefer
CREATE TABLE IF NOT EXISTS media (
  id         BIGSERIAL PRIMARY KEY,
  device_id  TEXT, -- NULL for MVP
  url        TEXT NOT NULL,
  thumb_url  TEXT,
  label      TEXT,
  deleted    BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- If you query by device_id+id combos a lot:
CREATE INDEX IF NOT EXISTS idx_media_device_created ON media (device_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_media_device_id ON media (device_id, id);
