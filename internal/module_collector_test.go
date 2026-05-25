package internal

import (
	"strings"
	"testing"
)

func TestCollectorModulePlanExternal(t *testing.T) {
	mod, err := newCollectorModule("collector", map[string]any{
		"distribution": "otelcol",
		"topology":     "external",
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
	if env["OTEL_TRACES_EXPORTER"] != "otlp" || env["OTEL_METRICS_EXPORTER"] != "otlp" || env["OTEL_LOGS_EXPORTER"] != "otlp" {
		t.Fatalf("unexpected OTEL signal exporters: %#v", env)
	}
}

func TestCollectorModuleRenderConfig(t *testing.T) {
	mod, err := newCollectorModule("collector", map[string]any{
		"distribution": "otelcol",
		"topology":     "external",
		"signals":      []any{"traces"},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := mod.InvokeMethod("renderConfig", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out["config"] == "" {
		t.Fatal("missing rendered config")
	}
}

func TestCollectorModuleParsesConfiguredPipeline(t *testing.T) {
	mod, err := newCollectorModule("collector", map[string]any{
		"distribution": "otelcol",
		"topology":     "external",
		"signals":      []any{"metrics"},
		"receivers": map[string]any{
			"otlp": map[string]any{"type": "otlp", "protocols": []any{"http"}},
		},
		"exporters": map[string]any{
			"mimir": map[string]any{"type": "prometheus_remote_write", "endpoint": "https://mimir.example/api/v1/push"},
		},
		"routes": []any{
			map[string]any{"signals": []any{"metrics"}, "receivers": []any{"otlp"}, "exporters": []any{"mimir"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := mod.InvokeMethod("renderConfig", nil)
	if err != nil {
		t.Fatal(err)
	}
	rendered := out["config"].(string)
	if !strings.Contains(rendered, "mimir:") || !strings.Contains(rendered, "https://mimir.example/api/v1/push") {
		t.Fatalf("rendered config did not preserve configured exporter:\n%s", rendered)
	}
}
