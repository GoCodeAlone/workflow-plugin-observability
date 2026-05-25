# Observability Plugin Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the first shippable observability slice: neutral Workflow telemetry interfaces, a host-side bridge, a new observability plugin with selectable collector configuration/rendering, and removal of one custom `/metrics` endpoint behind parity tests.

**Architecture:** Workflow core owns producer interfaces and the in-process bridge because external plugins cannot inspect the host service registry directly. `workflow-plugin-observability` owns telemetry sinks, collector/pipeline validation, OTel Collector rendering, and external collector wiring. Provider-specific provisioning is represented as typed plan data in phase one, with concrete provider adapters deferred to later PRs.

**Tech Stack:** Go 1.26, Workflow external plugin SDK, Modular modules, YAML rendering, OpenTelemetry Collector config model, existing Workflow test conventions.

**Base branch:** main

---

## Scope Manifest

**PR Count:** 4
**Tasks:** 8
**Estimated Lines of Change:** ~2200

**Out of scope:**
- Managed Kubernetes/DigitalOcean/AWS/GCP/Azure collector provisioning.
- Grafana dashboard authoring and Datadog monitor authoring.
- Runtime probing of installed collector binary component versions.
- Removing every historical `metrics.collector` example in `workflow/example/**`.

**PR Grouping:**

| PR # | Title | Tasks | Branch |
|------|-------|-------|--------|
| 1 | Add neutral Workflow telemetry contracts | Task 1, Task 2 | feat/observability-core-contracts |
| 2 | Add observability plugin phase-one runtime | Task 3, Task 4, Task 5 | feat/observability-plugin-runtime |
| 3 | Prefer observability telemetry in wfctl templates | Task 6 | feat/observability-wfctl-templates |
| 4 | Remove CMS custom metrics endpoint | Task 7, Task 8 | feat/observability-cms-metrics |

**Status:** Draft

### Task 1: Add Neutral Telemetry Contracts

**Files:**
- Create: `/Users/jon/workspace/workflow/telemetry/contracts.go`
- Create: `/Users/jon/workspace/workflow/telemetry/contracts_test.go`
- Modify: `/Users/jon/workspace/workflow/plugin/external/sdk/interfaces.go`

**Step 1: Write the failing tests**

Create `workflow/telemetry/contracts_test.go` with compile-time and behavior tests:

```go
package telemetry

import (
	"context"
	"testing"
	"time"
)

type testMetricEmitter struct{}

func (testMetricEmitter) EmitMetrics(_ context.Context, r MetricRecorder) error {
	r.Counter("requests_total", 2, Attrs{"tenant": "acme"})
	r.Gauge("active_sessions", 3, nil)
	r.Histogram("request_duration_seconds", 0.15, Attrs{"route": "/"})
	return nil
}

func TestSnapshotRecorderCapturesMetrics(t *testing.T) {
	rec := NewSnapshotRecorder()
	if err := (testMetricEmitter{}).EmitMetrics(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	got := rec.Metrics()
	if len(got) != 3 {
		t.Fatalf("metric count = %d, want 3", len(got))
	}
	if got[0].Name != "requests_total" || got[0].Kind != MetricCounter || got[0].Value != 2 {
		t.Fatalf("first metric = %#v", got[0])
	}
	if got[0].Attrs["tenant"] != "acme" {
		t.Fatalf("tenant attr = %q, want acme", got[0].Attrs["tenant"])
	}
}

func TestLogRecordDefaults(t *testing.T) {
	now := time.Now()
	rec := LogRecord{Timestamp: now, Level: "info", Message: "ok"}
	if rec.Timestamp.IsZero() || rec.Level != "info" || rec.Message != "ok" {
		t.Fatalf("bad log record: %#v", rec)
	}
}
```

Add compile-time SDK assertions in a new test section or `contracts_test.go` that imports `github.com/GoCodeAlone/workflow/plugin/external/sdk` and proves the public SDK aliases match the core contract types.

**Step 2: Run tests to verify they fail**

Run: `GOWORK=off go test ./telemetry ./plugin/external/sdk`

Expected: FAIL with missing package/types such as `undefined: MetricRecorder`.

**Step 3: Implement contracts**

Create `workflow/telemetry/contracts.go`:

