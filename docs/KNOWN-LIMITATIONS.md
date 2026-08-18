# Known limitations — 1.0.0

## 1. Telemost remains the AUTO service/control channel

The **useful work/data call can be VK or Telemost**, but current Android/Desktop `fabric://auto` resolves a Telemost service/control call first.

If Telemost control is completely unavailable while VK remains available, the current AUTO bootstrap does not automatically switch to a VK service-control implementation. This needs a service-aware VK control/captcha path, not just changing a URL field.

## 2. Full Android USB gateway is platform-dependent

See `ANDROID-GATEWAY.md`. The privileged/system track can implement the intended seamless UX; arbitrary stock Android tethering cannot be promised generically.

## 3. Full mobile/desktop artifacts were not built in this sandbox

The source VConnect modules require Go 1.26.1 and have Pion/gomobile dependencies that are not present in the sandbox module cache. Android Gradle and Electron dependency downloads are also unavailable.

The added standard-library Fabric handoff core, Controller and Python tooling were compiled/tested separately; see `../VALIDATION.md`.

## 4. Controller is deliberately simple

Edge ranking is deterministic `online → priority → load`. It is not yet a predictive network-quality scheduler. RU→EU selection remains Xray's responsibility.

## 5. Trust provisioning still needs product UI

Controller can verify real `ZOWNER/1` format and issue one-shot short-lived tokens, but the final cross-app Android UX that asks Zorin Trust Runtime for a Fabric proof is not packaged as an APK in this release.

## 6. KnotRoute is integrated as carrier adapter, not merged

The adapter changes transport routing only. A unified client UX/DNS split between normal Internet and `.knot` still needs client-level orchestration.

## 7. Matrix is not included

Matrix remains a future application integration and is intentionally outside the critical network path.
