-- +goose Up

ALTER TABLE admin.moderation_audit_logs
    ADD COLUMN correlation_id varchar(128);

-- +goose Down

ALTER TABLE admin.moderation_audit_logs
    DROP COLUMN correlation_id;
