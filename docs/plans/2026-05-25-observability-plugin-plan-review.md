### Adversarial Review Report

**Phase:** plan
**Artifact:** docs/plans/2026-05-25-observability-plugin.md
**Status:** FAIL

**Findings (Critical):**
- None.

**Findings (Important):**
- [Missing failure modes] Task 2: The telemetry bridge is described as a wiring
  hook, but the plan does not specify continuous collection or when snapshots
  are flushed after startup. One startup-time collection cannot replace a
  `/metrics` endpoint. Recommendation: add a ticker-based bridge module or
  lifecycle-managed collector with interval, timeout, and non-fatal diagnostics.
- [Repo-precedent conflicts] Task 2/4: The plan says the external plugin
  provides service name `observability.telemetry.sink`, but does not define the
  host-side adapter that calls the remote module's `ServiceInvoker`. External
  plugin processes cannot register ordinary in-process services by metadata.
  Recommendation: add an explicit host adapter that wraps the remote module or
  its invoker.
- [Security / privacy] Task 3/4/5: Sensitive-key deny-list validation is
  required by the design, but no test asserts redaction/drop behavior for
  metrics/log attributes before sink storage or rendering. Recommendation: add
  tests for default drop of `authorization`, `cookie`, `token`, and `secret`.
- [Verification-class mismatch] Task 7: The plan allows a temporary local
  `replace github.com/GoCodeAlone/workflow => ../workflow` in `go.mod` but does
  not require removing it before commit. Recommendation: use workspace/local
  testing only or add an explicit step that fails if a local replace remains.

**Findings (Minor):**
- [Over-decomposition / under-decomposition] Task 5 combines collector module,
  OTel YAML rendering, env wiring, and ownership labels. Recommendation: during
  execution, split into separate commits even if it remains one plan task.
- [Missing rollback wiring] Task 6 changes new templates but does not say how
  existing generated apps are migrated. Recommendation: explicitly state no
  existing app YAML is rewritten by this task.
- [Unstated assumptions] Task 1 assumes `workflow/plugin/external/sdk` can import
  `workflow/telemetry` without undesirable dependency growth. Recommendation:
  keep `telemetry` dependency-free and enforce it in review.

**Bug-class scan transcript:**

| Class | Result | Note |
|---|---|---|
| Unstated assumptions | Finding | SDK import dependency shape needs to stay tiny and acyclic. |
| Repo-precedent conflicts | Finding | External modules need a host adapter, not in-process service metadata. |
| YAGNI violations | Clean | Scope manifest keeps dashboards, alerts, and managed provisioning out of phase one. |
| Missing failure modes | Finding | Bridge collection cadence/backpressure is unspecified. |
| Security / privacy at architecture level | Finding | PII/sensitive attr filtering lacks tests. |
| Rollback story | Clean | Runtime-affecting tasks include rollback notes. |
| Simpler alternative not considered | Clean | External collector only is explicitly phase-one. |
| User-intent drift | Clean | The plan targets Approach 1 and hard-removes a custom `/metrics` route. |
| Over-decomposition / under-decomposition | Finding | Task 5 is broad but manageable with sub-commits. |
| Verification-class mismatch | Finding | Local replace cleanup needs a hard verification step. |
| Hidden serial dependencies | Clean | Tasks sharing files are grouped sequentially in the same PR rows. |
| Missing rollback wiring | Finding | Template task needs explicit no-rewrite boundary for existing apps. |

**Options the author may not have considered:**

1. Make the bridge a normal core module type rather than a wiring hook. This
   aligns lifecycle/interval behavior with existing modules and avoids hidden
   startup-only collection.
2. Keep the external plugin sink purely pull-based via `snapshot` service calls.
   This simplifies the external module, but the host still needs a lifecycle
   bridge to push or poll on interval.

**Verdict reasoning:** The plan is close, but execution would likely ship a
startup-only bridge and an ambiguous external service boundary. Fix those
Important issues before alignment and execution.

