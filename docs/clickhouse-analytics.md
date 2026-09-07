# ClickHouse и аналитика

Инструкция для локального стенда General Project.

## Важное ограничение текущей реализации

Контейнер ClickHouse уже добавлен в `docker-compose.yaml`, но текущий Go
analytics-сервис пока не записывает туда события. Сейчас рабочее хранилище
аналитики — PostgreSQL, схема `analytics`:

- `analytics.event_records` — общий журнал интеграционных событий;
- `analytics.video_events` — просмотры и телеметрия видео;
- `analytics.processed_events` — дедупликация событий Kafka.

Поэтому статистику, которую собирает приложение, сейчас нужно смотреть через
PostgreSQL или HTTP API analytics. ClickHouse можно запускать для знакомства,
ручных запросов и следующего этапа миграции аналитики.

## Запуск ClickHouse

```bash
docker compose up -d clickhouse
docker compose ps clickhouse
curl http://localhost:8123/ping
```

Ожидаемый ответ последней команды — `Ok.`.

Параметры локального контейнера:

| Подключение | Значение |
|---|---|
| HTTP | `http://localhost:8123` |
| Native protocol | `localhost:9002` |
| Пользователь | `default` |
| Пароль | `password` |
| База по умолчанию | `default` |
| Volume | `clickhouse_data` |

Порт `9002` на компьютере проброшен в стандартный порт ClickHouse `9000`
внутри контейнера.

## Подключение к ClickHouse

### Через clickhouse-client внутри контейнера

```bash
docker compose exec clickhouse clickhouse-client \
  --user default \
  --password password
```

Пример без интерактивной консоли:

```bash
docker compose exec -T clickhouse clickhouse-client \
  --user default \
  --password password \
  --query 'SELECT version()'
```

### Через HTTP API

```bash
curl -u default:password \
  'http://localhost:8123/?database=default' \
  --data-binary 'SELECT version()'
```

Для удобного вывода в формате JSON:

```bash
curl -u default:password \
  'http://localhost:8123/?database=default' \
  --data-binary 'SELECT 1 AS ok FORMAT JSONEachRow'
```

## Основные команды ClickHouse

Их можно выполнять в `clickhouse-client` или передавать через `--query`.

```sql
SHOW DATABASES;
SHOW TABLES FROM default;
SELECT name, engine
FROM system.tables
WHERE database = 'default';

SELECT
    database,
    table,
    formatReadableSize(sum(bytes_on_disk)) AS disk_size,
    sum(rows) AS total_rows
FROM system.parts
WHERE active
GROUP BY database, table
ORDER BY sum(bytes_on_disk) DESC;
```

Полезные операции с базой и таблицами:

```sql
CREATE DATABASE IF NOT EXISTS analytics;
SHOW TABLES FROM analytics;
DESCRIBE TABLE analytics.video_events;
SELECT count() FROM analytics.video_events;
```

`CREATE DATABASE` и `CREATE TABLE` нужны только для будущего ClickHouse
read-model или ручного эксперимента. Они не подключают ClickHouse к Go-сервису
автоматически.

## Метрики текущего приложения через PostgreSQL

Подключение:

```bash
psql "postgres://admin:password@localhost:5432/database?sslmode=disable"
```

Количество событий по типам:

```sql
SELECT event_type, count(*) AS events
FROM analytics.event_records
GROUP BY event_type
ORDER BY events DESC;
```

Общая статистика по просмотрам видео:

```sql
SELECT
    count(*) AS views,
    count(DISTINCT COALESCE(user_id::text, 'session:' || NULLIF(session_id, ''))) AS unique_viewers,
    round(sum(watch_seconds)::numeric, 2) AS total_watch_seconds,
    round(avg(watch_seconds)::numeric, 2) AS average_watch_seconds,
    count(*) FILTER (WHERE completed) AS completed_views
FROM analytics.video_events
WHERE event_type = 'media.video.viewed';
```

Самые просматриваемые видео:

```sql
SELECT
    video_id,
    count(*) AS views,
    count(DISTINCT COALESCE(user_id::text, 'session:' || NULLIF(session_id, ''))) AS unique_viewers,
    round(sum(watch_seconds)::numeric, 2) AS watch_seconds,
    count(*) FILTER (WHERE completed) AS completed_views
FROM analytics.video_events
WHERE event_type = 'media.video.viewed'
GROUP BY video_id
ORDER BY views DESC, watch_seconds DESC
LIMIT 20;
```

Просмотры по дням:

```sql
SELECT
    date_trunc('day', occurred_at) AS day,
    count(*) AS views,
    count(DISTINCT COALESCE(user_id::text, 'session:' || NULLIF(session_id, ''))) AS unique_viewers
FROM analytics.video_events
WHERE event_type = 'media.video.viewed'
GROUP BY day
ORDER BY day DESC;
```

