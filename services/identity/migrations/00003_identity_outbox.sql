-- +goose Up

CREATE TABLE identity.outbox_events (
    id uuid PRIMARY KEY,
    event_type varchar(160) NOT NULL,
    aggregate_id uuid,
    payload jsonb NOT NULL,
    correlation_id varchar(255) NOT NULL DEFAULT '',
    attempts integer NOT NULL DEFAULT 0,
    available_at timestamptz NOT NULL DEFAULT now(),
    locked_at timestamptz,
    published_at timestamptz,
    last_error text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX identity_outbox_pending_idx
    ON identity.outbox_events (available_at, created_at)
    WHERE published_at IS NULL;

-- +goose Down

DROP TABLE identity.outbox_events;
