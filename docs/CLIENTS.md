# Clients

## Fabric identity fields

A client needs:

- Controller URL;
- Remnawave `shortUuid`;
- Fabric credential (`ft1` preferred);
- desired work platform (`telemost` or `vk`);
- desired egress route (`auto` preferred).

## `fabric://auto`

This is a virtual destination, not a call URL.

```text
fabric://auto
  → GET Controller /v1/bootstrap
  → choose first online edge exposing required control platform
  → obtain current service-call URL
  → run existing VConnect service-control handshake
  → receive new work-call URL
  → tunnel traffic through that work call
```

## Android source

Settings include Controller URL and `shortUuid:credential` provisioning. Bootstrap is performed off the UI thread.

In 1.0 AUTO uses **Telemost as persistent service/control call**. The subsequently created work/data call can be Telemost or VK. A fully automatic VK control-call fallback is kept separate because VK headless control requires service-aware captcha/auth plumbing that should not be faked as complete.

## Desktop Electron source

The desktop joiner now supports the same Controller/user/credential fields.

- credential input uses password field;
- credential is not persisted in localStorage;
- credential is passed to the Go joiner through environment rather than command-line arguments;
- reconnect re-runs bootstrap and may select another RU edge.

## Local SOCKS

Existing VConnect clients can expose a local SOCKS endpoint. This is also the integration point used by KnotRoute transport when Fabric is selected as carrier.
