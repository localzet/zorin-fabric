#!/usr/bin/env python3
"""Install a VConnect Creator `cookies.zip` into RU edge secrets.

The Creator App exports cookies-yandex.json and cookies-vk.json in the exact
minimal format used by the headless creators: [{"name": "...", "value": "..."}].
This tool validates that format, checks for the expected auth cookies and writes
files with mode 0600. It never reads or decrypts browser cookie databases.
"""
from __future__ import annotations

import argparse
import json
import os
import shutil
import stat
import sys
import tempfile
import zipfile
from pathlib import Path

PLATFORMS = {
    "yandex": {
        "filename": "cookies-yandex.json",
        "auth": {"Session_id", "sessionid2"},
        "required_any": {"Session_id", "sessionid2"},
    },
    "vk": {
        "filename": "cookies-vk.json",
        "auth": {"remixsid"},
        "required_any": {"remixsid"},
    },
}


def load_entries(path: Path) -> list[dict[str, str]]:
    try:
        raw = json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:
        raise ValueError(f"{path.name}: invalid JSON: {exc}") from exc
    if not isinstance(raw, list):
        raise ValueError(f"{path.name}: expected a JSON array")
    out: list[dict[str, str]] = []
    seen: set[str] = set()
    for i, item in enumerate(raw):
        if not isinstance(item, dict):
            raise ValueError(f"{path.name}: item #{i} is not an object")
        name = item.get("name")
        value = item.get("value")
        if not isinstance(name, str) or not name:
            raise ValueError(f"{path.name}: item #{i} has no cookie name")
        if not isinstance(value, str):
            raise ValueError(f"{path.name}: item #{i} has no string value")
        if name in seen:
            # Keep the latest copy, matching Creator's effective cookie set.
            out = [x for x in out if x["name"] != name]
        out.append({"name": name, "value": value})
        seen.add(name)
    return out


def validate(platform: str, entries: list[dict[str, str]], allow_empty: bool) -> None:
    names = {x["name"] for x in entries}
    if not entries and allow_empty:
        return
    required_any = PLATFORMS[platform]["required_any"]
    if not (names & required_any):
        wanted = ", ".join(sorted(required_any))
        raise ValueError(
            f"{PLATFORMS[platform]['filename']}: authentication cookie not found "
            f"(expected one of: {wanted}). Log in in VConnect Creator and export again."
        )


def atomic_write(path: Path, entries: list[dict[str, str]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, temp_name = tempfile.mkstemp(prefix=f".{path.name}.", dir=str(path.parent))
    temp = Path(temp_name)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as fh:
            json.dump(entries, fh, ensure_ascii=False, indent=2)
            fh.write("\n")
        os.chmod(temp, stat.S_IRUSR | stat.S_IWUSR)
        os.replace(temp, path)
    finally:
        temp.unlink(missing_ok=True)


def extract_source(source: Path, temp_dir: Path) -> Path:
    if source.is_dir():
        return source
    if not source.is_file():
        raise ValueError(f"source does not exist: {source}")
    if source.suffix.lower() != ".zip":
        raise ValueError("source must be Creator cookies.zip or a directory containing cookie JSON files")
    with zipfile.ZipFile(source) as zf:
        for member in zf.infolist():
            name = Path(member.filename)
            if name.is_absolute() or ".." in name.parts:
                raise ValueError(f"unsafe ZIP member: {member.filename}")
        zf.extractall(temp_dir)
    return temp_dir


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("source", help="cookies.zip exported by VConnect Creator, or extracted directory")
    parser.add_argument(
        "--output",
        default="deploy/ru-edge/secrets",
        help="RU edge secrets directory (default: deploy/ru-edge/secrets)",
    )
    parser.add_argument(
        "--require-vk",
        action="store_true",
        help="fail when VK cookies are missing/unauthenticated",
    )
    args = parser.parse_args()

    source = Path(args.source).expanduser().resolve()
    output = Path(args.output).expanduser().resolve()

    try:
        with tempfile.TemporaryDirectory(prefix="fabric-cookies-") as td:
            root = extract_source(source, Path(td))
            installed: list[str] = []
            for platform, cfg in PLATFORMS.items():
                candidate = root / cfg["filename"]
                if not candidate.exists():
                    if platform == "vk" and not args.require_vk:
                        entries: list[dict[str, str]] = []
                    else:
                        raise ValueError(f"{cfg['filename']} not found in {source.name}")
                else:
                    entries = load_entries(candidate)
                validate(platform, entries, allow_empty=(platform == "vk" and not args.require_vk))
                target = output / cfg["filename"]
                atomic_write(target, entries)
                installed.append(f"{target} ({len(entries)} cookies)")
    except (ValueError, zipfile.BadZipFile) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 2

    print("Installed provider cookies:")
    for item in installed:
        print(f"  - {item}")
    print("Permissions: 0600")
    print("Do not commit these files to Git.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
