-- +goose Up

CREATE SCHEMA IF NOT EXISTS analytics;

CREATE TABLE analytics.outbox_events (
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

CREATE INDEX analytics_outbox_pending_idx
    ON analytics.outbox_events (available_at, created_at)
    WHERE published_at IS NULL;

CREATE TABLE analytics.processed_events (
    event_id uuid PRIMARY KEY,
    status varchar(16) NOT NULL CHECK (status IN ('processing', 'processed')),
    claimed_at timestamptz NOT NULL DEFAULT now(),
    processed_at timestamptz
);

CREATE INDEX analytics_processed_events_claimed_idx
    ON analytics.processed_events (claimed_at)
    WHERE status = 'processing';

CREATE TABLE analytics.event_records (
    event_id uuid PRIMARY KEY,
    event_type varchar(160) NOT NULL,
    aggregate_id uuid,
    payload jsonb NOT NULL,
    correlation_id varchar(128),
    created_at timestamptz NOT NULL
);

-- +goose Down

DROP TABLE analytics.event_records;
DROP TABLE analytics.processed_events;
DROP TABLE analytics.outbox_events;
