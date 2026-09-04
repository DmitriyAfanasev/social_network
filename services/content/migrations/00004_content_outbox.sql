-- +goose Up

CREATE TABLE content.outbox_events (
    id uuid PRIMARY KEY,
    event_type varchar(160) NOT NULL,
    aggregate_id uuid,
    payload jsonb NOT NULL,
    correlation_id varchar(128),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    available_at timestamptz NOT NULL DEFAULT now(),
    locked_at timestamptz,
    published_at timestamptz,
    last_error text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX content_outbox_pending_idx
    ON content.outbox_events (available_at, created_at)
    WHERE published_at IS NULL;

-- +goose Down

DROP TABLE content.outbox_events;
