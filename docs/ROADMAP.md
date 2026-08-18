# Post-1.0 roadmap

1. **Dual-provider control discovery** — native Telemost + VK service/control edge advertisements and automatic provider fallback.
2. **Access quality probes** — client-observed RU edge reachability/RTT fed back into bootstrap ranking without moving data through Controller.
3. **Session migration** — reconnect/migrate VConnect work session between RU edges with minimal interruption.
4. **Trust-first enrollment UI** — Android requests `ZOWNER/1`, receives `ft1`, rotates automatically, no manually pasted long-lived credential.
5. **Android privileged gateway build** — USB tether + policy routing + `.knot` DNS for the zero-install foreign-PC scenario.
6. **Unified Fabric client** — normal Internet split to VConnect/Xray and `.knot` split to KnotRoute behind one UI/TUN/DNS runtime.
7. **KnotRoute admission capabilities** — Trust-backed membership/service authorization while keeping overlay crypto independent.
8. **Observability** — RU access availability, work-call success rate, per-user/session counters, route health, alerting.
9. **Predictive route policy** — only after enough telemetry exists; do not replace Xray `leastPing/leastLoad` prematurely.
10. **Matrix** — optional internal federated messaging application over Fabric/KnotRoute.
