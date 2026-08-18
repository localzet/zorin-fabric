# Zorin Fabric Controller

Control plane для multi-RU доступа. Он **не проксирует пользовательский трафик**.

- `fabric-controller` проверяет Fabric credential + статус Remnawave и отдаёт список живых RU edge.
- `fabric-edge-agent` читает актуальные ссылки service-call VK/Telemost из файлов и шлёт heartbeat.
- `fabric-bootstrap` — CLI для проверки/автовыбора edge.
- `/v1/trust/exchange` меняет свежий `ZOWNER/1` proof (`action=fabric.session.issue`, `resource=fabric:user:<shortUuid>`) на короткоживущий `ft1` token.

Трафик после выбора edge идёт напрямую клиент → VConnect service call → RU edge; Controller в data plane не участвует.
