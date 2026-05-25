### Adversarial Review Report

**Phase:** design
**Artifact:** docs/plans/2026-05-25-observability-plugin-design.md
**Status:** PASS

**Findings (Critical):**
- None.

**Findings (Important):**
- None.

**Findings (Minor):**
- [Repo-precedent conflicts] Workflow Core Contracts: The host-side bridge adds
  a new core runtime responsibility. Recommendation: keep it small, no-op by
  default, and wire it through existing service invocation patterns rather than
  adding plugin-specific imports.
- [Missing failure modes] Workflow Core Contracts: Sink forwarding can fail or
  block. Recommendation: plan bounded collection intervals, context timeouts,
  and non-fatal diagnostics so telemetry never breaks application traffic.
- [YAGNI violations] Delivery Phases: The first phase now includes both core
  interfaces and bridge behavior. Recommendation: implement only in-memory/test
  sink plus plugin service sink in phase one, leaving provider deployment to
  later tasks.

**Bug-class scan transcript:**

| Class | Result | Note |
|---|---|---|
| Unstated assumptions | Clean | The design now states external plugins cannot inspect host service registries directly. |
| Repo-precedent conflicts | Finding | The bridge must follow existing service invocation patterns. |
| YAGNI violations | Finding | Bridge scope must stay minimal in phase one. |
| Missing failure modes | Finding | Sink failure/backpressure must be non-fatal and time-bounded. |
| Security / privacy at architecture level | Clean | Secret handling, private endpoints, public ingest auth, and PII filtering remain covered. |
| Rollback story | Clean | The no-op sink provides a clean rollback/inert behavior when the plugin is absent. |
| Simpler alternative not considered | Clean | The no-op bridge plus external collector wiring remains the smallest viable architecture that satisfies interface discovery. |
| User-intent drift | Clean | Host-side discovery is necessary to satisfy the user's interface-without-hard-dependency requirement. |

**Options the author may not have considered:**

1. Make every emitter push directly to a global OTel provider. This is simpler
   but reintroduces hard OTel assumptions and global state.
2. Require apps to manually call the observability plugin. This avoids a bridge
   but violates plug-and-play YAML composition.

**Verdict reasoning:** The design now accounts for the process boundary between
Workflow core and external plugins. Remaining concerns are implementation
constraints for the plan, not design blockers.

