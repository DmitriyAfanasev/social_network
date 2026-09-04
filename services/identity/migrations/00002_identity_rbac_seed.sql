-- +goose Up

INSERT INTO identity.permissions (id, code, description)
VALUES
    ('00000000-0000-0000-0000-000000000101', 'comments.delete_any', 'Удалять чужие комментарии'),
    ('00000000-0000-0000-0000-000000000102', 'users.block', 'Блокировать пользователей'),
    ('00000000-0000-0000-0000-000000000103', 'analytics.read', 'Просматривать аналитику')
ON CONFLICT (code) DO NOTHING;

INSERT INTO identity.role_permissions (role_id, permission_id)
SELECT roles.id, permissions.id
FROM identity.roles AS roles
JOIN identity.permissions AS permissions
    ON permissions.code IN ('comments.delete_any', 'users.block', 'analytics.read')
WHERE roles.name = 'admin'
ON CONFLICT DO NOTHING;

INSERT INTO identity.role_permissions (role_id, permission_id)
SELECT roles.id, permissions.id
FROM identity.roles AS roles
JOIN identity.permissions AS permissions
    ON permissions.code IN ('comments.delete_any', 'users.block')
WHERE roles.name = 'moderator'
ON CONFLICT DO NOTHING;

-- +goose Down

DELETE FROM identity.role_permissions
WHERE permission_id IN (
    SELECT id FROM identity.permissions
    WHERE code IN ('comments.delete_any', 'users.block', 'analytics.read')
);
DELETE FROM identity.permissions
WHERE code IN ('comments.delete_any', 'users.block', 'analytics.read');
