# Целевая production-like архитектура

## Потоки

```text
Browser
  ↓ HTTP
Envoy Gateway / Ingress
  ├── frontend Service
  └── api Service
        ├── PostgreSQL
        ├── Redis (WebSocket/SSE)
        ├── MinIO/S3 (media)
        └── Outbox table
                ↓
             Kafka
              ├── analytics Service → ClickHouse
              ├── email/file-effects worker
              └── другие consumers
```

## Сервисы и ответственность

| Сервис | Цель | Состояние | Больные места |
|---|---|---|---|
| `frontend` | UI и клиентские SSE/WebSocket-подключения | нет | кеш браузера, совместимость API |
| `api` | HTTP API, auth, бизнес-сценарии | нет | миграции, connection pool, latency внешних систем |
| `outbox-worker` | переносит committed события из PostgreSQL в Kafka | offset/outbox в БД | дубли после crash между publish и mark-published |
| `analytics` | Kafka → ClickHouse | consumer offsets, ClickHouse | дедупликация и backpressure |
| `Kafka` | durable event stream | persistent log | partition ordering, retention, rebalance |
| `Redis` | realtime fan-out для WebSocket/SSE | ephemeral | потеря события при отключённом клиенте |
| `PostgreSQL` | source of truth для домена и outbox | persistent volume | backup, migrations, locks |
| `ClickHouse` | аналитическое хранилище | persistent volume | eventual consistency, duplicate inserts |
| `MinIO` | медиафайлы | persistent volume | backup и orphan objects |

Analytics не должен быть dependency для HTTP-запроса: недоступный ClickHouse не должен ломать создание поста или отправку сообщения. Для аналитики допустима eventual consistency.

## Паттерны

- **Outbox** — бизнес-транзакция и событие фиксируются атомарно в PostgreSQL.
- **Publisher/Consumer** — транспорт Kafka изолирован адаптерами.
- **Pub/Sub** — realtime fan-out через Redis.
- **Adapter** — ClickHouse, Redis, S3 и Kafka не проникают в application/domain API.
- **Strategy** — политики retry/delivery могут меняться независимо от use case.
- **Bulkhead** — analytics и side effects работают отдельными процессами и consumer groups.

## Границы владения данными

PostgreSQL — каноническое состояние пользователей, постов, комментариев, дружбы и outbox. Kafka — транспорт и временная история событий с retention. ClickHouse — производная аналитическая проекция, которую можно перестроить из событий или backfill-процесса.
