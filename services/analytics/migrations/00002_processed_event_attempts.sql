-- +goose Up

ALTER TABLE analytics.processed_events
    ADD COLUMN attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    ADD COLUMN last_error text;

-- +goose Down

ALTER TABLE analytics.processed_events
    DROP COLUMN last_error,
    DROP COLUMN attempts;
