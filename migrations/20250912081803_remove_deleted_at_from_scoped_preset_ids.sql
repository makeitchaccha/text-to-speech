-- +goose Up
-- +goose StatementBegin
DROP INDEX idx_deleted_at;
ALTER TABLE scoped_preset_ids DROP COLUMN deleted_at;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE scoped_preset_ids ADD COLUMN deleted_at DATETIME;
CREATE INDEX idx_deleted_at ON scoped_preset_ids (deleted_at);
-- +goose StatementEnd
