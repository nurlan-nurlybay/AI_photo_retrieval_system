-- +goose Up
ALTER TABLE media ADD COLUMN local_path TEXT;
CREATE INDEX IF NOT EXISTS idx_media_local_path ON media (user_id, local_path);


-- +goose Down
DROP INDEX IF EXISTS idx_media_local_path;
ALTER TABLE media DROP COLUMN IF EXISTS local_path;
