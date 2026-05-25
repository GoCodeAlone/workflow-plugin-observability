### Adversarial Review Report

**Phase:** design
**Artifact:** docs/plans/2026-05-25-observability-plugin-design.md
**Status:** PASS

**Findings (Critical):**
- None.

**Findings (Important):**
- None.

**Findings (Minor):**
- [Unstated assumptions] Provider and IaC Integration: The shared
  `ObservabilityPlan` owner is written as "workflow or wfctl", which leaves one
  decision for the implementation plan. Recommendation: make the plan choose one
  repo for the first interface and record it in task scope.
- [YAGNI violations] Configuration Sketch: The example still shows four
  exporters at once, which is useful for expressiveness but too broad for the
  first implementation task. Recommendation: first tests should cover OTLP
  receiver plus one exporter, then add matrix cases.
- [Missing failure modes] Distribution Rendering: Renderer validation is named,
  but component availability/version skew for collector distributions is not
  detailed. Recommendation: plan a fixture-based renderer test per supported
  distribution and defer runtime version probing until managed provisioning.

**Bug-class scan transcript:**

| Class | Result | Note |
|---|---|---|
| Unstated assumptions | Finding | The exact home of `ObservabilityPlan` must be selected during planning. |
| Repo-precedent conflicts | Clean | The revised design now uses explicit module/plan contracts and provider-owned translation, which matches local plugin/provider boundaries. |
| YAGNI violations | Finding | The example remains broad, but the new Delivery Phases constrain MVP scope. |
| Missing failure modes | Finding | Renderer version skew needs plan-level test coverage, but no design-level blocker remains. |
| Security / privacy at architecture level | Clean | Private endpoints, explicit public ingest auth, secret references, tenant/resource attrs, and default deny-listed sensitive keys are now included. |
| Rollback story | Clean | Rollback now includes app rollback, collector rollback, inert core interfaces, `/metrics` route rollback, and generated-resource ownership labels. |
| Simpler alternative not considered | Clean | The revised design adds an independently shippable phase-one slice. |
| User-intent drift | Clean | The design stays aligned with Approach 1, selectable collectors, OTel preference, direct Grafana/Prometheus/Loki/Datadog support, and no plugin hard dependency for emitters. |

**Options the author may not have considered:**

1. Make `ObservabilityPlan` initially private to `workflow-plugin-observability`
   and expose it later. This reduces cross-repo churn but delays provider IaC.
2. Start with external collectors only. This is the smallest runtime surface but
   does not prove `wfctl` provisioning, which the user explicitly requested.

**Verdict reasoning:** The revised design addresses the blocking issues from
the first review. Remaining concerns are plan-level choices and test granularity,
not design blockers.

