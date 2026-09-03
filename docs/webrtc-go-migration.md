# WebRTC: перенос signaling с Python на Go

В проекте WebRTC теперь разделён на две части:

```text
браузер A ──────── media (SRTP/DTLS, обычно напрямую) ──────── браузер B
    │                                                             │
    └── call WebSocket ── Go call-signaling ── Redis ────────────┘
                                  │
                                  └── PostgreSQL: права доступа

браузер ───────────── message WebSocket ───────────── Python API
```

Go-сервис находится в `services/call-signaling`. Python больше не создаёт
call-session, не принимает SDP/ICE и не публикует `call.*` события. Python
WebSocket по-прежнему нужен для сообщений, typing и read receipts.

## Почему это называется signaling

WebRTC — это набор браузерных API и протоколов для интерактивного media:

- `RTCPeerConnection` — объект соединения;
- `MediaStream` и `getUserMedia` — микрофон/камера;
- SDP — описание параметров соединения и media-дорожек;
- ICE — поиск рабочего сетевого маршрута;
- STUN — определение внешнего адреса;
- TURN — relay, если прямой маршрут невозможен;
- DTLS-SRTP — шифрование media после выбора маршрута.

Signaling не является частью стандарта WebRTC. Это прикладной канал, через
который два браузера договариваются о параметрах:

1. caller создаёт `offer` и отправляет его через Go;
2. callee устанавливает remote offer, создаёт `answer` и отправляет его;
3. обе стороны отправляют найденные ICE candidates;
4. браузеры сами выполняют ICE checks и устанавливают зашифрованный media
   канал;
5. после этого аудио/видео не проходит через Go и Python.

Поэтому в signaling-сервисе нет `Pion PeerConnection`: Pion нужен, когда Go
сам становится WebRTC endpoint (например, SFU, recorder или media gateway).
В текущем one-to-one mesh-сценарии Go является control-plane сервисом, а
браузеры — media endpoints. Это правильнее и дешевле, чем проксировать
видеопоток через backend.

## Что сделано в Go

### Аутентификация

`internal/auth/jwt.go` читает cookie `access-token`, проверяет подпись HMAC,
алгоритм, срок действия и claim `user_id`. Сервис использует тот же secret и
алгоритм, что и Python API.

### Авторизация

`internal/authz/postgres.go` проверяет:

- пользователь входит в `conversation_participants`;
- звонок относится к прямому диалогу ровно из двух пользователей;
- caller и callee действительно являются участниками этого диалога;
- нет блокировки ни в одну сторону;
- `profiles.message_policy` разрешает взаимодействие (`everyone`, `friends`,
  `friends_of_friends`).

Проверка повторяется и для accept/reject/end, и для каждого signaling-сообщения.
Нельзя украсть `call_id`, подписавшись на другой диалог.

### Call state machine

Допустимые переходы:

```text
start ──> ringing ──accept──> active ──end──> ended
                 ├─reject──> rejected
                 └──────────end─────────────> ended
```

Сессия хранится в Redis hash с ключом `calls:session:<call_id>` и TTL две
минуты. Переходы выполняются Lua-скриптом атомарно. Это защищает от гонки,
когда два экземпляра Go почти одновременно получают два `accept` или `end`.

### Масштабирование

Каждый экземпляр Go хранит только локальные WebSocket-подключения. Событие
публикуется в Redis Pub/Sub channel `calls:events`, после чего каждый экземпляр
доставляет его своим локальным клиентам. В payload есть `recipient_ids`, поэтому
событие не попадёт постороннему подключению.

## Протокол WebSocket

URL по умолчанию: `ws://localhost:8001/ws`.

Клиент сначала подписывается на диалог:

```json
{"type":"conversation.subscribe","conversation_id":10}
```

Начало звонка:

```json
{
  "type":"call.start",
  "conversation_id":10,
  "target_user_id":20,
  "call_type":"video"
}
```

Go отвечает обоим участникам событием `call.invite`. После нажатия «Принять»
callee отправляет:

```json
{"type":"call.accept","conversation_id":10,"call_id":"..."}
```

SDP и ICE пересылаются как opaque JSON — signaling-сервис не изменяет SDP:

```json
{
  "type":"call.signal",
  "conversation_id":10,
  "call_id":"...",
  "signal":{"kind":"offer","sdp":{"type":"offer","sdp":"v=0..."}}
}
```

В production следует дополнительно валидировать размер и форму SDP/candidate,
вести audit-метрики и ограничивать частоту сообщений. Ограничение размера
WebSocket уже установлено на 256 KiB.

## Локальный запуск

1. Запустить зависимости и сервис:

   ```bash
   docker compose up --build pg redis call-signaling
   ```

2. Запустить Python API и frontend обычным способом.

3. В `frontend/.env` указать:

   ```env
   VITE_API_BASE_URL=http://localhost:8000
   VITE_CALL_SIGNALING_URL=ws://localhost:8001/ws
   ```

4. Открыть два браузера под разными пользователями, создать прямой диалог и
   разрешить браузеру доступ к микрофону/камере.

## Проверка с телефона в одной Wi-Fi сети

