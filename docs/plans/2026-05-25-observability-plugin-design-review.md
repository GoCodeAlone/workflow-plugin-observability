### Adversarial Review Report

**Phase:** design
**Artifact:** docs/plans/2026-05-25-observability-plugin-design.md
**Status:** FAIL

**Findings (Critical):**
- None.

**Findings (Important):**
- [Missing failure modes] Refactor Scope: The design says custom `/metrics`
  endpoints should be hard-updated, but it does not define the migration gate
  that proves dependent apps have equivalent telemetry before endpoint removal.
  Recommendation: add a migration invariant requiring parity tests or runtime
  smoke evidence for every removed endpoint.
- [Repo-precedent conflicts] Provider and IaC Integration: The design says the
  observability plugin emits provider-neutral deployment intent, but it does not
  name the Workflow/wfctl contract that provider plugins consume. Existing
  provider plugins usually expose explicit module/step schemas and concrete
  resource config. Recommendation: add a specific `ObservabilityPlan` or
  provider-planner contract boundary and state which repo owns it.
- [Security / privacy] Security and Privacy: Public OTLP ingest is mentioned,
  but authn/authz, tenant isolation, and secret redaction are underspecified.
  Recommendation: require private endpoints by default, explicit auth for public
  endpoints, tenant/resource labels, and deny-by-default handling for known PII
  attributes in generated pipelines.
- [YAGNI violations] Observability Plugin Runtime: The optional dashboard and
  alert module types risk expanding MVP scope into backend management before
  collect/export is working. Recommendation: mark these as later phases and
  exclude them from the first implementation plan.

**Findings (Minor):**
- [Unstated assumptions] Assumptions: The design assumes collector config
  generation is enough for Alloy and OTel Collector, but Alloy's River config is
  not the same as OTel Collector YAML. Recommendation: explicitly define an
  intermediate pipeline model with per-distribution renderers.
- [Rollback story] Rollback: Rollback is described at a high level, but not tied
  to generated resource naming or ownership labels. Recommendation: require
  generated resources to be labeled/annotated so provider plugins can delete or
  ignore only plugin-owned resources.
- [Simpler alternative not considered] Alternatives: The design rejects keeping
  everything in core, but does not consider a smaller first slice that only
  creates neutral SDK interfaces and external collector env wiring. Recommendation:
  add phased delivery and make the first phase independently shippable.

**Bug-class scan transcript:**

| Class | Result | Note |
|---|---|---|
| Unstated assumptions | Finding | Alloy/OTel config equivalence and provider contract shape were implicit. |
| Repo-precedent conflicts | Finding | Provider plugins need an explicit schema/contract boundary, not vague intent. |
| YAGNI violations | Finding | Dashboard and alert modules broaden the first deliverable. |
| Missing failure modes | Finding | `/metrics` removal lacks parity and dependent-app migration gates. |
| Security / privacy at architecture level | Finding | Public ingest and tenant isolation are underspecified. |
| Rollback story | Finding | Rollback lacks generated-resource ownership details. |
| Simpler alternative not considered | Finding | A smaller first slice was not documented. |
| User-intent drift | Clean | The design targets the user's requested composable OTel-first observability plugin with direct backend support. |

**Options the author may not have considered:**

1. Two-phase rollout: first ship neutral interfaces plus external collector
   wiring, then add managed collector provisioning. This reduces blast radius
   while preserving the selected architecture.
2. Contract-first provider integration: define a typed `ObservabilityPlan`
   consumed by wfctl/provider plugins before implementing any provider-specific
   collector deployment. This keeps IaC ownership explicit.

**Verdict reasoning:** The design matches the user's chosen approach, but it is
not ready for planning because the provider contract, `/metrics` removal gate,
security boundary, and distribution renderer model are too implicit. Revise
these before writing the implementation plan.