```go
package telemetry

import (
	"context"
	"sync"
	"time"
)

type Attrs map[string]string

type MetricKind string

const (
	MetricCounter   MetricKind = "counter"
	MetricGauge     MetricKind = "gauge"
	MetricHistogram MetricKind = "histogram"
)

type MetricRecord struct {
	Name      string
	Kind      MetricKind
	Value     float64
	Attrs     Attrs
	Timestamp time.Time
}

type MetricRecorder interface {
	Counter(name string, value float64, attrs Attrs)
	Gauge(name string, value float64, attrs Attrs)
	Histogram(name string, value float64, attrs Attrs)
}

type MetricEmitter interface {
	EmitMetrics(context.Context, MetricRecorder) error
}

type LogRecord struct {
	Timestamp time.Time
	Level     string
	Message   string
	Module    string
	Attrs     Attrs
}

type LogEmitter interface {
	DrainTelemetryLogs(context.Context) []LogRecord
}

type SpanEvent struct {
	Name      string
	Attrs     Attrs
	Timestamp time.Time
}

type SpanRecorder interface {
	Event(name string, attrs Attrs)
	Attribute(key, value string)
}

type TraceAnnotator interface {
	AnnotateSpan(context.Context, SpanRecorder)
}

type SnapshotRecorder struct {
	mu      sync.Mutex
	metrics []MetricRecord
}

func NewSnapshotRecorder() *SnapshotRecorder { return &SnapshotRecorder{} }

func (r *SnapshotRecorder) Counter(name string, value float64, attrs Attrs) {
	r.record(MetricCounter, name, value, attrs)
}

func (r *SnapshotRecorder) Gauge(name string, value float64, attrs Attrs) {
	r.record(MetricGauge, name, value, attrs)
}

func (r *SnapshotRecorder) Histogram(name string, value float64, attrs Attrs) {
	r.record(MetricHistogram, name, value, attrs)
}

func (r *SnapshotRecorder) Metrics() []MetricRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]MetricRecord, len(r.metrics))
	copy(out, r.metrics)
	return out
}

func (r *SnapshotRecorder) record(kind MetricKind, name string, value float64, attrs Attrs) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := make(Attrs, len(attrs))
	for k, v := range attrs {
		copied[k] = v
	}
	r.metrics = append(r.metrics, MetricRecord{
		Name:      name,
		Kind:      kind,
		Value:     value,
		Attrs:     copied,
		Timestamp: time.Now(),
	})
}
```

Expose SDK aliases in `workflow/plugin/external/sdk/interfaces.go`:

```go
type TelemetryAttrs = telemetry.Attrs
type TelemetryMetricRecord = telemetry.MetricRecord
type TelemetryMetricRecorder = telemetry.MetricRecorder
type TelemetryMetricEmitter = telemetry.MetricEmitter
type TelemetryLogRecord = telemetry.LogRecord
type TelemetryLogEmitter = telemetry.LogEmitter
type TelemetrySpanEvent = telemetry.SpanEvent
type TelemetrySpanRecorder = telemetry.SpanRecorder
type TelemetryTraceAnnotator = telemetry.TraceAnnotator
```

**Step 4: Run tests to verify they pass**

Run: `GOWORK=off go test ./telemetry ./plugin/external/sdk`

Expected: PASS.

**Step 5: Commit**

```bash
git -C /Users/jon/workspace/workflow add telemetry plugin/external/sdk/interfaces.go
git -C /Users/jon/workspace/workflow commit -m "feat: add neutral telemetry contracts"
```

Rollback: revert this commit; no runtime paths depend on the new additive interfaces yet.

### Task 2: Add Host-Side Telemetry Bridge

**Files:**
- Create: `/Users/jon/workspace/workflow/telemetry/bridge.go`
- Create: `/Users/jon/workspace/workflow/telemetry/bridge_test.go`
- Modify: `/Users/jon/workspace/workflow/plugins/observability/wiring.go`
- Test: `/Users/jon/workspace/workflow/plugins/observability/plugin_test.go`

**Step 1: Write the failing tests**

Create bridge tests proving discovery is in-process and sink failures are non-fatal:

