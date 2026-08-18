# Source notes

`vconnect-fabric/` — копия загруженного `vconnect-main.zip` без `.git`, поверх которой наложен первый Zorin Fabric integration patch.

Встроенные в исходный архив prebuilt/debug artifacts не менялись этой итерацией. Перед публичным релизом их стоит отдельно классифицировать и убрать из source release там, где они не нужны.

KnotRoute и Zorin Trust не копировались внутрь bundle: v0.1 меняет только приоритетный access path VConnect + Remnawave/Xray. Контракты их следующей интеграции зафиксированы в `docs/ARCHITECTURE.md` и `docs/ROADMAP.md`.
