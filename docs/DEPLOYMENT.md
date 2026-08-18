# Deployment — Zorin Fabric 1.0

Ниже минимальный production-oriented порядок. Секреты и реальные provider cookies в репозиторий не добавляются.

## 1. Подготовить RU Remnawave/Xray Config Profile

Сохраните текущий RU profile локально как `ru-profile.json` и выполните:

```bash
python3 tools/remnawave-fabric-compile.py \
  --ru-config ./ru-profile.json \
  --xray-out ./ru-profile.fabric.json \
  --routes-out ./deploy/ru-edge/generated/fabric-routes.json \
  --inventory-out ./deploy/ru-edge/generated/fabric-inventory.json
```

Для текущей переданной топологии результат содержит:

```text
auto
auto-warp
ae-direct
fi-direct
fi-warp
ge-direct
ge-warp
nl-direct
nl-warp
```

Compiler идемпотентен: повторная компиляция удаляет ранее сгенерированные `FABRIC-VCONNECT-*` объекты и строит их заново.

### В Remnawave Panel

1. Импортируйте/перенесите `ru-profile.fabric.json` в соответствующий **Config Profile**.
2. Убедитесь, что RU Node использует этот Config Profile.
3. Включите новые `FABRIC-VCONNECT-*` inbounds для RU Node.
4. Добавьте эти inbounds в **Internal Squad**, предназначенный для Fabric users, либо в существующий squad, который должен иметь Fabric access.
5. Назначьте пользователей этому squad.

Не заполняйте `clients` вручную. Fabric compiler специально оставляет `clients: []`, чтобы Remnawave продолжал управлять Xray users.

## 2. Controller

Скопируйте:

```text
deploy/controller/fabric-controller.json
```

и замените:

- `remnawave.baseUrl`;
- список `edges`;
- id/region/priority;
- `heartbeatTokenFile` под реальные edge secret filenames.

Controller по умолчанию опубликован только на:

```text
127.0.0.1:47800
```

Выставляйте его клиентам через HTTPS reverse proxy.

### Secrets

```bash
python3 tools/init-fabric-secrets.py --edges ru-1,ru-2
```

Скрипт создаёт:

- общий Fabric auth key;
- edge heartbeat tokens;
- edge vault key.

Он **не** создаёт Remnawave API token и cookies провайдеров.

Заполните вручную:

```text
deploy/controller/secrets/remnawave-api-token.txt
deploy/ru-edge/secrets/remnawave-api-token.txt
deploy/ru-edge/secrets/cookies-yandex.json
deploy/ru-edge/secrets/cookies-vk.json
```

Для первого RU edge `edge-heartbeat-token.txt` должен совпадать с соответствующим token в Controller.

### Запуск

```bash
cd deploy/controller
docker compose up -d --build
```

Проверка:

```bash
curl http://127.0.0.1:47800/healthz
```

## 3. RU Edge

Создайте:

```bash
cd deploy/ru-edge
cp .env.example .env
```

Настройте:

```text
REMNAWAVE_URL=
FABRIC_CONTROLLER_URL=
FABRIC_EDGE_ID=ru-1
MAX_ACTIVE_USERS=64
RESOURCES=default
WORK_PLATFORM=telemost  # или vk
WORK_VK_PEER_ID=        # обязательно для VK work-call
WORK_COOKIE_SOURCE=service
```

### Что означает WORK_PLATFORM

Это **не control plane Xray** и не «режим proxy». Это провайдер рабочего звонка, внутри которого VConnect несёт пользовательский payload:

```text
WORK_PLATFORM=telemost → data/work call = Яндекс Телемост
WORK_PLATFORM=vk       → data/work call = VK Звонки
```

В 1.0 persistent service/control call для AUTO-клиентов остаётся Telemost. Из него creator может создать work call обоих типов.

### Запуск edge

```bash
docker compose up -d --build
```

Compose использует `network_mode: host`, потому что VConnect creator и Remnawave Node/Xray должны видеть одни и те же loopback-only `FABRIC-VCONNECT-*` listeners.

`edge-agent` читает текущую service-call ссылку и active-user count из общего volume и отправляет heartbeat Controller.

## 4. Проверить bootstrap

Legacy credential:

```bash
python3 tools/fabric-credential.py issue \
  --key-file deploy/controller/secrets/fabric-auth-key.txt \
  --user YOUR_REMNAWAVE_SHORT_UUID \
  --provision
```

Или short-lived token:

```bash
python3 tools/fabric-credential.py token \
  --key-file deploy/controller/secrets/fabric-auth-key.txt \
  --user YOUR_REMNAWAVE_SHORT_UUID \
  --ttl 1h
```

После этого можно проверить Controller CLI:

```bash
cd fabric-controller
go run ./cmd/fabric-bootstrap \
  --controller https://fabric.example.com \
  --user YOUR_REMNAWAVE_SHORT_UUID \
  --credential 'ft1....'
```

## 5. Client

### Android / desktop source

В настройках указываются:

```text
Join/URL: fabric://auto
Fabric Controller: https://fabric.example.com
Remnawave short UUID: ...
Fabric credential: ft1....
```

AUTO client получает от Controller живой RU edge и его актуальный Telemost service/control call. После service handshake запрашиваемая `workPlatform` определяет, где идёт **реальный payload** — VK или Telemost.

### Explicit route

По умолчанию egress policy = `auto`. При необходимости можно выбрать region/mode из опубликованного Fabric route inventory.

## 6. Добавить второй RU edge

На новом сервере:

1. использовать тот же Fabric-aware Remnawave Config Profile;
2. включить нужные Fabric inbounds;
3. развернуть `deploy/ru-edge`;
4. задать уникальный `FABRIC_EDGE_ID`;
5. создать отдельный heartbeat token;
6. добавить edge в Controller config;
7. перезапустить Controller.

Клиентское `fabric://auto` менять не нужно.

## 7. Rollback

Fabric additions изолированы тегом `FABRIC-VCONNECT-*` и отдельными AUTO balancers. Для rollback можно вернуть предыдущий Remnawave Config Profile и остановить VConnect edge/Controller; существующие обычные Xray hosts и users от этого не обязаны меняться.
