-- +goose Up

ALTER TABLE profiles.profiles
    DROP CONSTRAINT IF EXISTS profiles_message_policy_check;

ALTER TABLE profiles.profiles
    ADD CONSTRAINT profiles_message_policy_check
    CHECK (message_policy IN ('everyone', 'friends', 'friends_of_friends', 'nobody'));

-- +goose Down

UPDATE profiles.profiles
SET message_policy = 'everyone'
WHERE message_policy = 'nobody';

ALTER TABLE profiles.profiles
    DROP CONSTRAINT IF EXISTS profiles_message_policy_check;

ALTER TABLE profiles.profiles
    ADD CONSTRAINT profiles_message_policy_check
    CHECK (message_policy IN ('everyone', 'friends', 'friends_of_friends'));
