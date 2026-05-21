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