`localhost` всегда означает «это же устройство». На телефоне адрес
`localhost` указывает на сам телефон, поэтому нужен LAN IP компьютера.

Узнайте IP на компьютере, где запущены сервисы:

```bash
# Linux
ip -4 route get 1.1.1.1 | sed -n 's/.*src \([0-9.]*\).*/\1/p'
# или
hostname -I
```

На macOS можно выполнить `ipconfig getifaddr en0`, а в Windows —
`ipconfig` и взять IPv4-адрес Wi-Fi адаптера. Обычно это адрес вида
`192.168.1.50` или `10.0.0.25`. Команду нужно запускать на самом ПК, а не
внутри Docker-контейнера: контейнер может видеть только loopback-интерфейс.

Оба устройства должны быть в одной обычной Wi-Fi сети. Guest Wi-Fi часто
включает client isolation и запрещает телефону обращаться к компьютеру.
Также разрешите входящие TCP-порты `5173`, `8000` и `8001` в firewall ПК.

Запустите зависимости:

```bash
docker compose up -d pg redis
```

Затем запустите Python API в LAN-режиме:

```bash
APP_HOST=0.0.0.0 \
FRONTEND_ALLOWED_ORIGINS=http://192.168.1.50:5173 \
uv run uvicorn backend.main:main_app --host 0.0.0.0 --port 8000
```

Замените `192.168.1.50` на IP своего компьютера. В отдельном терминале
запустите Go signaling с тем же origin:

```bash
CALL_ALLOWED_ORIGINS=http://192.168.1.50:5173 \
docker compose up --build call-signaling
```

В `frontend/.env` укажите LAN-адрес:

```env
VITE_API_BASE_URL=http://192.168.1.50:8000
VITE_CALL_SIGNALING_URL=ws://192.168.1.50:8001/ws
VITE_WEBRTC_ICE_SERVERS=[{"urls":"stun:stun.l.google.com:19302"}]
```

Запустите frontend (`npm run dev`) и на телефоне откройте
`http://192.168.1.50:5173`. На ПК откройте тот же LAN URL, войдите под одним
пользователем, на телефоне — под другим. Создайте диалог, разрешите доступ к
микрофону/камере и нажмите «Позвонить».

Если звонок не начинается — проверьте разрешения браузера, cookie, CORS/origin
и WebSocket в DevTools. Если media не соединяется, для локальной Wi-Fi сети
обычно достаточно STUN, а для мобильной сети или разных NAT потребуется TURN.

Проверка health endpoint:

```bash
curl http://localhost:8001/healthz
```

Проверка кода:

```bash
cd services/call-signaling
GOCACHE=/tmp/general-project-call-gocache \
GOMODCACHE=/tmp/general-project-call-modcache \
go test ./...
```

## STUN и TURN

STUN не передаёт media. Он помогает браузеру узнать адрес, с которого его
видит интернет. Прямой маршрут часто не работает из-за symmetric NAT,
корпоративных firewall и мобильных сетей.

TURN — это relay. Когда ICE не находит прямой маршрут, media идёт через TURN:

```text
browser A ── encrypted SRTP ── TURN ── encrypted SRTP ── browser B
```

Для production нужно поднять coturn, использовать временные credentials и
передать браузеру массив `iceServers` с `stun:` и `turn:` URL. Секрет TURN
нельзя вшивать в frontend навсегда; обычно backend выдаёт короткоживущие
credentials по авторизованному endpoint.

## Что спросить на собеседовании

**Почему WebRTC не работает без signaling?** Потому что `RTCPeerConnection`
не знает, с кем соединяться и как передать offer/answer/candidates. Способ
signaling не стандартизирован; это может быть WebSocket, HTTP или другой канал.

**Зачем STUN и TURN?** STUN помогает узнать public mapping, TURN даёт relay,
когда peer-to-peer маршрут невозможен. Надёжный production WebRTC почти всегда
требует TURN.

**Передаёт ли signaling-сервис видео?** Нет. Он передаёт маленькие control
messages. Media идёт по ICE-selected transport и шифруется DTLS-SRTP.

**Почему не отправлять видео через Kafka?** Kafka подходит для событий и
буферизации, но не для низколатентного RTP media: появятся задержка, нагрузка,
огромный объём данных и плохая реакция на packet loss.

**Что делать для группового звонка?** Не строить mesh со всеми участниками:
число исходящих потоков растёт как `N-1` на клиента. Для группы нужен SFU
(например, LiveKit, Janus или mediasoup), который принимает uplink и выбирает,
какие downlink-потоки отправить каждому участнику.

**Какие состояния мониторить?** `iceConnectionState`,
`connectionState`, `signalingState`, RTT, packet loss, jitter, bitrate,
reconnect count и длительность звонка. Эти метрики важнее самого факта
успешного `call.start`.

## Ограничения текущей версии

Это полноценный one-to-one demo/production-shaped контур, но перед публичным
запуском нужно добавить timeout/reaper для UI, busy policy, reconnect/resume,
rate limiting, TURN credentials endpoint, Prometheus metrics и интеграционные
тесты с Redis/PostgreSQL. Origin allowlist уже настраивается через
`CALL_ALLOWED_ORIGINS`.
