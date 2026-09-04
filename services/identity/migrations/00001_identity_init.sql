-- +goose Up

CREATE SCHEMA IF NOT EXISTS identity;

CREATE TABLE identity.users (
    id uuid PRIMARY KEY,
    email varchar(320) NOT NULL UNIQUE,
    status varchar(20) NOT NULL DEFAULT 'active',
    last_seen_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE identity.credentials (
    user_id uuid PRIMARY KEY REFERENCES identity.users (id) ON DELETE CASCADE,
    password_hash text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE identity.refresh_tokens (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
    token_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX refresh_tokens_user_id_idx ON identity.refresh_tokens (user_id);
CREATE INDEX refresh_tokens_active_idx ON identity.refresh_tokens (user_id, expires_at) WHERE revoked_at IS NULL;

CREATE TABLE identity.verification_tokens (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
    purpose varchar(32) NOT NULL,
    token_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX verification_tokens_lookup_idx ON identity.verification_tokens (user_id, purpose, expires_at) WHERE consumed_at IS NULL;

CREATE TABLE identity.roles (
    id uuid PRIMARY KEY,
    name varchar(50) NOT NULL UNIQUE,
    description varchar(255),
    is_system boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE identity.permissions (
    id uuid PRIMARY KEY,
    code varchar(100) NOT NULL UNIQUE,
    description varchar(255),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE identity.user_roles (
    user_id uuid NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
    role_id uuid NOT NULL REFERENCES identity.roles (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE identity.role_permissions (
    role_id uuid NOT NULL REFERENCES identity.roles (id) ON DELETE CASCADE,
    permission_id uuid NOT NULL REFERENCES identity.permissions (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, permission_id)
);

-- +goose StatementBegin
INSERT INTO identity.roles (id, name, description)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'user', 'Обычный пользователь'),
    ('00000000-0000-0000-0000-000000000002', 'moderator', 'Модератор'),
    ('00000000-0000-0000-0000-000000000003', 'admin', 'Администратор');
-- +goose StatementEnd

-- +goose Down

DROP TABLE identity.role_permissions;
DROP TABLE identity.user_roles;
DROP TABLE identity.permissions;
DROP TABLE identity.roles;
DROP TABLE identity.verification_tokens;
DROP TABLE identity.refresh_tokens;
DROP TABLE identity.credentials;
DROP TABLE identity.users;
