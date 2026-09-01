# Agent Instructions

These instructions apply to the whole repository.

## General Workflow

- Read the relevant code before changing it. Prefer existing architecture, naming, and helper APIs.
- Keep edits scoped to the user request. Do not refactor unrelated code while fixing or testing a feature.
- Do not revert user changes. If the worktree is dirty, work with the existing changes.
- Prefer `rg`/`rg --files` for searching.
- Use `apply_patch` for manual file edits.
- Do not add generated, temporary, or machine-local files to the repository.

## Python Style

- Use native generic type hints: `dict[str, Any]`, `list[int]`, `set[str]`, `tuple[str, ...]`.
- Do not introduce old `typing.Dict`, `typing.List`, `typing.Set`, `typing.Tuple` in new code.
- Add explicit return types for functions and methods.
- If a dictionary has a stable non-trivial shape, use `TypedDict`, a dataclass, or a Pydantic model instead of `dict[str, Any]`.
- Use `Protocol` for structural interfaces when it makes tests or application ports clearer.
- Keep `Any` local and intentional. Do not let it leak through public application/domain APIs.
- Never type application/domain ports, commands, DTOs, results, use case inputs, or use case outputs as `Any`.
- If an application result wraps a user, post, comment, or another known concept, use the domain entity or an explicit read model/`Protocol`; do not write fields like `user: Any`, `post: Any`, or `items: Sequence[Any]`.
- Do not use `object` in type annotations. Use a precise union, `Protocol`, `TypedDict`, dataclass, Pydantic model, or a local intentional `Any` when the value is genuinely dynamic.
- Add docstrings for public use cases, services, repositories, domain policies, and complex test helpers.
- Avoid comments that repeat the code. Add comments only for non-obvious behavior or important constraints.

## Application Naming

- Treat `*UseCase` and `*Interactor` as synonyms for one application scenario. One class should orchestrate one actor-visible action.
- Prefer one canonical suffix across the codebase for scenario classes. Default to `*UseCase`; use `*Interactor` only in legacy modules that already follow that style.
- Use `*DomainService` for pure domain logic that does not fit an entity/value object and does not perform I/O.
- Avoid generic `*Service` for infrastructure concerns. Use explicit names such as `*Repository`, `*Gateway`, `*Client`, `*Provider`, or `*Storage`.
- If a class owns transaction boundaries, permissions, and orchestration of multiple ports, it belongs to the application layer and should be a `*UseCase`/`*Interactor`.
- Keep names action-oriented for scenarios (`CreateTripUseCase`, `ApprovePaymentUseCase`) and noun-oriented for infrastructure adapters (`TripRepository`, `PaymentGateway`).

## Architecture and Design Quality

- Соблюдай SOLID: один класс и один use case должны иметь одну причину для изменения; зависимости направляй через порты/протоколы; application/domain слои не должны зависеть от FastAPI, Kafka, Redis, ClickHouse или SQLAlchemy.
- Соблюдай DRY: не дублируй бизнес-правила, форматы событий, сериализацию и обработку ошибок; выноси повторяемую логику в именованные policy, factory, adapter или helper.
- Соблюдай KISS: выбирай самое простое решение, достаточное для требований; не добавляй абстракции, конфигурацию и паттерны без конкретной пользы.
- Используй паттерны GoF осознанно и документируй нетривиальное применение: Strategy для взаимозаменяемых алгоритмов, Factory/Abstract Factory для создания инфраструктурных адаптеров, Adapter для внешних систем, Observer/Pub-Sub для событий, Command для actor-visible use cases, Facade для сложных подсистем и Template Method только при действительно общем алгоритме.
- Не используй паттерны ради названий: композиция предпочтительнее наследования, глобальное состояние и Singleton запрещены без обоснования, service locator и скрытые зависимости запрещены.
- Для событий применяй outbox: бизнес-транзакция и запись события должны быть атомарны; publisher и consumers должны быть идемпотентными и безопасными для повторной доставки.
- Разделяй процессы по ответственности: HTTP API, realtime delivery, outbox publisher и analytics consumers не должны быть связаны общей mutable-состоянием.

## Testing Pyramid

- Сохраняй пирамиду тестирования: много быстрых unit-тестов чистой бизнес-логики, меньше integration-тестов адаптеров/БД/Redis/Kafka, минимум end-to-end тестов критических пользовательских сценариев.
- Каждый новый use case и policy покрывай unit-тестами с фейковыми портами; внешние системы проверяй контрактными/integration-тестами, а не моками их внутренностей.
- Для Kafka/ClickHouse/Redis проверяй схемы сообщений, retry/idempotency и wiring отдельными integration/contract-тестами; тесты не должны требовать продакшен-сервисы без явной pytest-маркировки.
- Не удаляй и не ослабляй существующие тесты ради прохождения новой реализации; при изменении контракта обновляй тест и добавляй regression case.
