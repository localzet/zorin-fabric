# Changelog

## 1.0.0

- Added standalone multi-RU Fabric Controller and edge heartbeat agent.
- Added authenticated `fabric://auto` bootstrap with Remnawave ACTIVE-user verification.
- Added deterministic legacy credentials and short-lived `ft1` credentials.
- Added Zorin Trust `ZOWNER/1` → `ft1` exchange with P-256 verification, phone binding, timing checks and one-shot nonce replay protection.
- Reworked Remnawave compiler to generate loopback-only **VLESS** Fabric inbounds and schema-2 route inventory.
- Added per-user ephemeral Xray bridge so VConnect traffic enters main RU Xray using the actual Remnawave `vlessUuid`.
- Kept RU→EU routing entirely in the existing Remnawave/Xray `BL-*` / `HO-*` topology.
- Added AUTO and explicit region/mode policy generation from the current profile.
- Added active-user stats file and edge capacity/load heartbeat.
- Fixed VK work creator plumbing with `WORK_VK_PEER_ID` and optional service-side VK cookies.
- Added official Xray binary to the VConnect creator-service image for per-user bridges.
- Added Android source support for Fabric Controller + `fabric://auto`.
- Added desktop Electron source support for Fabric Controller + `fabric://auto`; Fabric credential is not persisted and is not passed in process arguments.
- Added KnotRoute transport compiler and example.
- Added Controller and RU Edge compose deployments.
- Added secret initializer and production/security documentation.

## 0.2.0

- Initial deployment-oriented Remnawave route compiler.
- Added AUTO/region policies against the supplied RU topology.
- Added host-network RU edge model.

## 0.1.0

- Added Remnawave-backed VConnect authorization and Fabric credentials.
