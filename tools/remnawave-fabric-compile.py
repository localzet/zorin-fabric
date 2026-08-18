#!/usr/bin/env python3
"""Компилятор Remnawave/Xray-профиля для Zorin Fabric.

Существующий RU Config Profile остаётся источником истины. Скрипт добавляет
loopback-only VLESS ingress'ы для VConnect и привязывает их к уже имеющимся
BL-<REGION>-(DIRECT|WARP) балансировщикам. Массив clients намеренно остаётся
пустым: его заполняет Remnawave для разрешённых пользователей.
"""
from __future__ import annotations

import argparse
import copy
import json
import re
import sys
from pathlib import Path

FABRIC_PREFIX = "FABRIC-VCONNECT-"
AUTO_DIRECT_BALANCER = "BL-FABRIC-AUTO-DIRECT"
AUTO_WARP_BALANCER = "BL-FABRIC-AUTO-WARP"
SCHEMA_VERSION = 2


def load_json(path: Path) -> dict:
    with path.open("r", encoding="utf-8") as fh:
        return json.load(fh)


def save_json(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as fh:
        json.dump(value, fh, ensure_ascii=False, indent=2)
        fh.write("\n")


def discover_balancers(config: dict) -> dict[tuple[str, str], dict]:
    result: dict[tuple[str, str], dict] = {}
    for balancer in config.get("routing", {}).get("balancers", []):
        tag = str(balancer.get("tag", ""))
        match = re.fullmatch(r"BL-([A-Z0-9]+)-(DIRECT|WARP)", tag)
        if match:
            result[(match.group(1), match.group(2))] = balancer
    return result


def discover_outbound_tags(config: dict) -> set[str]:
    return {
        str(item.get("tag", ""))
        for item in config.get("outbounds", [])
        if item.get("tag")
    }


def pick_fallback(selectors: list[str], outbound_tags: set[str]) -> str | None:
    # Сначала предпочитаем PUB: это предсказуемый fallback при проблемах
    # observatory. Если PUB нет, берём первый существующий outbound по префиксу.
    for prefix in selectors:
        preferred = prefix + "PUB"
        if preferred in outbound_tags:
            return preferred
    for prefix in selectors:
        candidates = sorted(tag for tag in outbound_tags if tag.startswith(prefix))
        if candidates:
            return candidates[0]
    return None


def fabric_vless_inbound(tag: str, port: int) -> dict:
    return {
        "tag": tag,
        "listen": "127.0.0.1",
        "port": port,
        "protocol": "vless",
        "settings": {
            "clients": [],
            "decryption": "none",
        },
        "sniffing": {
            "enabled": True,
            "routeOnly": True,
            "destOverride": ["http", "tls", "quic"],
        },
        "streamSettings": {
            "network": "raw",
            "security": "none",
            "rawSettings": {
                "header": {"type": "none"},
                "acceptProxyProtocol": False,
            },
        },
    }


def route_descriptor(
    profile_id: str,
    name: str,
    region: str,
    mode: str,
    auto: bool,
    port: int,
    default: bool,
    inbound_tag: str,
) -> dict:
    return {
        "id": profile_id,
        "displayName": name,
        "region": region,
        "mode": mode.lower(),
        "auto": auto,
        "default": default,
        "xray": {
            "address": "127.0.0.1",
            "port": port,
            "protocol": "vless",
            "network": "raw",
            "security": "none",
            "inboundTag": inbound_tag,
        },
    }


def remove_previous_fabric(config: dict) -> None:
    config["inbounds"] = [
        item
        for item in config.get("inbounds", [])
        if not str(item.get("tag", "")).startswith(FABRIC_PREFIX)
    ]
    routing = config.setdefault("routing", {})
    routing["rules"] = [
        rule
        for rule in routing.get("rules", [])
        if not any(
            str(tag).startswith(FABRIC_PREFIX)
            for tag in rule.get("inboundTag", [])
        )
    ]
    routing["balancers"] = [
        balancer
        for balancer in routing.get("balancers", [])
        if balancer.get("tag") not in {AUTO_DIRECT_BALANCER, AUTO_WARP_BALANCER}
    ]


def insertion_index(rules: list[dict]) -> int:
    """Сохраняем DNS/private/RU bypass перед Fabric-маршрутизацией."""
    idx = 0
    for i, rule in enumerate(rules):
        inbound_tags = [str(tag) for tag in rule.get("inboundTag", [])]
        if inbound_tags and not all(tag == "DNS-INTERNAL" for tag in inbound_tags):
            return i
        if rule.get("network") == "tcp,udp" and rule.get("outboundTag"):
            return i
        idx = i + 1
    return idx


def ensure_stats_policy(config: dict) -> None:
    """Включаем user stats, не ломая существующие policy-настройки."""
    config.setdefault("stats", {})
    policy = config.setdefault("policy", {})
    levels = policy.setdefault("levels", {})
    level0 = levels.setdefault("0", {})
    level0["statsUserUplink"] = True
    level0["statsUserDownlink"] = True


def compile_config(config: dict, base_port: int, strategy: str) -> tuple[dict, dict, dict]:
    output = copy.deepcopy(config)
    remove_previous_fabric(output)
    ensure_stats_policy(output)

    balancers = discover_balancers(output)
    if not balancers:
        raise ValueError("no BL-<REGION>-(DIRECT|WARP) balancers were found")

    outbound_tags = discover_outbound_tags(output)
    regions = sorted({region for region, _ in balancers})
    direct_selectors: list[str] = []
    warp_selectors: list[str] = []
    for (region, mode), balancer in sorted(balancers.items()):
        selectors = [str(value) for value in balancer.get("selector", []) if value]
        if mode == "DIRECT":
            direct_selectors.extend(selectors)
        elif mode == "WARP":
            warp_selectors.extend(selectors)

    routing = output.setdefault("routing", {})
    routing.setdefault("balancers", [])
    routing.setdefault("rules", [])

    routes: list[dict] = []
    new_rules: list[dict] = []
    new_inbounds: list[dict] = []
    next_port = base_port

    def add_route(
        profile_id: str,
        name: str,
        region: str,
        mode: str,
        auto: bool,
        default: bool,
        balancer_tag: str,
    ) -> None:
        nonlocal next_port
        tag = FABRIC_PREFIX + profile_id.upper().replace("-", "-")
        new_inbounds.append(fabric_vless_inbound(tag, next_port))
        new_rules.append({"inboundTag": [tag], "balancerTag": balancer_tag})
        routes.append(
            route_descriptor(
                profile_id,
                name,
                region,
                mode,
                auto,
                next_port,
                default,
                tag,
            )
        )
        next_port += 1

    if direct_selectors:
        fallback = pick_fallback(direct_selectors, outbound_tags)
        balancer = {
            "tag": AUTO_DIRECT_BALANCER,
            "selector": sorted(set(direct_selectors)),
            "strategy": {"type": strategy},
        }
        if fallback:
            balancer["fallbackTag"] = fallback
        routing["balancers"].append(balancer)
        add_route(
            "auto",
            "AUTO · best EU route",
            "AUTO",
            "DIRECT",
            True,
            True,
            AUTO_DIRECT_BALANCER,
        )

    if warp_selectors:
        fallback = pick_fallback(warp_selectors, outbound_tags)
        balancer = {
            "tag": AUTO_WARP_BALANCER,
            "selector": sorted(set(warp_selectors)),
            "strategy": {"type": strategy},
        }
        if fallback:
            balancer["fallbackTag"] = fallback
        routing["balancers"].append(balancer)
        add_route(
            "auto-warp",
            "AUTO · best EU route · WARP egress",
            "AUTO",
            "WARP",
            True,
            False,
            AUTO_WARP_BALANCER,
        )

    for region in regions:
        for mode in ("DIRECT", "WARP"):
            balancer = balancers.get((region, mode))
            if not balancer:
                continue
            add_route(
                f"{region.lower()}-{mode.lower()}",
                f"{region} · {mode.lower()}",
                region,
                mode,
                False,
                False,
                str(balancer["tag"]),
            )

    output.setdefault("inbounds", []).extend(new_inbounds)
    idx = insertion_index(routing["rules"])
    routing["rules"][idx:idx] = new_rules

    route_inventory = {
        "schemaVersion": SCHEMA_VERSION,
        "defaultRoute": "auto" if any(route["id"] == "auto" for route in routes) else routes[0]["id"],
        "routes": routes,
    }
    inventory = {
        "schemaVersion": SCHEMA_VERSION,
        "regions": regions,
        "existingBalancers": [
            balancer["tag"] for _, balancer in sorted(balancers.items())
        ],
        "fabricRoutes": [
            {
                "id": route["id"],
                "displayName": route["displayName"],
                "region": route["region"],
                "mode": route["mode"],
                "auto": route["auto"],
                "default": route["default"],
                "inboundTag": route["xray"]["inboundTag"],
            }
            for route in routes
        ],
        "listenRange": [base_port, next_port - 1],
        "strategy": strategy,
        "handoff": "per-user-vless",
    }
    return output, route_inventory, inventory


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--ru-config", required=True, type=Path)
    parser.add_argument("--xray-out", required=True, type=Path)
    parser.add_argument("--routes-out", "--egress-out", dest="routes_out", required=True, type=Path)
    parser.add_argument("--inventory-out", type=Path)
    parser.add_argument("--base-port", type=int, default=11880)
    parser.add_argument("--strategy", choices=["leastPing", "leastLoad"], default="leastPing")
    args = parser.parse_args()

    if not 1024 <= args.base_port <= 65000:
        parser.error("--base-port must be between 1024 and 65000")

    source = load_json(args.ru_config)
    patched, routes, inventory = compile_config(source, args.base_port, args.strategy)
    save_json(args.xray_out, patched)
    save_json(args.routes_out, routes)
    if args.inventory_out:
        save_json(args.inventory_out, inventory)

    print(f"Fabric routes: {len(routes['routes'])}")
    for item in routes["routes"]:
        target = item["xray"]
        print(
            f"  {item['id']:<16} "
            f"{target['address']}:{target['port']:<5} "
            f"{item['displayName']}"
        )
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        raise SystemExit(2)
