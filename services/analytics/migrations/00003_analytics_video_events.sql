-- +goose Up

CREATE TABLE analytics.video_events (
    event_id uuid PRIMARY KEY,
    event_type varchar(160) NOT NULL,
    video_id uuid NOT NULL,
    user_id uuid,
    session_id varchar(128),
    watch_seconds numeric(12,3) NOT NULL DEFAULT 0 CHECK (watch_seconds >= 0),
    progress_seconds numeric(12,3) NOT NULL DEFAULT 0 CHECK (progress_seconds >= 0),
    duration_seconds numeric(12,3) NOT NULL DEFAULT 0 CHECK (duration_seconds >= 0),
    completed boolean NOT NULL DEFAULT false,
    occurred_at timestamptz NOT NULL
);

CREATE INDEX analytics_video_events_video_idx
    ON analytics.video_events (video_id, occurred_at DESC);

CREATE INDEX analytics_video_events_user_idx
    ON analytics.video_events (user_id, occurred_at DESC)
    WHERE user_id IS NOT NULL;

CREATE INDEX analytics_video_events_type_idx
    ON analytics.video_events (event_type, occurred_at DESC);

-- +goose Down

DROP TABLE analytics.video_events;
