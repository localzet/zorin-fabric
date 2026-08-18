#!/usr/bin/env python3
"""Generate local Zorin Fabric secret files without printing secret values."""

from __future__ import annotations

import argparse
import base64
import os
import secrets
from pathlib import Path


def b64_key(size: int = 32) -> str:
    return base64.urlsafe_b64encode(os.urandom(size)).decode("ascii").rstrip("=")


def token(size: int = 32) -> str:
    return secrets.token_urlsafe(size)


def write_secret(path: Path, value: str, force: bool) -> bool:
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists() and not force:
        return False
    path.write_text(value + "\n", encoding="utf-8")
    try:
        path.chmod(0o600)
    except OSError:
        pass
    return True


def main() -> int:
    parser = argparse.ArgumentParser(description="Initialize Zorin Fabric secret files")
    parser.add_argument("--controller-dir", default="deploy/controller/secrets")
    parser.add_argument("--edge-dir", default="deploy/ru-edge/secrets")
    parser.add_argument("--edges", default="ru-1,ru-2", help="comma-separated edge ids")
    parser.add_argument("--force", action="store_true", help="overwrite generated secret files")
    args = parser.parse_args()

    controller_dir = Path(args.controller_dir)
    edge_dir = Path(args.edge_dir)
    edge_ids = [item.strip() for item in args.edges.split(",") if item.strip()]
    if not edge_ids:
        parser.error("--edges must contain at least one edge id")

    shared_fabric_key = b64_key()
    controller_key_path = controller_dir / "fabric-auth-key.txt"
    edge_key_path = edge_dir / "fabric-auth-key.txt"

    if controller_key_path.exists() and not args.force:
        shared_fabric_key = controller_key_path.read_text(encoding="utf-8").strip()

    created: list[Path] = []
    skipped: list[Path] = []

    for path, value in (
        (controller_key_path, shared_fabric_key),
        (edge_key_path, shared_fabric_key),
        (edge_dir / "vault-key.txt", b64_key()),
    ):
        (created if write_secret(path, value, args.force) else skipped).append(path)

    for edge_id in edge_ids:
        safe_name = edge_id.replace("/", "-")
        controller_path = controller_dir / f"edge-{safe_name}.txt"
        value = token()
        if controller_path.exists() and not args.force:
            value = controller_path.read_text(encoding="utf-8").strip()
        (created if write_secret(controller_path, value, args.force) else skipped).append(controller_path)

        # RU edge deployment directory represents one edge at a time. For convenience,
        # copy the first requested edge token to the canonical compose filename.
        if edge_id == edge_ids[0]:
            edge_path = edge_dir / "edge-heartbeat-token.txt"
            (created if write_secret(edge_path, value, args.force) else skipped).append(edge_path)

    print("Zorin Fabric secrets initialized.")
    if created:
        print("Created:")
        for path in created:
            print(f"  {path}")
    if skipped:
        print("Kept existing:")
        for path in skipped:
            print(f"  {path}")
    print("Remnawave API token and provider cookie files are intentionally not generated.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
