#!/usr/bin/env python3
"""Attach KnotRoute's overlay transport to a local Zorin Fabric/VConnect SOCKS gateway.

The tool edits only KnotRoute's `transport` section. KnotRoute keeps its own
SOCKS/HTTP application proxies and all overlay cryptography/routing semantics.
"""

from __future__ import annotations

import argparse
import json
import pathlib
import sys
from typing import Any

FABRIC_ENDPOINT_NAME = "zorin-fabric-vconnect"


def load_json(path: pathlib.Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as handle:
        value = json.load(handle)
    if not isinstance(value, dict):
        raise ValueError("KnotRoute config root must be a JSON object")
    return value


def compile_config(
    config: dict[str, Any],
    *,
    socks_endpoint: str,
    priority: int,
    mode: str,
    direct_first: bool,
    fallback_direct: bool,
    username: str = "",
    password: str = "",
) -> dict[str, Any]:
    out = json.loads(json.dumps(config))
    transport = out.setdefault("transport", {})
    if not isinstance(transport, dict):
        raise ValueError("transport must be a JSON object")

    endpoints = transport.get("endpoints") or []
    if not isinstance(endpoints, list):
        raise ValueError("transport.endpoints must be an array")

    endpoints = [
        endpoint
        for endpoint in endpoints
        if not (
            isinstance(endpoint, dict)
            and endpoint.get("name") == FABRIC_ENDPOINT_NAME
        )
    ]

    endpoint: dict[str, Any] = {
        "name": FABRIC_ENDPOINT_NAME,
        "type": "socks5",
        "endpoint": socks_endpoint,
        "priority": priority,
        "enabled": True,
    }
    if username:
        endpoint["username"] = username
    if password:
        endpoint["password"] = password

    endpoints.append(endpoint)
    transport["mode"] = mode
    transport["direct_first"] = direct_first
    transport["fallback_direct"] = fallback_direct
    transport["endpoints"] = endpoints
    return out


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Attach KnotRoute transport.Manager to a local VConnect/Fabric SOCKS gateway."
    )
    parser.add_argument("--input", required=True, help="KnotRoute JSON config")
    parser.add_argument("--output", required=True, help="Output KnotRoute JSON config")
    parser.add_argument(
        "--socks",
        default="127.0.0.1:1080",
        help="SOCKS5 listener exposed by the local VConnect/Fabric client",
    )
    parser.add_argument("--priority", type=int, default=10)
    parser.add_argument(
        "--mode",
        choices=("auto", "proxy", "direct"),
        default="auto",
        help="KnotRoute transport mode",
    )
    parser.add_argument(
        "--fabric-first",
        action="store_true",
        help="Try the Fabric/VConnect carrier before direct dialing",
    )
    parser.add_argument(
        "--no-direct-fallback",
        action="store_true",
        help="Do not fall back to direct peer dialing",
    )
    parser.add_argument("--username", default="")
    parser.add_argument("--password", default="")
    args = parser.parse_args()

    src = pathlib.Path(args.input)
    dst = pathlib.Path(args.output)

    try:
        config = load_json(src)
        compiled = compile_config(
            config,
            socks_endpoint=args.socks,
            priority=args.priority,
            mode=args.mode,
            direct_first=not args.fabric_first,
            fallback_direct=not args.no_direct_fallback,
            username=args.username,
            password=args.password,
        )
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 2

    dst.parent.mkdir(parents=True, exist_ok=True)
    with dst.open("w", encoding="utf-8") as handle:
        json.dump(compiled, handle, indent=2, ensure_ascii=False)
        handle.write("\n")

    print(f"Wrote {dst}")
    print(
        "KnotRoute transport: "
        f"{FABRIC_ENDPOINT_NAME} -> SOCKS5 {args.socks}; "
        f"direct_first={not args.fabric_first}; "
        f"fallback_direct={not args.no_direct_fallback}"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
