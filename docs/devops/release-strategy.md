# Релизы и политики развёртывания

## Базовый релизный pipeline

1. Pull Request: unit, lint, type-check, contract и integration tests.
2. Merge в main: собрать immutable images с тегом commit SHA.
3. Push в GHCR: API, frontend, outbox-worker, analytics.
4. Migration job: применить backwards-compatible миграции.
5. Deploy: обновить только нужные image tags.
6. Rollout status и smoke-тест через публичный URL.
7. Записать версию и результат в release summary.

`latest` не используется для релиза: он делает rollback неоднозначным и ломает воспроизводимость.

## Rolling update

Рекомендуемая политика по умолчанию для stateless API/frontend:

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxUnavailable: 0
    maxSurge: 1
```

Плюсы: почти нет downtime, проста в Kubernetes, хорошо подходит для backward-compatible API.

Минусы: некоторое время работают две версии, поэтому схема БД и API должны быть совместимыми; WebSocket/SSE-соединения могут быть закрыты при удалении старого Pod.

## Blue/Green

Два набора Deployment (`blue` и `green`) существуют одновременно, а Service переключается после smoke-теста.

Плюсы: быстрый rollback и чистая проверка новой версии.

Минусы: почти двойное потребление ресурсов, отдельная логика переключения трафика и сложнее миграции БД.

## Canary

Новая версия получает небольшой процент трафика, после чего процент увеличивается по метрикам.

Плюсы: меньший blast radius, реальная проверка под нагрузкой.

Минусы: нужен traffic splitting через Envoy Gateway/service mesh и хорошие SLI; локальный single-node Minikube не показывает преимущества полноценно.

## Rollback

```bash
kubectl rollout status deployment/api -n general
kubectl rollout history deployment/api -n general
kubectl rollout undo deployment/api -n general
```

Rollback приложения не равен rollback БД. Миграции должны быть expand/contract:

```text
expand: добавить совместимую структуру
deploy: новая версия начинает её использовать
contract: удалить старую структуру отдельным поздним релизом
```

## Kafka и миграции событий

Схема event payload должна быть обратно совместимой. Переименование поля делается через период dual-read/dual-write, а не резким изменением. Consumer сначала должен переживать старый payload, затем producer переводится на новый, и только после этого удаляется legacy-поле.

## Delivery policy

- domain → outbox: атомарно;
- outbox → Kafka: at-least-once, `acks=all`, idempotent producer;
- consumer side effects: at-least-once и идемпотентность по `event_id`;
- SSE/Redis: best-effort realtime delivery, без исторической гарантии;
- analytics: at-least-once с дедупликацией, иначе точность отчётов деградирует.
- topic creation: управляемый migration/job step, auto-create отключён.
