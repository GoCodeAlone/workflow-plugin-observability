# Observability Plugin Alignment Report

**Status:** PASS

**Programmatic Scope Check:** PASS (`plan-scope-check.sh --plan`)

## Coverage

| Design Requirement | Plan Task(s) | Status |
|---|---|---|
| Create `workflow-plugin-observability` as reusable observability plugin | Task 3, Task 4, Task 5, Task 8 | Covered |
| Prefer OpenTelemetry and OTLP as default app-to-collector contract | Task 5, Task 8 | Covered |
| Support selectable collector runtimes and future Grafana/Datadog paths without hard-coding one runtime | Task 3, Task 5, Task 8 | Covered |
| Support Grafana/Prometheus/Loki/Tempo/Mimir and Datadog integration intent, with OTel-first preference | Task 3, Task 5, Task 8 | Covered |
| Remove custom `/metrics` endpoints where unnecessary behind parity tests | Task 6, Task 7, Task 8 | Covered |
| Let modules/plugins emit telemetry via Workflow-owned interfaces without importing observability plugin | Task 1, Task 2, Task 7, Task 8 | Covered |
| Provide host-side bridge because external plugin processes cannot inspect in-process services | Task 2, Task 4 | Covered |
| Produce provider-neutral observability intent for later wfctl/IaC/provider adapters | Task 3, Task 5, Task 8 | Covered |
| Validate security/privacy constraints including secrets, public endpoint auth, and sensitive attributes | Task 3, Task 4, Task 5, Task 8 | Covered |
| Use deterministic labels/rollback ownership in generated plans | Task 5, Task 8 | Covered |
| Keep managed provisioning, dashboards, and alert authoring out of phase one | Scope Manifest, Task 8 docs | Covered |

## Scope Check

| Plan Task | Design Requirement | Status |
|---|---|---|
| Task 1 | Workflow core/SDK neutral producer contracts | Justified |
| Task 2 | Host-side telemetry bridge and no-op/sink behavior | Justified |
| Task 3 | Observability plugin collector/pipeline model and validation | Justified |
| Task 4 | Observability plugin telemetry sink service boundary | Justified |
| Task 5 | OTel Collector rendering, app env wiring, ownership labels | Justified |
| Task 6 | New wfctl templates prefer observability over `metrics.collector` | Justified |
| Task 7 | Hard update CMS `/metrics` endpoint to neutral telemetry | Justified |
| Task 8 | Smoke fixture, docs, rollback, and final verification | Justified |

## Drift Items

None.
