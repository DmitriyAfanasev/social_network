# OpenAPI contracts

Здесь хранятся исходные OpenAPI-контракты HTTP-сервисов. Каждый сервис владеет
своей спецификацией и генерирует из неё только DTO transport-слоя; domain и
application DTO остаются рукописными.

После изменения спецификации выполните `task generate:openapi`. Команда
обновит соответствующий `services/<service>/internal/transport/http/openapi.gen.go`;
этот файл должен быть закоммичен вместе со спецификацией. `task check:openapi`
проверяет, что генерация не оставляет diff.
