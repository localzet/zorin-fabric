# Zorin Fabric 1.0.0 — final status

## Production-shaped core: implemented

- VConnect through VK Calls / Yandex Telemost to RU edge.
- Multi-user creator-service.
- Multi-RU Fabric Controller and edge heartbeat/load discovery.
- Remnawave ACTIVE-user admission.
- Short-lived Fabric credentials and Trust proof exchange.
- Per-user VConnect→Xray bridge using Remnawave `vlessUuid`.
- Existing Remnawave/Xray RU→EU route topology retained as the transit source of truth.
- AUTO + explicit region/mode route inventory generated from the current RU profile.
- VK work-call server plumbing (`peer_id`, service cookies).
- Android/Desktop `fabric://auto` source integration.
- KnotRoute carrier adapter.
- Controller/RU Edge Docker Compose deployments.

## Deliberately not claimed complete

- universal stock-Android zero-click USB gateway;
- automatic VK service/control fallback when Telemost control is unavailable;
- packaged Android APK / Electron installers built inside this sandbox;
- Matrix integration;
- predictive/ML route engine.

These are no longer blockers for the core network architecture and are isolated follow-up tracks.
