# 0002. Use Neutral Telemetry Interfaces

**Status:** Accepted
**Date:** 2026-05-25
**Decision-makers:** Jon Langevin, Codex
**Related:** docs/plans/2026-05-25-observability-plugin-design.md

## Context

Plugins and modules need to emit custom metrics, logs, and trace annotations
without depending on `workflow-plugin-observability` or assuming that telemetry
will be collected. Go interface satisfaction is the desired model: a module can
offer telemetry behavior, and a collector can discover it when present.

## Decision

Workflow core or the external plugin SDK will define small telemetry producer
and recorder interfaces. Plugins may satisfy these interfaces without importing
the observability plugin. The observability plugin will discover emitters and
adapt records to OTel SDKs, collector config, or direct backend SDK/API paths.

Rejected alternatives: require direct OTel SDK imports in every emitting plugin,
which creates hard dependencies; define interfaces in the observability plugin,
which still forces plugin imports; preserve only HTTP `/metrics` endpoints,
which is too narrow for logs, traces, and provider-neutral IaC.

## Consequences

Custom telemetry becomes easy to add and cheap to ignore when no collector is
installed. The interfaces must be stable and intentionally small, because they
will become a cross-plugin contract. Some advanced OTel features may require
optional escape hatches, but the default path remains backend-neutral.

