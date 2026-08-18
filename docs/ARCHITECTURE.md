# Zorin Fabric 1.0 — архитектура

## 1. Границы ответственности

Zorin Fabric не заменяет существующие проекты одним монолитом.

| Слой | Компонент | Ответственность |
|---|---|---|
| Access | VConnect | Доставить трафик клиента до доступного RU edge внутри VK Звонков / Яндекс Телемоста |
| Transit / Internet | Xray | Маршрутизация RU→EU, DIRECT/WARP, протоколы и egress |
| Fleet / users | Remnawave | Пользователи, ноды, Config Profiles, inbounds, lifecycle |
| Access control | Fabric Controller | Multi-RU bootstrap, heartbeat, admission, short-lived credentials |
| Overlay | KnotRoute | `.knot`, внутренний DNS, private services, peer/service overlay |
| Assurance | Zorin Trust | Hardware-backed owner/device identity, approval, выдача short-lived Fabric session |

## 2. Data plane

```text
                                     CONTROL ONLY
                           ┌───────────────────────────┐
                           │      Fabric Controller    │
                           │ Remnawave auth + RU state │
                           └─────────────┬─────────────┘
                                         │ bootstrap
                                         │
Client ──────────────────────────────────┘
  │
  │ VConnect service/control call
  │ (1.0 client AUTO: Telemost)
  ▼
RU Edge service creator
  │
  │ creates work call for this session
  │ ┌───────────────┐
  ├─┤ Telemost work │
  │ └───────────────┘
  │ or
  │ ┌───────────────┐
  └─┤ VK work call  │
    └───────┬───────┘
            │ useful traffic
            ▼
      VConnect creator
            │
            │ local ephemeral SOCKS
            ▼
  Per-user Xray bridge
            │
            │ VLESS/raw/none over 127.0.0.1
            │ user.id = Remnawave vlessUuid
            ▼
  FABRIC-VCONNECT-* inbound
  main Remnawave-managed RU Xray
            │
            ├─ private/RU → existing DIRECT
            │
            └─ external → existing BL-* balancer
                              │
                          HO-* pool
                              │
                       EU / WARP / Internet
```

Критическая граница: Xray credentials/REALITY details/HO-* topology никогда не должны становиться частью VConnect client configuration.

## 3. Почему per-user bridge

Простой VConnect→SOCKS→Xray handoff теряет Remnawave identity. 1.0 создаёт на каждую активную work-session маленький Xray process:

```text
VConnect user shortUuid
      │
      ├─ Remnawave lookup → ACTIVE + vlessUuid
      │
      ▼
per-session bridge
      │ outbound VLESS user.id = vlessUuid
      ▼
main RU Xray Fabric VLESS inbound
```

Внутренние bridge listeners привязаны к `127.0.0.1`, конфиги создаются во временном каталоге с ограниченными правами и удаляются вместе с session lease.

Основной Xray остаётся единственным владельцем RU→EU routing. Sidecar ничего не знает о REALITY/XHTTP/WARP/европейских адресах — он знает только локальные Fabric VLESS ingress'ы.

## 4. Два независимых автоматических выбора

### Access selection

Controller сортирует RU edge:

1. online/stale heartbeat;
2. configured priority;
3. текущая load.

Это отвечает только на вопрос «через какой RU edge войти».

### Transit selection

После RU edge Xray использует существующие балансы/observatory. Для текущей топологии compiler создаёт `auto` и `auto-warp` поверх уже существующих региональных `HO-*` selectors и сохраняет явные FI/NL/GE/AE policies.

Это отвечает на отдельный вопрос «каким RU→EU маршрутом отправить внешний трафик».

Не надо объединять эти алгоритмы в один routing engine: деградация VK/Telemost access и деградация RU→EU transit имеют разные причины и telemetry.

## 5. Remnawave ownership

`tools/remnawave-fabric-compile.py` берёт текущий RU Config Profile и добавляет:

- loopback VLESS inbounds `FABRIC-VCONNECT-*`;
- routing rules на уже существующие `BL-*`;
- AUTO balancers;
- user stats policy;
- secret-free route inventory для VConnect Edge.

Он **не** записывает пользователей в `clients`. Эти массивы остаются пустыми в source Config Profile и должны наполняться обычным Remnawave lifecycle после включения Fabric inbounds для соответствующих пользователей/squad.

## 6. Controller

Controller не держит пользовательские payload-потоки.

Он предоставляет:

- `GET /healthz`;
- `GET /v1/bootstrap` — authenticated список RU edge;
- `POST /v1/edges/{id}/heartbeat` — service calls + load/capacity;
- `POST /v1/trust/exchange` — one-shot Trust proof → short-lived Fabric token.

При bootstrap одновременно проверяются Fabric credential и текущий Remnawave user status.

## 7. KnotRoute plane

KnotRoute остаётся самостоятельным E2E overlay:

```text
ordinary Internet
    → VConnect → RU Xray → EU

*.knot / private services
    → KnotRoute overlay
       → transport.Manager
          → direct and/or local Fabric/VConnect SOCKS
```

Таким образом `.knot` DNS, circuits, hidden/user services и публикация локальных сервисов не зависят от того, каким carrier физически достигнут peer.

## 8. Zorin Trust plane

Trust не становится универсальным сетевым private key. Он выдаёт assurance:

```text
hardware phone key
  → ZOWNER/1 proof
  → action = fabric.session.issue
  → resource = fabric:user:<shortUuid>
  → Controller verifies trusted phone + binding + nonce
  → ft1 short-lived credential
```

Remnawave/Xray/KnotRoute identities могут оставаться отдельными ключами и связываться attestations/capabilities.

## 9. Failure model

- EU route умер → Xray выбирает другой наблюдаемый HO-* route.
- RU edge умер → heartbeat становится stale; Controller перестаёт ставить его первым/online.
- Work call умер → VConnect session reconnect создаёт новый work call.
- Controller временно недоступен после established session → data plane не обязан падать, потому что Controller в нём не участвует.
- Remnawave API недоступен → новые неизвестные users fail closed; edge authorizer может использовать только ограниченный stale cache для ранее подтверждённых пользователей.
