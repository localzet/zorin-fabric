# RU VConnect Edge — 1.0

Этот compose — access terminator Zorin Fabric.

```text
client
  → VK/Telemost work call
  → vconnect-edge
  → per-user Xray bridge
  → 127.0.0.1:FABRIC-VCONNECT-*
  → main Remnawave Xray
  → existing RU→EU routes
```

## Before start

1. Скомпилируйте и разверните Fabric-aware Remnawave Config Profile по `../remnawave/README.md`.
2. Убедитесь, что `generated/fabric-routes.json` соответствует profile на этом host.
3. Создайте `.env` из `.env.example`.
4. Создайте secret files по `secrets/README.md`.
5. Заполните Yandex cookies; для VK work-call заполните VK cookies + `WORK_VK_PEER_ID`.

## Host networking

`vconnect-edge` работает с `network_mode: host`. Это обязательно для текущей схемы, потому что main Remnawave Xray Fabric inbounds привязаны только к `127.0.0.1`, а per-user sidecar должен подключаться к этому же host loopback.

## Service call vs work call

Persistent service/control call используется только для admission и создания короткоживущего рабочего звонка.

`WORK_PLATFORM=telemost|vk` определяет **work/data call**, внутри которого идёт полезный трафик.

## Multi-user

На каждого активного user/session:

- проверяется Fabric credential + Remnawave status;
- разрешается `vlessUuid`;
- создаётся временный Xray sidecar;
- sidecar получает отдельные local SOCKS listeners для Fabric route choices;
- VConnect work creator использует эти listeners;
- при завершении session sidecar и его временный config удаляются.

## Edge agent

`edge-agent` публикует Controller:

- текущую Telemost service-call ссылку;
- active users;
- configured capacity;
- load;
- software version.

Если heartbeat устарел, Controller перестаёт считать edge online.
