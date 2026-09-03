# Эксплуатация, наблюдаемость и больные места

## Минимальные SLI/SLO для учебного стенда

| Область | SLI | Цель |
|---|---|---|
| API | доля 2xx/4xx без 5xx | ≥ 99% локального времени |
| API | p95 latency | < 500 ms для простых GET |
| Outbox | age самого старого pending event | < 60 секунд |
| Kafka | consumer lag analytics | стремится к 0 |
| Analytics | `analytics_events_failed_total` | 0 в штатном режиме |
| Analytics | `analytics_dlq_events_total` | 0 в штатном режиме |
| Analytics | задержка event → ClickHouse | < 2 минут |
| Realtime | время до SSE клиента | < 3 секунд при активном подключении |

SLO здесь учебные: они нужны для практики измерения, а не являются обещанием production availability.

## Логи и correlation

Каждый HTTP request и event должны иметь `request_id`/`event_id`. В логах нужны как минимум:

- service name и version;
- timestamp в UTC;
- event type и event id;
- Kafka topic, partition и offset для consumer;
- duration и результат операции;
- exception class без секретов и персональных данных.

## Основные failure modes

### PostgreSQL недоступен

API должен вернуть контролируемую ошибку/503, а не необработанный traceback. Readiness не должен быть зелёным, если приложение обязательно требует БД для работы.

### Kafka недоступна

HTTP mutation не должна silently терять событие: committed outbox остаётся pending и будет повторён. Worker должен логировать retry и возраст outbox.

### ClickHouse недоступен

Analytics consumer делает ограниченный exponential retry. Если событие не удалось обработать после лимита, оно публикуется в `analytics.dlq` и подтверждается, чтобы poison-событие не блокировало partition. После восстановления события из DLQ нужно re-drive-нуть отдельной операционной процедурой. Вставка должна быть идемпотентной по `event_id`.

### Kafka healthcheck

Healthcheck Kafka — это не отдельный сервис Kafka, а активная проверка готовности брокера. `kafka-topics.sh --list` проверяет network listener, metadata и обработку admin-запроса. Проверки контейнерного процесса недостаточно: JVM может быть запущена, а broker ещё не принимать подключения.

### Consumer упал после side effect

Повторная доставка обязательна. Email, удаление файла и запись analytics должны иметь idempotency strategy: dedup key, проверка существования или transactional marker.

### Redis перезапущен

SSE/WebSocket клиенты переподключаются. Redis Pub/Sub не является очередью: пропущенные realtime-события восстанавливаются только через повторную загрузку состояния из API.

### Kafka UI недоступен

Это диагностический инструмент, а не runtime dependency. API и consumers не должны зависеть от доступности UI.

## Security checklist

- Secrets только через Kubernetes Secret/локальный secret manager.
- GHCR token с минимальными правами `read:packages` для pull.
- Kafka, PostgreSQL, Redis и ClickHouse не публиковать наружу в production.
- NetworkPolicy: frontend → API, API → DB/Redis/S3, workers → Kafka/DB, analytics → Kafka/ClickHouse.
- Не логировать access token, password, cookie и полный email payload.

## Диагностика релиза

```bash
kubectl get pods -n general
kubectl describe pod <pod> -n general
kubectl logs deploy/api -n general --tail=200
kubectl logs deploy/analytics -n general --tail=200
kubectl get events -n general --sort-by=.lastTimestamp
```
