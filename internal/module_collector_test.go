package internal

import "testing"

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
