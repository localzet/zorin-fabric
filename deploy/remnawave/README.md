# Remnawave/Xray integration — 1.0

Fabric не заменяет Remnawave Node и не создаёт второй RU→EU router.

`tools/remnawave-fabric-compile.py` берёт действующий RU Xray Config Profile и добавляет loopback-only **VLESS** inbounds `FABRIC-VCONNECT-*`, которые направляются в уже существующие `BL-*` balancers.

## Почему VLESS, а не SOCKS

VConnect Edge знает Remnawave `shortUuid`, а через API получает соответствующий `vlessUuid`. Для каждой work-session он поднимает ephemeral local Xray bridge и входит в основной RU Xray с `user.id = vlessUuid`.

Так основной Xray снова видит того же пользователя и может вести user-level Xray stats/accounting, вместо общего анонимного SOCKS ingress.

## Compiler

```bash
python3 ../../tools/remnawave-fabric-compile.py \
  --ru-config ./ru-profile.json \
  --xray-out ./ru-profile.fabric.json \
  --routes-out ../ru-edge/generated/fabric-routes.json \
  --inventory-out ../ru-edge/generated/fabric-inventory.json
```

Compiler сохраняет существующий порядок private/RU bypass и использует существующие FI/NL/GE/AE DIRECT/WARP balancers. `auto`/`auto-warp` строятся поверх обнаруженных `HO-*` selectors.

## Remnawave остаётся владельцем users

Сгенерированные Fabric inbounds содержат:

```json
"clients": []
```

Это намеренно. Не вставляйте UUID пользователей в source profile вручную.

После импорта profile:

1. включите `FABRIC-VCONNECT-*` inbounds на RU Node;
2. включите их в Internal Squad для Fabric users;
3. назначьте пользователей этому squad.

Remnawave продолжает инжектировать actual Xray client list и управлять user lifecycle.

## Runtime ownership

Не правьте runtime Xray JSON внутри `remnanode` как persistent source. Fabric-aware JSON должен стать Remnawave Config Profile (или его изменения должны быть перенесены в профиль), после чего обычный Remnawave lifecycle разворачивает его на Node.
