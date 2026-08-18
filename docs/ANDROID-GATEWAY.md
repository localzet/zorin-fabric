# Android USB Gateway

Target UX:

```text
foreign PC
   │ USB
   ▼
Android phone
   ├─ Internet → VConnect/VK-or-Telemost → RU → Xray → EU
   └─ *.knot   → KnotRoute
```

The foreign computer should only see a normal network gateway/DNS and should not receive Fabric, Remnawave, Xray, KnotRoute or Trust private credentials.

## What 1.0 provides

The Android VConnect source now understands Fabric Controller settings and `fabric://auto`. Trust-side short-lived credential exchange exists server-side. These are prerequisites for a phone-centric gateway.

## What is not honestly universal on stock Android

A normal third-party Android app cannot assume it can programmatically control every tethering function on every OEM, and an app `VpnService` does not by itself prove that traffic forwarded from a tethered USB client will always traverse the app VPN path.

Therefore Fabric does **not** claim that arbitrary stock Android phones already implement the final “plug cable, press Allow, all PC traffic is tunneled” experience.

## Supported implementation tracks

### A. Privileged/system build — target full experience

For an AOSP/custom/system-app environment:

1. manage USB tethering programmatically;
2. own forwarding/policy-routing rules for tether clients;
3. run Fabric/VConnect + KnotRoute on phone;
4. expose phone DNS that resolves `.knot` locally and forwards ordinary DNS through Fabric;
5. keep all network credentials in Android keystore/app storage;
6. revoke forwarding immediately when Trust session ends.

This is the path to the exact no-software-on-PC UX.

### B. Stock Android compatibility mode

User manually enables USB tethering. Fabric verifies per-device behavior and can provide diagnostics, but OEM/kernel behavior must be tested rather than assumed.

### C. Portable companion fallback

Where stock tethering cannot be forced through Fabric reliably, use a tiny temporary host companion that speaks to the phone over USB and leaves no long-lived account/network secret on the foreign PC.

## Security rule

The foreign PC never receives a reusable Remnawave subscription, Xray private material, KnotRoute identity or phone private key. At most it receives a temporary local network route through the phone.