Удержание видео по каждому ролику. `completion_rate` — доля просмотров, в
которых клиент прислал `completed = true`:

```sql
SELECT
    video_id,
    count(*) AS views,
    round(avg(
        CASE
            WHEN duration_seconds > 0
            THEN least(watch_seconds / duration_seconds, 1)
            ELSE 0
        END
    )::numeric, 3) AS average_watch_ratio,
    round((count(*) FILTER (WHERE completed)::numeric / count(*)), 3) AS completion_rate
FROM analytics.video_events
WHERE event_type = 'media.video.viewed'
GROUP BY video_id
ORDER BY views DESC;
```

Проверка задержки обработки событий:

```sql
SELECT
    records.event_type,
    count(*) AS events,
    round(avg(EXTRACT(EPOCH FROM (processed.occurred_at - records.created_at)))::numeric, 3) AS average_seconds
FROM analytics.event_records AS records
JOIN analytics.video_events AS processed ON processed.event_id = records.event_id
GROUP BY records.event_type
ORDER BY events DESC;
```

Проверка очереди и ошибок analytics:

```sql
SELECT count(*) AS pending_events
FROM analytics.outbox_events
WHERE published_at IS NULL;

SELECT event_id, status, attempts, claimed_at, processed_at
FROM analytics.processed_events
ORDER BY claimed_at DESC
LIMIT 20;
```

Все текущие таблицы аналитики принадлежат PostgreSQL. ClickHouse-запросы к
этим таблицам напрямую не выполняются.

## Метрики через HTTP API

Analytics HTTP API работает через gateway на `http://localhost:8000` и требует
access-токен.

Статистика конкретного видео:

```bash
VIDEO_ID="полный-uuid-видео"
ACCESS_TOKEN="ваш-access-token"

curl -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  "http://localhost:8000/v1/analytics/videos/${VIDEO_ID}"
```

Ответ содержит:

- `views` — количество событий фиксации просмотра;
- `unique_viewers` — уникальные авторизованные пользователи и анонимные
  сессии;
- `total_watch_seconds` и `average_watch_seconds` — время просмотра;
- `completed_views` — завершённые просмотры;
- `last_viewed_at` — время последнего события.

История видео текущего пользователя:

```bash
curl -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  "http://localhost:8000/v1/analytics/me/videos?limit=20"
```

Эти endpoint’ы читают PostgreSQL read-model analytics, а не ClickHouse.

## Если нужны именно ClickHouse-метрики

Для ручного теста можно создать отдельную таблицу. Это пример схемы, а не ещё
подключённый production-пайплайн:

```sql
CREATE DATABASE IF NOT EXISTS analytics;

CREATE TABLE IF NOT EXISTS analytics.video_events (
    event_id UUID,
    event_type LowCardinality(String),
    video_id UUID,
    user_id Nullable(UUID),
    session_id Nullable(String),
    watch_seconds Float64 DEFAULT 0,
    progress_seconds Float64 DEFAULT 0,
    duration_seconds Float64 DEFAULT 0,
    completed Bool DEFAULT false,
    occurred_at DateTime64(3, 'UTC')
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(occurred_at)
ORDER BY (video_id, occurred_at, event_id);
```

После этого пример аналитического запроса будет выглядеть так:

```sql
SELECT
    video_id,
    countIf(event_type = 'media.video.viewed') AS views,
    uniqExactIf(user_id, event_type = 'media.video.viewed') AS registered_viewers,
    sumIf(watch_seconds, event_type = 'media.video.viewed') AS watch_seconds,
    countIf(event_type = 'media.video.viewed' AND completed) AS completed_views
FROM analytics.video_events
GROUP BY video_id
ORDER BY views DESC
LIMIT 20;
```

Для реальной интеграции понадобится отдельный consumer/поток доставки
`application.events → ClickHouse`, дедупликация по `event_id` и политика
повторной доставки. Пока такой поток не включён, не следует считать пустую
ClickHouse-таблицу признаком отсутствия просмотров — проверяйте PostgreSQL.

## Диагностика

```bash
docker compose logs -f clickhouse
docker compose logs --tail=200 analytics
docker compose exec clickhouse clickhouse-client \
  --user default --password password --query 'SELECT 1'
```

Проверить Kafka и analytics consumer:

```bash
docker compose exec kafka /opt/kafka/bin/kafka-consumer-groups.sh \
  --bootstrap-server kafka:29092 \
  --describe --group analytics-read-model
```

Если у ClickHouse отвечает `/ping`, но статистика пустая, сначала проверьте,
есть ли строки в `analytics.video_events` PostgreSQL и запущен ли контейнер
`analytics`.

## Остановка

```bash
docker compose stop clickhouse
```

Остановка не удаляет volume `clickhouse_data`. Не используйте `docker compose
down -v`, если нужно сохранить локальные данные: эта команда удалит volumes
Compose.
