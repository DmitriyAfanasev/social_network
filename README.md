# WebRTC-звонки

One-to-one signaling вынесен в Go-сервис `services/call-signaling`, а сообщения
обслуживает `services/messaging`. Полный разбор WebRTC, протокола,
авторизации, запуска и вопросов для собеседования находится в
[`docs/webrtc-go-migration.md`](docs/webrtc-go-migration.md).

Локальный запуск:

```bash
docker compose up -d pg
task db:bootstrap
task db:migrate:all
docker compose up --build
```

Единый Swagger UI gateway доступен по адресу
[`http://localhost:8000/swagger/`](http://localhost:8000/swagger/). Исходные
спецификации находятся по адресам `/swagger.json` и `/swagger.yaml`. После
изменения HTTP-аннотаций обновите их командой `task generate:swagger`.

Для проверки с телефона в той же Wi-Fi сети используйте LAN IP компьютера,
например `http://192.168.1.50:5173`, а не `localhost`. Полная инструкция,
включая CORS и firewall для локального теста:
[`docs/webrtc-go-migration.md`](docs/webrtc-go-migration.md#проверка-с-телефона-в-одной-wi-fi-сети).
