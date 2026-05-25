# 0001. Own Observability In A Plugin

**Status:** Accepted
**Date:** 2026-05-25
**Decision-makers:** Jon Langevin, Codex
**Related:** docs/plans/2026-05-25-observability-plugin-design.md

## Context

Workflow needs plug-and-play telemetry that can provision collectors through
`wfctl` and provider IaC while supporting OpenTelemetry, Grafana, Prometheus,
Loki, and Datadog. Core `workflow` already contains OTel tracing and Prometheus
metrics modules, and some provider/application plugins have isolated log or
metrics behavior. Keeping this in core would keep growing backend-specific
deployment assumptions. Moving everything into provider plugins would duplicate
collector semantics and make application YAML less portable.

## Decision

We will create `workflow-plugin-observability` to own collector declarations,
telemetry routing, backend-specific integration intent, and generated collector
configuration. Workflow core will retain only small producer interfaces and
compatibility shims. Provider plugins will translate neutral observability
intent into concrete IaC resources for their platforms.

Rejected alternatives: keep all observability in core because it centralizes too
many backend assumptions; push observability into each provider plugin because
that duplicates semantics and reduces portability.

## Consequences

Telemetry deployment becomes composable and customer-selectable. Core Workflow
stays smaller and plugin authors can avoid backend-specific dependencies. The
cost is a cross-repo contract between core interfaces, the observability plugin,
`wfctl`, and provider plugins. Rollback remains straightforward because the
plugin can be removed from YAML while additive core interfaces remain inert.

