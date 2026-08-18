# Security model

## Trust boundaries

### Client ↔ VConnect service

Fabric credential подтверждает право Remnawave user открыть VConnect work-session. `shortUuid` сам по себе credential не является.

### VConnect Edge ↔ Remnawave

Перед созданием session edge проверяет текущий account status и получает `vlessUuid`. Unknown/invalid users не получают work session.

### VConnect Edge ↔ main Xray

Per-user sidecar подключается только к loopback `FABRIC-VCONNECT-*` VLESS inbounds и предъявляет `vlessUuid` пользователя. Внешние REALITY/XHTTP/HO-* credentials остаются внутри основного Remnawave Config Profile.

### Controller ↔ edge

Heartbeat защищён отдельным bearer token на каждый edge. Один скомпрометированный edge token не должен давать право регистрировать heartbeat от другого edge.

## Credentials

Поддерживаются два режима:

### Legacy deterministic credential

HMAC от нормализованного user id. Оставлен для migration/compatibility. Это долгоживущий credential, поэтому для новых клиентов предпочтителен `ft1`.

### `ft1`

Short-lived HMAC token:

```text
ft1.<expires_unix>.<nonce>.<signature>
```

Controller и edge проверяют expiration/signature. Максимальный Trust-issued TTL ограничен 24 часами.

## Zorin Trust exchange

При включённом Trust Controller принимает только:

```text
version  = ZOWNER/1
action   = fabric.session.issue
resource = fabric:user:<normalized shortUuid>
```

Проверяются:

- P-256 signature;
- canonical phone fingerprint;
- trusted-phone registry;
- phone→Remnawave user binding;
- proof timing;
- nonce one-shot replay cache;
- текущий Remnawave user status.

После этого выдаётся `ft1`; raw phone signing key в сетевые компоненты не передаётся.

## Secrets that must not be committed

- Fabric auth master key;
- per-edge heartbeat tokens;
- Remnawave API token;
- Yandex/VK service cookies;
- Xray REALITY private keys;
- Shadowsocks passwords;
- provider account sessions;
- vault keys.

`deploy/**/secrets/README.md` и `*.example` можно коммитить; реальные secret files должны оставаться вне Git.

## Provider account isolation

Для production разумно использовать отдельные service accounts для VK/Telemost edge automation и отдельные cookies на разные failure/security domains. Cookie vault key не должен совпадать с Fabric auth key.

## Exposure

- `FABRIC-VCONNECT-*` Xray inbounds: только `127.0.0.1`.
- Controller container: по умолчанию только `127.0.0.1:47800`; наружу — через HTTPS reverse proxy.
- Remnawave Panel/API: используйте существующую защищённую deployment model; Fabric не требует публиковать её клиентам.

## Operational rotation

Если production-конфиг с реальными private keys/passwords/API credentials был отправлен в чат, ticket, лог или иной внешний канал, безопаснее считать эти значения раскрытыми и выполнить rotation до финального production rollout.
