-- +goose Up
ALTER TABLE media ADD COLUMN status TEXT NOT NULL DEFAULT 'active';

-- +goose Down
ALTER TABLE media DROP COLUMN IF EXISTS status;
