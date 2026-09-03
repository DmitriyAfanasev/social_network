# WebRTC-звонки

One-to-one signaling вынесен в Go-сервис `services/call-signaling`; Python
WebSocket отвечает только за сообщения. Полный разбор WebRTC, протокола,
авторизации, запуска и вопросов для собеседования находится в
[`docs/webrtc-go-migration.md`](docs/webrtc-go-migration.md).

Локальный запуск:

```bash
docker compose up --build pg redis call-signaling
```

Для проверки с телефона в той же Wi-Fi сети используйте LAN IP компьютера,
например `http://192.168.1.50:5173`, а не `localhost`. Полная инструкция,
включая CORS, firewall и HTTP-cookie для локального теста:
[`docs/webrtc-go-migration.md`](docs/webrtc-go-migration.md#проверка-с-телефона-в-одной-wi-fi-сети).
