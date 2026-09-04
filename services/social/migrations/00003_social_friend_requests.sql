-- +goose Up

CREATE TABLE social.friend_requests (
    id uuid PRIMARY KEY,
    sender_id uuid NOT NULL,
    recipient_id uuid NOT NULL,
    pair_key text GENERATED ALWAYS AS (
        LEAST(sender_id::text, recipient_id::text) || ':' || GREATEST(sender_id::text, recipient_id::text)
    ) STORED,
    status varchar(16) NOT NULL DEFAULT 'pending',
    created_at timestamptz NOT NULL DEFAULT now(),
    responded_at timestamptz,
    CONSTRAINT friend_requests_not_self_check CHECK (sender_id <> recipient_id),
    CONSTRAINT friend_requests_status_check CHECK (status IN ('pending', 'accepted', 'declined', 'cancelled')),
    CONSTRAINT friend_requests_response_time_check CHECK (
        (status = 'pending' AND responded_at IS NULL)
        OR (status <> 'pending' AND responded_at IS NOT NULL)
    )
);

CREATE UNIQUE INDEX friend_requests_pending_pair_idx
    ON social.friend_requests (pair_key)
    WHERE status = 'pending';
CREATE INDEX friend_requests_recipient_pending_idx
    ON social.friend_requests (recipient_id, created_at, id)
    WHERE status = 'pending';
CREATE INDEX friend_requests_sender_pending_idx
    ON social.friend_requests (sender_id, created_at, id)
    WHERE status = 'pending';

-- +goose Down

DROP TABLE social.friend_requests;
