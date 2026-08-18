# Patchset относительно загруженного VConnect

Основные изменённые файлы:

- `relay/tunnel/protocol.go` — optional credential в control messages;
- `relay/tunnel/relay_bridge.go` — identity + credential для egress discovery;
- `relay/service_control.go`, `relay/main.go` — передача credential из joiner;
- `relay/controlplane/user_authorizer.go` — pluggable auth contract;
- `relay/controlplane/remnawave_authorizer.go` — Remnawave + HMAC auth;
- `relay/controlplane/remnawave_authorizer_test.go` — auth tests;
- `relay/controlplane/service_handler.go` — использование authorizer;
- `headless/telemost/main.go`, `service_control.go` — `allowlist|remnawave` modes;
- `headless/docker/*` — secrets/env для RU edge;
- `joiner-desktop-app/desktop-joiner/main.go` — `--service-credential`;
- Android `Prefs`, `ServiceControl`, `ServiceJoinController`, Settings — Fabric provisioning.

Полный unified diff лежит в `vconnect-fabric.patch`.
