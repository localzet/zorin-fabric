# Zorin Trust integration

Fabric 1.0 не смешивает Trust keys с Xray/VConnect identities. Trust используется как hardware-backed approval layer.

## Exchange flow

```text
Android Zorin Trust Runtime
      │
      │ ZOWNER/1 proof
      ▼
Fabric Controller /v1/trust/exchange
      │
      ├─ verify phone key/fingerprint
      ├─ verify binding phone → Remnawave shortUuid
      ├─ verify action/resource/timing
      ├─ consume nonce once
      └─ verify Remnawave ACTIVE
      │
      ▼
short-lived ft1 Fabric credential
      │
      ▼
fabric://auto bootstrap + VConnect session
```

Required proof scope:

```text
action   = fabric.session.issue
resource = fabric:user:<shortUuid>
```

## Registry format

`deploy/controller/trust-registry.json`:

```json
{
  "trustedPhones": {
    "AAAA:BBBB:CCCC:...": "<PKIX DER HEX>"
  },
  "bindings": {
    "AAAA:BBBB:CCCC:...": "remnawave-short-uuid"
  }
}
```

Trust disabled by default in example Controller config. Enable only after registry provisioning is automated/audited.

## What remains intentionally separate

- Android hardware private key;
- KnotRoute Ed25519 identities;
- Remnawave/Xray VLESS UUID;
- ephemeral Fabric token.

This avoids one universal credential whose compromise breaks every layer.
