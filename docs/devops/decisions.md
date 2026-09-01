# DevOps ADR: почему решения выглядят именно так

## ADR-001: Kafka для событий, Redis для realtime

Kafka хранит поток и позволяет повторно читать события consumer group. Redis Pub/Sub проще для немедленной доставки SSE/WebSocket, но не хранит историю. Объединять их в один транспорт можно, но это ухудшает ясность границ и жизненный цикл сообщений.

## ADR-002: отдельный analytics service

ClickHouse не участвует в пользовательской транзакции. Отдельный consumer позволяет независимо масштабировать аналитику, ловить lag и перезапускать её без остановки API.

Цена: отдельный image, deployment, consumer group, логи и monitoring.

## ADR-003: Envoy Gateway как production-like target

Envoy Gateway сложнее ingress-nginx для первого запуска, но лучше показывает Gateway API, traffic policy, retries, timeouts и canary routing. Поэтому ingress-nginx допустим как bootstrap-профиль, а Envoy Gateway — целевой overlay.

## ADR-004: Kubernetes manifests в этом репозитории на первом этапе

Рядом с кодом проще видеть связь image tag, config и deployment. Когда появятся несколько окружений и approval workflow, манифесты следует вынести в GitOps-репозиторий и применять через Argo CD/Flux.

## ADR-005: Kustomize до Helm

Для одного локального стенда Kustomize проще и прозрачнее. Helm появится, когда понадобятся values для dev/stage/prod, повторное использование chart и параметризация большого числа ресурсов.

## ADR-006: at-least-once для analytics

At-most-once уменьшает дубли, но теряет данные при ошибке ClickHouse. Для аналитики обычно важнее полнота, поэтому выбирается at-least-once + deduplication по `event_id`. Если нужна строгая at-most-once семантика, это должно быть осознанным продуктовым решением с допустимой потерей данных.
