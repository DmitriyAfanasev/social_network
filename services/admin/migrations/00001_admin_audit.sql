-- +goose Up

CREATE TABLE admin.moderation_audit_logs (
    id uuid PRIMARY KEY,
    event_id uuid NOT NULL UNIQUE,
    actor_id uuid,
    action varchar(100) NOT NULL,
    target_type varchar(50) NOT NULL,
    target_id uuid,
    details jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX admin_moderation_audit_created_idx
    ON admin.moderation_audit_logs (created_at DESC, id DESC);

CREATE INDEX admin_moderation_audit_actor_idx
    ON admin.moderation_audit_logs (actor_id, created_at DESC);

-- +goose Down

DROP TABLE admin.moderation_audit_logs;
