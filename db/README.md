# Database workflow

The local PostgreSQL database is disposable. For a clean migration run, use
the explicitly named reset command below. It stops and removes only the Compose
`pg` container and its `postgres_data` volume; Redis, MinIO, ClickHouse and the
separate `pg_test` volume remain intact.

```bash
task db:reset
docker compose up -d pg
psql "$DATABASE_URL" -f db/bootstrap.sql
goose -dir services/identity/migrations -table identity.goose_db_version postgres "$DATABASE_URL" up
goose -dir services/profiles/migrations -table profiles.goose_db_version postgres "$DATABASE_URL" up
goose -dir services/social/migrations -table social.goose_db_version postgres "$DATABASE_URL" up
goose -dir services/content/migrations -table content.goose_db_version postgres "$DATABASE_URL" up
goose -dir services/media/migrations -table media.goose_db_version postgres "$DATABASE_URL" up
goose -dir services/messaging/migrations -table messaging.goose_db_version postgres "$DATABASE_URL" up
goose -dir services/analytics/migrations -table analytics.goose_db_version postgres "$DATABASE_URL" up
goose -dir services/admin/migrations -table admin.goose_db_version postgres "$DATABASE_URL" up
```

The reset is intentionally destructive for local PostgreSQL data. Run it only
when a clean database is required.

Each service owns its migration directory and its goose version table. Domain
tables must never be added to `public`, and service migrations must not create
foreign keys to another service schema. The profiles service owns the optional
public `handle` and `display_name`; identity stores only authentication data
and user UUIDs.
