# KnotRoute integration

KnotRoute is a functional overlay plane, not the public-Internet egress router.

## Intended split

```text
ordinary domains/IP
  → Fabric/VConnect
  → RU Xray
  → EU / Internet

*.knot / private service identities
  → KnotRoute
  → overlay routing/circuits
```

KnotRoute can still use Fabric as its physical carrier when direct peer reachability is poor.

## Compiler

```bash
python3 tools/knotroute-fabric-compile.py \
  --input integrations/knotroute/knotroute.base.example.json \
  --output knotroute.fabric.json \
  --socks 127.0.0.1:1080 \
  --fabric-first
```

This only modifies the `transport` section. It does not replace KnotRoute identity, `.knot` DNS, services, circuits, HTTP/SOCKS application proxies or overlay encryption.

Example result is in `integrations/knotroute/knotroute.fabric.example.json`.

## Product features enabled by this plane

- internal `.knot` DNS;
- private websites/services;
- users publishing their own local services without public IPv4/port forwarding;
- future file/RPC/messaging application protocols;
- a future private Matrix deployment can use this plane for internal/federation connectivity without becoming part of the anti-blocking core.