```go
func TestBridgeCollectsMetricEmitters(t *testing.T) {
	app := newTestAppWithServices(map[string]any{"emitter": testMetricEmitter{}})
	sink := &recordingSink{}
	bridge := NewBridge(sink, BridgeConfig{Timeout: time.Second})
	if err := bridge.Collect(context.Background(), app); err != nil {
		t.Fatal(err)
	}
	if len(sink.metrics) != 3 {
		t.Fatalf("metric count = %d, want 3", len(sink.metrics))
	}
}

func TestBridgeSinkFailureIsDiagnostic(t *testing.T) {
	app := newTestAppWithServices(map[string]any{"emitter": testMetricEmitter{}})
	bridge := NewBridge(failingSink{}, BridgeConfig{Timeout: time.Second})
	err := bridge.Collect(context.Background(), app)
	if err == nil {
		t.Fatal("expected diagnostic error")
	}
}

func TestNoopSinkKeepsEmittersInert(t *testing.T) {
	app := newTestAppWithServices(map[string]any{"emitter": testMetricEmitter{}})
	bridge := NewBridge(NoopSink{}, BridgeConfig{Timeout: time.Second})
	if err := bridge.Collect(context.Background(), app); err != nil {
		t.Fatal(err)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `GOWORK=off go test ./telemetry ./plugins/observability`

Expected: FAIL with missing `NewBridge`, `NoopSink`, or wiring assertions.

**Step 3: Implement bridge**

Implement:

- `TelemetrySink` interface with `RecordMetrics`, `RecordLogs`, `RecordSpanEvents`.
- `BridgeConfig{Timeout time.Duration}` with default 2 seconds.
- `Bridge.Collect(ctx, app modular.Application) error` that loops over `app.SvcRegistry()`, calls `MetricEmitter` and `LogEmitter`, batches records, and forwards to sink with context timeout.
- `NoopSink` for absent plugin.

In `workflow/plugins/observability/wiring.go`, add a low-priority hook named `observability.telemetry-bridge` that only runs when a service named `observability.telemetry.sink` exists; otherwise it installs or uses `NoopSink`. Keep it non-fatal for application startup and log diagnostics through the app logger.

**Step 4: Run tests to verify they pass**

Run: `GOWORK=off go test ./telemetry ./plugins/observability`

Expected: PASS.

**Step 5: Commit**

```bash
git -C /Users/jon/workspace/workflow add telemetry plugins/observability
git -C /Users/jon/workspace/workflow commit -m "feat: add telemetry bridge"
```

Rollback: revert this commit; bridge defaults to no-op and does not change startup config unless wired.

### Task 3: Add Observability Plugin Models And Validation

**Files:**
- Create: `/Users/jon/workspace/workflow-plugin-observability/internal/model.go`
- Create: `/Users/jon/workspace/workflow-plugin-observability/internal/model_test.go`
- Modify: `/Users/jon/workspace/workflow-plugin-observability/internal/plugin.go`
- Modify: `/Users/jon/workspace/workflow-plugin-observability/plugin.json`

**Step 1: Write the failing tests**

Add table tests for collector validation:

```go
func TestCollectorConfigValidate(t *testing.T) {
	cfg := CollectorConfig{
		Distribution: "otelcol",
		Topology:     "external",
		Signals:      []string{"traces", "metrics", "logs"},
		Receivers: map[string]ReceiverConfig{
			"otlp": {Type: "otlp", Protocols: []string{"grpc", "http"}},
		},
		Exporters: map[string]ExporterConfig{
			"debug": {Type: "debug"},
		},
		Routes: []RouteConfig{{Signals: []string{"traces"}, Receivers: []string{"otlp"}, Exporters: []string{"debug"}}},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestCollectorConfigValidateRejectsUnknownRouteExporter(t *testing.T) {
	cfg := minimalValidCollectorConfig()
	cfg.Routes[0].Exporters = []string{"missing"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing exporter error")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `GOWORK=off go test ./internal -run 'TestCollectorConfigValidate'`

Expected: FAIL with missing config types.

**Step 3: Implement models**

Implement typed config structs:

- `TelemetryConfig`
- `CollectorConfig`
- `ReceiverConfig`
- `ProcessorConfig`
- `ExporterConfig`
- `RouteConfig`
- `ObservabilityPlan`
- `GeneratedResourceRef`

Validation must check:

- distribution in `otelcol`, `alloy`, `datadog-agent`, `datadog-otlp`, `external`
- topology in `sidecar`, `service`, `daemonset`, `app-component`, `external`
- signals in `traces`, `metrics`, `logs`
- every route references declared receivers/exporters
- public endpoints require auth fields
- default sensitive key deny-list is present

**Step 4: Run tests to verify they pass**

Run: `GOWORK=off go test ./internal -run 'TestCollectorConfigValidate'`

Expected: PASS.

**Step 5: Commit**

```bash
git -C /Users/jon/workspace/workflow-plugin-observability add internal plugin.json
git -C /Users/jon/workspace/workflow-plugin-observability commit -m "feat: add observability config model"
```

Rollback: revert this commit; plugin still contains only scaffold behavior.

### Task 4: Add Telemetry Sink Module

**Files:**
- Create: `/Users/jon/workspace/workflow-plugin-observability/internal/module_telemetry.go`
- Create: `/Users/jon/workspace/workflow-plugin-observability/internal/module_telemetry_test.go`
- Modify: `/Users/jon/workspace/workflow-plugin-observability/internal/plugin.go`
- Modify: `/Users/jon/workspace/workflow-plugin-observability/plugin.json`

**Step 1: Write the failing tests**

Test service invocation for sink methods:

```go
func TestTelemetryModuleRecordsMetrics(t *testing.T) {
	mod, err := newTelemetryModule("telemetry", map[string]any{"serviceName": "checkout"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := mod.InvokeMethod("recordMetrics", map[string]any{
		"metrics": []any{map[string]any{"name": "requests_total", "kind": "counter", "value": 1.0}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out["accepted"] != true {
		t.Fatalf("accepted = %v, want true", out["accepted"])
	}
	if len(mod.Snapshot().Metrics) != 1 {
		t.Fatalf("snapshot metrics = %d, want 1", len(mod.Snapshot().Metrics))
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `GOWORK=off go test ./internal -run 'TestTelemetryModule'`

Expected: FAIL with missing `newTelemetryModule`.

**Step 3: Implement telemetry module**

Implement module type `observability.telemetry`:

- Implements `sdk.ModuleInstance` and `sdk.ServiceInvoker`.
- Accepts service/env/resource/log/trace/metric config.
- Stores received neutral records in an in-memory sink for tests and phase-one smoke checks.
- Exposes methods `recordMetrics`, `recordLogs`, and `snapshot`.
- Provides service name `observability.telemetry.sink` through module schema metadata.

Update plugin module registration and `plugin.json` capabilities.

**Step 4: Run tests to verify they pass**

Run: `GOWORK=off go test ./internal -run 'TestTelemetryModule'`

Expected: PASS.

**Step 5: Commit**

```bash
git -C /Users/jon/workspace/workflow-plugin-observability add internal plugin.json plugin.contracts.json
git -C /Users/jon/workspace/workflow-plugin-observability commit -m "feat: add telemetry sink module"
```

Rollback: revert this commit; remove `observability.telemetry` from app YAML.

### Task 5: Add OTel Collector Renderer And External Env Wiring

**Files:**
- Create: `/Users/jon/workspace/workflow-plugin-observability/internal/render_otel.go`
- Create: `/Users/jon/workspace/workflow-plugin-observability/internal/render_otel_test.go`
- Create: `/Users/jon/workspace/workflow-plugin-observability/internal/module_collector.go`
- Create: `/Users/jon/workspace/workflow-plugin-observability/internal/module_collector_test.go`
- Modify: `/Users/jon/workspace/workflow-plugin-observability/internal/plugin.go`
- Modify: `/Users/jon/workspace/workflow-plugin-observability/plugin.json`

**Step 1: Write the failing tests**

Test renderer output and env wiring:

```go
func TestRenderOTelCollectorYAML(t *testing.T) {
	cfg := minimalValidCollectorConfig()
	got, err := RenderOTelCollectorYAML(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"receivers:",
		"otlp:",
		"exporters:",
		"service:",
		"pipelines:",
		"traces:",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered config missing %q:\n%s", want, got)
		}
	}
}

func TestCollectorModulePlanExternal(t *testing.T) {
	mod, err := newCollectorModule("collector", map[string]any{
		"distribution": "otelcol",
		"topology": "external",
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := mod.InvokeMethod("plan", nil)
	if err != nil {
		t.Fatal(err)
	}
	env := out["env"].(map[string]string)
	if env["OTEL_EXPORTER_OTLP_ENDPOINT"] == "" {
		t.Fatal("missing OTEL_EXPORTER_OTLP_ENDPOINT")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `GOWORK=off go test ./internal -run 'Test(RenderOTel|CollectorModule)'`

Expected: FAIL with missing renderer/module.

**Step 3: Implement renderer and collector module**

Implement:

- `RenderOTelCollectorYAML(CollectorConfig) (string, error)` using `gopkg.in/yaml.v3` or `go.yaml.in/yaml/v2` already available.
- Deterministic map ordering in tests by rendering stable structs where possible.
- `observability.collector` module with `plan` and `renderConfig` service methods.
- `AppEnv` output with `OTEL_SERVICE_NAME`, `OTEL_RESOURCE_ATTRIBUTES`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_TRACES_EXPORTER`, `OTEL_METRICS_EXPORTER`, and `OTEL_LOGS_EXPORTER`.
- Ownership labels in generated plan resources.

**Step 4: Run tests to verify they pass**

Run: `GOWORK=off go test ./internal -run 'Test(RenderOTel|CollectorModule)'`

Expected: PASS.

**Step 5: Commit**

```bash
git -C /Users/jon/workspace/workflow-plugin-observability add internal plugin.json plugin.contracts.json
git -C /Users/jon/workspace/workflow-plugin-observability commit -m "feat: render otel collector plans"
```

Rollback: remove `observability.collector` from YAML and revert generated collector config.

### Task 6: Update wfctl Templates To Prefer Observability Telemetry

**Files:**
- Modify: `/Users/jon/workspace/workflow/cmd/wfctl/templates/api-service/workflow.yaml.tmpl`
- Modify: `/Users/jon/workspace/workflow/cmd/wfctl/templates/full-stack/workflow.yaml.tmpl`
- Modify: `/Users/jon/workspace/workflow/cmd/wfctl/type_registry.go`
- Modify: `/Users/jon/workspace/workflow/cmd/wfctl/type_registry_test.go`
- Modify: `/Users/jon/workspace/workflow/cmd/wfctl/docs_test.go`

**Step 1: Write the failing tests**

Update template tests to expect `observability.telemetry` and optional `observability.collector`, not a default `metrics.collector` module:

```go
func TestAPIServiceTemplateUsesObservabilityTelemetry(t *testing.T) {
	rendered := renderTemplate(t, "api-service", map[string]any{"Name": "orders"})
	if !strings.Contains(rendered, "type: observability.telemetry") {
		t.Fatalf("template missing observability.telemetry:\n%s", rendered)
	}
	if strings.Contains(rendered, "type: metrics.collector") {
		t.Fatalf("template still creates metrics.collector:\n%s", rendered)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `GOWORK=off go test ./cmd/wfctl -run 'Test.*Template|TestTypeRegistry'`

Expected: FAIL because templates still include `metrics.collector` and registry lacks new types.

**Step 3: Implement template and registry updates**

Update generated templates:

```yaml
  - name: {{.Name}}-telemetry
    type: observability.telemetry
    config:
      serviceName: "{{.Name}}"
      environment: "local"

  - name: {{.Name}}-collector
    type: observability.collector
    config:
      distribution: external
      topology: external
      signals: [traces, metrics, logs]
```

Add `observability.telemetry` and `observability.collector` to type registry as observability modules. Keep `metrics.collector` registered for backwards compatibility and mark docs/deprecation text where the registry supports descriptions.

**Step 4: Run tests to verify they pass**

Run: `GOWORK=off go test ./cmd/wfctl -run 'Test.*Template|TestTypeRegistry|TestDocs'`

Expected: PASS.

**Step 5: Commit**

```bash
git -C /Users/jon/workspace/workflow add cmd/wfctl
git -C /Users/jon/workspace/workflow commit -m "feat: prefer observability telemetry in templates"
```

Rollback: revert this commit to restore `metrics.collector` in new templates.

### Task 7: Migrate CMS Metrics To Neutral Telemetry

**Files:**
- Modify: `/Users/jon/workspace/workflow-plugin-cms/host/server.go`
- Modify: `/Users/jon/workspace/workflow-plugin-cms/host/integration_test.go`
- Modify: `/Users/jon/workspace/workflow-plugin-cms/host/server_test.go`
- Delete: `/Users/jon/workspace/workflow-plugin-cms/monitoring/metrics.go`
- Delete: `/Users/jon/workspace/workflow-plugin-cms/monitoring/metrics_test.go`
- Modify: `/Users/jon/workspace/workflow-plugin-cms/go.mod`

**Step 1: Write parity tests before deleting endpoint**

Add a test proving the old counter semantics are emitted through `telemetry.MetricEmitter`:

```go
func TestServerEmitMetrics_Parity(t *testing.T) {
	s := New(Config{TenantResolverStore: resolverWithTenant("acme.test", "acme")})
	req := httptest.NewRequest("GET", "http://acme.test/", nil)
	req.Host = "acme.test"
	s.ServeHTTP(httptest.NewRecorder(), req)

	rec := telemetry.NewSnapshotRecorder()
	if err := s.EmitMetrics(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	metrics := rec.Metrics()
	assertMetric(t, metrics, "cms_requests_total", 1, telemetry.Attrs{"tenant": "acme"})
	assertMetric(t, metrics, "cms_requests_total", 1, telemetry.Attrs{"tenant": "_global"})
}
```

Add a route test:

```go
func TestServerMetricsEndpointRemoved(t *testing.T) {
	s := New(Config{})
	rec := doReq(t, s, "GET", "/metrics", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("/metrics status = %d, want 404", rec.Code)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `GOWORK=off go test ./host ./monitoring`

Expected: FAIL because `EmitMetrics` is missing and `/metrics` still returns 200.

**Step 3: Implement neutral metrics and remove endpoint**

In `host/server.go`:

- Replace `monitoring.Counters` with an internal counter struct in `host` or use simple maps protected by the server mutex.
- Add `EmitMetrics(context.Context, telemetry.MetricRecorder) error` on `*Server`.
- Remove `/metrics` route handling.
- Remove `Metrics()` if only tests used it; otherwise deprecate it and return a snapshot-only helper not tied to HTTP.
- Delete `monitoring` package.

Update `go.mod` to depend on local/new Workflow version during development. If this is not yet published, use a temporary local replace while testing:

```go
replace github.com/GoCodeAlone/workflow => ../workflow
```

Remove the replace before release if a tagged Workflow version exists.

**Step 4: Run tests to verify they pass**

Run: `GOWORK=off go test ./host ./...`

Expected: PASS; `/metrics` route returns 404 and parity metrics are emitted through `EmitMetrics`.

**Step 5: Commit**

```bash
git -C /Users/jon/workspace/workflow-plugin-cms add .
git -C /Users/jon/workspace/workflow-plugin-cms commit -m "feat: emit cms metrics through telemetry"
```

Rollback: revert this commit to restore the prior `/metrics` route while investigating telemetry migration.

### Task 8: End-To-End Smoke And Documentation

**Files:**
- Create: `/Users/jon/workspace/workflow-plugin-observability/examples/external-collector/workflow.yaml`
- Create: `/Users/jon/workspace/workflow-plugin-observability/README.md`
- Modify: `/Users/jon/workspace/workflow-plugin-observability/CLAUDE.md`
- Modify: `/Users/jon/workspace/workflow-plugin-observability/docs/plans/2026-05-25-observability-plugin.md`

**Step 1: Write smoke fixture**

Create an example Workflow YAML with `observability.telemetry` and `observability.collector` using `distribution: external`, plus expected env output in comments.

**Step 2: Run representative commands**

Run:

```bash
GOWORK=off go test ./...                         # in workflow-plugin-observability
GOWORK=off go test ./telemetry ./plugins/observability ./cmd/wfctl # in workflow
GOWORK=off go test ./...                         # in workflow-plugin-cms
```

Expected: all PASS.

**Step 3: Launch validation**

Run:

```bash
GOWORK=off go build ./cmd/workflow-plugin-observability
./workflow-plugin-observability --help
```

Expected: binary builds; help or plugin SDK startup output exits 0 or prints documented plugin serve usage. If the plugin binary is long-running by design, run it with the SDK's documented metadata/handshake command instead of leaving it running.

**Step 4: Document usage and rollback**

README must show:

- Minimal YAML for external collector.
- OTel-first recommendation.
- Direct Grafana/Prometheus/Loki/Datadog support status and phase labels.
- How modules emit custom telemetry by satisfying interfaces.
- Rollback: remove observability modules and redeploy; generated resources are ownership-labeled.

**Step 5: Commit**

```bash
git -C /Users/jon/workspace/workflow-plugin-observability add README.md CLAUDE.md examples docs/plans
git -C /Users/jon/workspace/workflow-plugin-observability commit -m "docs: add observability smoke example"
```

Rollback: remove the example/docs commit; no runtime behavior changes.

## Final Verification

Run these from the workspace:

```bash
git -C /Users/jon/workspace/workflow status --short
git -C /Users/jon/workspace/workflow-plugin-observability status --short
git -C /Users/jon/workspace/workflow-plugin-cms status --short
GOWORK=off go test ./telemetry ./plugins/observability ./cmd/wfctl
cd /Users/jon/workspace/workflow-plugin-observability && GOWORK=off go test ./...
cd /Users/jon/workspace/workflow-plugin-cms && GOWORK=off go test ./...
```

Expected:

- Only intentional files are modified.
- All listed tests pass.
- `workflow-plugin-cms` has no `/metrics` route or `monitoring` package.
- New templates create `observability.telemetry`, not `metrics.collector`.

