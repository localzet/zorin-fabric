#!/usr/bin/env python3
"""Утилита credential/token для Zorin Fabric.

`issue` сохраняет legacy deterministic credential для совместимости.
`token` выпускает короткоживущий ft1-токен, привязанный к Remnawave user id.
"""
from __future__ import annotations

import argparse
import base64
import hashlib
import hmac
import os
import secrets
import time
from pathlib import Path

USER_DOMAIN = b"vconnect-fabric/v1/user/"
TOKEN_DOMAIN = b"vconnect-fabric/v1/token|"


def decode_key(value: str) -> bytes:
    key = base64.b64decode(value.strip(), validate=True)
    if len(key) < 32:
        raise ValueError("master key must contain at least 32 bytes")
    return key


def read_key(path: str) -> bytes:
    return decode_key(Path(path).read_text(encoding="utf-8"))


def normalize_user(user_id: str) -> str:
    value = user_id.strip().lower()
    if not value:
        raise ValueError("user id is empty")
    return value


def issue_static(key: bytes, user_id: str) -> str:
    user_id = normalize_user(user_id)
    digest = hmac.new(key, USER_DOMAIN + user_id.encode("utf-8"), hashlib.sha256).digest()
    return base64.urlsafe_b64encode(digest).rstrip(b"=").decode("ascii")


def issue_token(key: bytes, user_id: str, ttl_seconds: int) -> str:
    user_id = normalize_user(user_id)
    if ttl_seconds <= 0 or ttl_seconds > 86400:
        raise ValueError("ttl must be between 1 and 86400 seconds")
    expires = int(time.time()) + ttl_seconds
    nonce = base64.urlsafe_b64encode(secrets.token_bytes(16)).rstrip(b"=").decode("ascii")
    message = (
        TOKEN_DOMAIN
        + user_id.encode("utf-8")
        + b"|"
        + str(expires).encode("ascii")
        + b"|"
        + nonce.encode("ascii")
    )
    digest = hmac.new(key, message, hashlib.sha256).digest()
    signature = base64.urlsafe_b64encode(digest).rstrip(b"=").decode("ascii")
    return f"ft1.{expires}.{nonce}.{signature}"


def main() -> None:
    parser = argparse.ArgumentParser(description="Zorin Fabric credential utility")
    sub = parser.add_subparsers(dest="command", required=True)

    generate = sub.add_parser("generate-key", help="generate a new server master key")
    generate.add_argument("--bytes", type=int, default=32)

    create = sub.add_parser("issue", help="issue legacy deterministic credential")
    create.add_argument("--key-file", required=True)
    create.add_argument("--user", required=True)
    create.add_argument("--provision", action="store_true")

    token = sub.add_parser("token", help="issue short-lived ft1 token")
    token.add_argument("--key-file", required=True)
    token.add_argument("--user", required=True)
    token.add_argument("--ttl", type=int, default=3600, help="token lifetime in seconds (max 86400)")
    token.add_argument("--provision", action="store_true")

    args = parser.parse_args()
    if args.command == "generate-key":
        if args.bytes < 32:
            parser.error("--bytes must be at least 32")
        print(base64.b64encode(os.urandom(args.bytes)).decode("ascii"))
        return

    key = read_key(args.key_file)
    if args.command == "issue":
        value = issue_static(key, args.user)
    else:
        value = issue_token(key, args.user, args.ttl)

    if args.provision:
        print(f"{normalize_user(args.user)}:{value}")
    else:
        print(value)


if __name__ == "__main__":
    main()
