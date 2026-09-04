-- +goose Up

ALTER TABLE profiles.profiles
    ADD COLUMN message_policy varchar(24) NOT NULL DEFAULT 'everyone';

ALTER TABLE profiles.profiles
    ADD CONSTRAINT profiles_message_policy_check
    CHECK (message_policy IN ('everyone', 'friends', 'friends_of_friends'));

-- +goose Down

ALTER TABLE profiles.profiles
    DROP CONSTRAINT profiles_message_policy_check;

ALTER TABLE profiles.profiles
    DROP COLUMN message_policy;
