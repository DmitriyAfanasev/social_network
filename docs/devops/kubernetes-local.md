# Локальный Kubernetes в Minikube

## Цель

Локальная установка должна показывать связи production-системы:

- отдельные Deployments для `frontend`, `api`, `outbox-worker` и `analytics`;
- Services для service-to-service DNS;
- StatefulSets/PersistentVolumes для PostgreSQL, Kafka, Redis, ClickHouse и MinIO;
- ConfigMap для несекретной конфигурации;
- Secret для credentials и `imagePullSecrets`;
- Ingress или Envoy Gateway для входящего HTTP-трафика;
- readiness/liveness/startup probes и resource requests/limits.

Рекомендуемый каталог манифестов:

```text
deploy/k8s/
  namespace.yaml
  configmap.yaml
  secrets.example.yaml
  api-deployment.yaml
  frontend-deployment.yaml
  outbox-worker-deployment.yaml
  analytics-deployment.yaml
  services.yaml
  ingress.yaml
  statefulsets/
  kustomization.yaml
```

## Локальный домен

Покупать домен не требуется:

```bash
minikube start
minikube addons enable ingress
minikube ip
sudo sh -c 'echo "<MINIKUBE_IP> my-site.ru" >> /etc/hosts'
```

После этого HTTP-маршрут будет доступен как `http://my-site.ru`.

## Ingress и Envoy

`ingress-nginx` проще для первого стенда: он включается addon-командой и использует знакомый ресурс `Ingress`. `Envoy Gateway` ближе к современной production-модели Gateway API и полезен для изучения маршрутов, timeout и retries, но добавляет GatewayClass/Gateway/HTTPRoute и отдельный controller.

Рекомендуемый путь:

1. первый рабочий стенд — ingress-nginx;
2. затем параллельный профиль `envoy-gateway` через Kustomize overlay;
3. сравнить поведение и оставить Envoy Gateway как целевой вариант, если нужны его политики.

## Images и GHCR

Kubernetes не следит за GitHub Container Registry автоматически. Надёжный поток:

```text
GitHub Actions
  → тесты
  → build api/frontend/worker images
  → push ghcr.io/org/project:<git-sha>
  → обновление image tag
  → kubectl apply / Argo CD sync
```

Для приватных образов нужен Kubernetes Secret типа `docker-registry`. В манифестах нельзя использовать пароль GHCR напрямую.

## Локальный запуск Kafka

Автоматическое создание topic отключено, чтобы опечатка в имени события не создавала новый поток незаметно:

```bash
docker compose up -d kafka
./scripts/create-kafka-topics.sh
./start-events.sh
./start-analytics.sh
```

В Minikube этот шаг станет Kubernetes Job или отдельным init-процессом, который выполняется после readiness Kafka.

## Что обязательно настроить

- `imagePullPolicy: IfNotPresent` для immutable SHA-tags;
- `readinessProbe`, чтобы Service не отправлял трафик неготовому Pod;
- `livenessProbe` только для восстановления зависшего процесса;
- `startupProbe` для медленного старта Kafka/ClickHouse;
- `terminationGracePeriodSeconds` для корректного закрытия consumers;
- `PodDisruptionBudget` и anti-affinity в production, но не обязательно в single-node Minikube;
- PVC и backup policy для stateful-компонентов.
