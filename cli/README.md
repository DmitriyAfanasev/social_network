# CLI

Команды запускаются из корня проекта после применения миграций:

```bash
python -m cli.manage_roles list-roles
python -m cli.manage_roles grant --email user@example.com --role admin
python -m cli.manage_roles revoke --email user@example.com --role admin
```

Скрипт использует настройки подключения к PostgreSQL из `.env` (`APP_CONFIG__DB__URL`),
находит пользователя по email и выполняет назначение роли в одной транзакции.
Роль `admin` и её permissions создаются миграцией RBAC.
