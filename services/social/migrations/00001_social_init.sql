-- +goose Up

CREATE SCHEMA IF NOT EXISTS social;

CREATE TABLE social.user_blocks (
    blocker_id uuid NOT NULL,
    blocked_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (blocker_id, blocked_id),
    CONSTRAINT user_blocks_not_self_check CHECK (blocker_id <> blocked_id)
);
CREATE INDEX user_blocks_blocked_idx ON social.user_blocks (blocked_id, blocker_id);

CREATE TABLE social.subscriptions (
    subscriber_id uuid NOT NULL,
    target_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (subscriber_id, target_id),
    CONSTRAINT subscriptions_not_self_check CHECK (subscriber_id <> target_id)
);
CREATE INDEX subscriptions_target_idx ON social.subscriptions (target_id, subscriber_id);

CREATE TABLE social.friendships (
    user_id uuid NOT NULL,
    friend_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, friend_id),
    CONSTRAINT friendships_normalized_pair_check CHECK (user_id < friend_id)
);
CREATE INDEX friendships_friend_idx ON social.friendships (friend_id, user_id);

-- +goose Down

DROP TABLE social.friendships;
DROP TABLE social.subscriptions;
DROP TABLE social.user_blocks;
