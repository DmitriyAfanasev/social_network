# DevOps-поле проекта

## Зачем эта папка

Проект развивается как production-like учебная система. Цель — видеть не только код бизнес-логики, но и полный жизненный цикл сервиса:

```text
commit → CI → image → GHCR → deploy → rollout → metrics/logs → rollback
```

Документы описывают целевую эксплуатационную модель, а не только локальный happy path. Там, где Minikube требует упрощения, это отмечено явно.

## Документы

- [Целевая архитектура и границы сервисов](./architecture.md)
- [Локальный Kubernetes/Minikube](./kubernetes-local.md)
- [Релизы, rollout и rollback](./release-strategy.md)
- [Эксплуатация, наблюдаемость и больные места](./operations.md)
- [ClickHouse и аналитика](../clickhouse-analytics.md)
- [DevOps ADR и компромиссы](./decisions.md)

## Что считается production-like

- процессы запускаются независимо и имеют отдельные health/readiness проверки;
- состояние хранится во внешних persistent volumes, а не в контейнерном filesystem;
- образы immutable и идентифицируются commit SHA, а не `latest`;
- миграции БД выполняются отдельным контролируемым шагом;
- Kafka consumers имеют явные consumer groups и retry/DLQ-политику;
- релиз наблюдаем, обратим и проверяется smoke-тестом;
- секреты не попадают в Git;
- для каждого важного отказа есть понятный способ диагностики.

## Осознанные упрощения учебного стенда

Один Minikube не является production-кластером: control plane и worker могут оказаться на одной машине, Kafka/PostgreSQL/ClickHouse будут single-node, а доступ к локальному домену будет настроен через `/etc/hosts`. Это удобно для изучения связей, но не даёт production SLA или настоящей отказоустойчивости.
