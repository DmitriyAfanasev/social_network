-- +goose Up

CREATE SCHEMA IF NOT EXISTS messaging;

CREATE TABLE messaging.conversations (
    id uuid PRIMARY KEY,
    direct_key varchar(80) NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE messaging.messages (
    id uuid PRIMARY KEY,
    conversation_id uuid NOT NULL REFERENCES messaging.conversations(id) ON DELETE CASCADE,
    sender_id uuid NOT NULL,
    body text NOT NULL,
    media_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    edited_at timestamptz,
    deleted_at timestamptz,
    CONSTRAINT messages_body_length_check CHECK (char_length(body) BETWEEN 1 AND 5000)
);

CREATE TABLE messaging.conversation_participants (
    conversation_id uuid NOT NULL REFERENCES messaging.conversations(id) ON DELETE CASCADE,
    user_id uuid NOT NULL,
    last_read_message_id uuid,
    read_at timestamptz,
    archived_at timestamptz,
    hidden_at timestamptz,
    pinned_at timestamptz,
    muted_at timestamptz,
    cleared_at timestamptz,
    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX conversation_participants_user_idx ON messaging.conversation_participants (user_id, conversation_id);
CREATE INDEX messages_conversation_created_idx ON messaging.messages (conversation_id, created_at DESC, id DESC);
CREATE INDEX messages_sender_created_idx ON messaging.messages (sender_id, created_at DESC);

-- +goose Down

DROP TABLE messaging.conversation_participants;
DROP TABLE messaging.messages;
DROP TABLE messaging.conversations;
