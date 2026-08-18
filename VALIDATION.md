# Validation — 1.0.0

This file records what was actually checked in the available sandbox and what was not.

## PASS — Fabric Controller

Executed from `fabric-controller/`:

```bash
go test ./...
go build ./cmd/fabric-controller \
         ./cmd/fabric-edge-agent \
         ./cmd/fabric-bootstrap \
         ./cmd/fabric-trust-exchange \
         ./cmd/fabric-health
```

Result: PASS with Go 1.23.2 available in the sandbox.

Controller tests cover credential verification/token expiry, bootstrap/heartbeat flow and Trust proof verification/replay protection.

## PASS — VConnect Fabric handoff core

The complete VConnect module cannot be rebuilt in this sandbox because its source declares Go 1.26.1 and required Pion/gomobile modules are not present in the local module cache. External toolchain/dependency download is blocked by the sandbox network.

To avoid treating `gofmt` as validation, the changed standard-library Fabric core was copied into an isolated compile/test harness with only the external UUID package stubbed. The harness contains the **same current source files** for:

- Fabric credentials;
- Remnawave directory/authorizer;
- session manager;
- process factory;
- egress registry/dialers;
- per-user Xray bridge generation.

Executed:

```bash
go test ./controlplane ./egress ./common
```

Result: PASS.

`xray_user_bridge_test.go` verifies that generated sidecar config contains the selected user's VLESS UUID and produces the per-route local SOCKS egress inventory.

This is not a substitute for the normal full VConnect CI with Go 1.26.1/Pion/media dependencies, but it is a real compile/test of the integration core added in this release.

## PASS — Remnawave compiler against the supplied current RU profile

Executed `tools/remnawave-fabric-compile.py` against the actual uploaded RU Config Profile.

Verified:

- exactly 9 policies are discovered/generated:
  `auto`, `auto-warp`, `ae-direct`, `fi-direct`, `fi-warp`, `ge-direct`, `ge-warp`, `nl-direct`, `nl-warp`;
- all generated Fabric Xray inbounds bind to `127.0.0.1`;
- all generated Fabric inbounds are VLESS/raw/security=none and keep `clients: []`;
- private/RU rules stay before generated Fabric routing rules;
- user uplink/downlink stats flags are enabled without replacing existing policy fields;
- compiler output is idempotent for Xray config, route inventory and topology inventory;
- generated secret-free route/inventory artifacts were refreshed from this run.

Result: PASS.

## PASS — KnotRoute adapter

Executed compiler against the bundled base example twice and compared output.

Verified:

- endpoint `zorin-fabric-vconnect` points to local SOCKS5 `127.0.0.1:1080`;
- Fabric-first/direct-fallback policy is preserved;
- second compilation is identical.

Result: PASS.

## PASS — Python / shell / config formats

- `python3 -m py_compile tools/*.py` — PASS.
- VConnect Docker entrypoints `sh -n` — PASS.
- Controller/RU-edge/creator Compose YAML parsed with PyYAML — PASS.
- bundled JSON examples/inventories parsed with `python3 -m json.tool` — PASS.

## PASS — Desktop renderer subset

The renderer/constants files changed for Fabric AUTO were checked with the available `tsc`:

```bash
tsc --noEmit --target ES2022 --lib ES2022,DOM \
    --module commonjs --strict false \
    src/constants.ts src/renderer/index.ts
```

Result: PASS.

A full Electron build is **not** claimed because the sandbox does not contain/fetch the project's Electron/Node dependency tree.

## Source-level only — Android

Android source now includes Fabric Controller settings and `fabric://auto`, but a full Gradle/APK build is not claimed. The Gradle distribution/dependencies are unavailable offline in this sandbox.

## PASS — Secret/topology hygiene

The final tree was scanned for the actual REALITY private keys, Shadowsocks password, user VLESS UUID and operational IP values that appeared in the supplied configs. No such values are present in the release tree.

The full supplied RU profile is intentionally not bundled.

## Known functional boundary

AUTO clients use a Telemost service/control call in 1.0. The work/data call itself can be either Telemost or VK. A fully automatic VK **service-control** fallback requires additional VK service-aware/captcha plumbing and is documented rather than falsely marked complete.
