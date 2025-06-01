-- +goose Up
CREATE TABLE scoped_preset_ids (
    scope VARCHAR(255) NOT NULL,
    id BIGINT NOT NULL,
    preset_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP NULL,
    PRIMARY KEY (scope, id),
    INDEX idx_deleted_at (deleted_at)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE scoped_preset_ids;
-- +goose StatementEnd
