# Zorin Fabric 1.0.1

Zorin Fabric объединяет существующие VConnect, Xray/Remnawave, KnotRoute и Zorin Trust вокруг одной задачи: дать клиенту устойчивый сетевой доступ, не переписывая уже работающие сетевые компоненты.

## Что здесь является ядром

VConnect в Fabric — это **access transport через VK Звонки / Яндекс Телемост**. Полезный трафик идёт внутри рабочего звонка до RU edge. После RU edge VConnect заканчивается: дальнейший транзит и выход в Internet выполняет Xray, а пользователи/ноды/Config Profiles остаются под управлением Remnawave.

```text
Client
  │
  │ fabric://auto → Fabric Controller выбирает живой RU edge
  ▼
VConnect service/control call
  │
  │ создаётся персональный work call: Telemost или VK
  ▼
RU VConnect Edge
  │
  │ per-user Xray bridge, identity = Remnawave vlessUuid
  ▼
RU Remnawave/Xray
  │
  ├─ private/RU → DIRECT
  └─ external   → AUTO / FI / NL / GE / AE, DIRECT/WARP
                    │
                    ▼
                  EU edge
                    │
                    ▼
                 Internet
```

Fabric Controller **не находится в data plane**. Он только проверяет пользователя, принимает heartbeat RU edge и выдаёт клиенту актуальную service-call ссылку.

## Что изменилось относительно v0.2

1. **Multi-RU control plane.** `fabric-controller` хранит состояние RU edge, сортирует их по online/priority/load и отдаёт bootstrap клиенту.
2. **User-aware VConnect → Xray handoff.** На work-session создаётся отдельный локальный Xray bridge. Он входит в основной RU Xray с `vlessUuid` конкретного Remnawave пользователя, поэтому Xray снова видит пользователя, а не общий SOCKS ingress.
3. **Короткоживущие Fabric tokens.** Помимо legacy deterministic HMAC поддержан `ft1` с TTL. Controller умеет обменять свежий Zorin Trust `ZOWNER/1` proof на такой token и защищает exchange от replay.
4. **`fabric://auto` в Android и desktop source.** Клиент получает живой RU edge от Controller и затем запускает обычный VConnect service/work flow.
5. **VK work path исправлен.** `WORK_VK_PEER_ID` и service-side VK cookies передаются spawned VK creator; полезный канал может быть VK или Telemost.
6. **Remnawave-aware compiler v2.** Он генерирует loopback-only VLESS Fabric inbounds и связывает их с существующими `BL-*`/`HO-*`, не копируя RU→EU routing внутрь VConnect.
7. **KnotRoute adapter.** Его transport manager можно направить через локальный Fabric/VConnect SOCKS, сохраняя `.knot`, E2E overlay и пользовательские services отдельным plane.

## Быстрый маршрут к запуску

Подробно: [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md).

```bash
# 1. Скомпилировать текущий RU Remnawave/Xray Config Profile
python3 tools/remnawave-fabric-compile.py \
  --ru-config ./ru-profile.json \
  --xray-out ./ru-profile.fabric.json \
  --routes-out ./deploy/ru-edge/generated/fabric-routes.json \
  --inventory-out ./deploy/ru-edge/generated/fabric-inventory.json

# 2. Создать базовые секреты (API token/cookies скрипт намеренно не генерирует)
python3 tools/init-fabric-secrets.py --edges ru-1,ru-2

# 3. Настроить Controller и первый RU edge
# deploy/controller/fabric-controller.json
# deploy/ru-edge/.env

# 4. Запустить control plane и edge
cd deploy/controller && docker compose up -d --build
cd ../ru-edge && docker compose up -d --build
```

После импорта `ru-profile.fabric.json` в Remnawave необходимо включить новые `FABRIC-VCONNECT-*` inbounds на RU node и дать их тем же Fabric-пользователям через Internal Squad. Массив `clients` компилятор намеренно оставляет пустым: пользователями этих inbound'ов продолжает владеть Remnawave.

## Репозиторий

- `fabric-controller/` — multi-RU bootstrap, heartbeat, Remnawave admission, Trust exchange.
- `vconnect-fabric/` — полный VConnect source с multi-user creator-service и per-user Xray bridge.
- `tools/remnawave-fabric-compile.py` — адаптация действующего RU Config Profile.
- `tools/fabric-credential.py` — legacy и short-lived Fabric credentials.
- `tools/init-fabric-secrets.py` — локальная генерация базовых secret files.
- `tools/knotroute-fabric-compile.py` — подключение KnotRoute transport к Fabric SOCKS.
- `deploy/controller/` — compose/config Controller.
- `deploy/ru-edge/` — compose/config RU edge.
- `integrations/knotroute/` — пример overlay integration.
- `docs/` — архитектура, deployment, Trust, Android gateway, security и ограничения.

## Что 1.0.0 не притворяется решённым

- Stock Android USB tethering нельзя честно объявить универсальным «в один клик» без системных привилегий/OEM-проверки. Архитектура описана в `docs/ANDROID-GATEWAY.md`.
- Android AUTO bootstrap сейчас использует Telemost как service/control call; **work/data call уже может быть VK или Telemost**. Полный automatic VK-control fallback требует отдельной service-aware VK/captcha интеграции.
- Matrix не включён в critical path. Его разумно добавлять позже как application поверх Fabric/KnotRoute.
- В этой песочнице не удалось собрать весь Android/Electron/Pion стек из-за отсутствующих внешних toolchain/dependencies. Что именно реально скомпилировано и протестировано, перечислено в `VALIDATION.md`.

## Security

Не коммитьте Remnawave API tokens, provider cookies, Fabric master keys и operational Xray credentials. Если реальный production config когда-либо передавался третьей стороне или попадал в историю/чат, используемые там секреты лучше ротировать до production rollout.

## 1.0.1 operational additions

- Provider cookies больше не нужно писать вручную: VConnect Creator уже имеет **Export Cookies**; `tools/cookies/install-cookie-bundle.py` валидирует его `cookies.zip` и устанавливает secrets.
- Серверный quickstart: [`docs/SERVER-QUICKSTART.md`](docs/SERVER-QUICKSTART.md).
- Git split: [`docs/GIT-REPOSITORIES.md`](docs/GIT-REPOSITORIES.md). Для публикации подготовлены отдельные деревья в `git-ready/`.
- RU edge compose разделён на runtime compose + `docker-compose.build.yml`, поэтому `zorin-fabric` и `vconnect` могут жить в отдельных Git repositories.
