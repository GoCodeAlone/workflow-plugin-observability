package internal

import "testing"

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

func TestTelemetryModuleFiltersSensitiveAttrs(t *testing.T) {
	mod, err := newTelemetryModule("telemetry", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = mod.InvokeMethod("recordMetrics", map[string]any{
		"metrics": []any{map[string]any{
			"name":  "requests_total",
			"kind":  "counter",
			"value": 1.0,
			"attrs": map[string]any{
				"tenant":        "acme",
				"authorization": "Bearer secret",
				"cookie":        "secret",
				"token":         "secret",
				"secret":        "secret",
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	attrs := mod.Snapshot().Metrics[0].Attrs
	if attrs["tenant"] != "acme" {
		t.Fatalf("tenant attr = %q, want acme", attrs["tenant"])
	}
	for _, key := range []string{"authorization", "cookie", "token", "secret"} {
		if _, ok := attrs[key]; ok {
			t.Fatalf("sensitive key %q was not filtered: %#v", key, attrs)
		}
	}
}

func TestTelemetryModuleRecordsLogsAndSpanEvents(t *testing.T) {
	mod, err := newTelemetryModule("telemetry", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mod.InvokeMethod("recordLogs", map[string]any{
		"logs": []any{map[string]any{"level": "info", "message": "ok", "module": "cms"}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := mod.InvokeMethod("recordSpanEvents", map[string]any{
		"events": []any{map[string]any{"name": "cache.hit", "attrs": map[string]any{"tenant": "acme"}}},
	}); err != nil {
		t.Fatal(err)
	}
	snapshot := mod.Snapshot()
	if len(snapshot.Logs) != 1 {
		t.Fatalf("snapshot logs = %d, want 1", len(snapshot.Logs))
	}
	if len(snapshot.SpanEvents) != 1 {
		t.Fatalf("snapshot span events = %d, want 1", len(snapshot.SpanEvents))
	}
}
