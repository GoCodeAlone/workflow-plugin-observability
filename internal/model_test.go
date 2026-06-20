package internal

import "testing"

func TestCollectorConfigValidate(t *testing.T) {
	cfg := minimalValidCollectorConfig()
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

func TestCollectorConfigValidateRejectsPublicEndpointWithoutAuth(t *testing.T) {
	cfg := minimalValidCollectorConfig()
	cfg.Exporters["public"] = ExporterConfig{
		Type:     "otlphttp",
		Endpoint: "https://collector.example.com:4318",
		Public:   true,
	}
	cfg.Routes[0].Exporters = append(cfg.Routes[0].Exporters, "public")
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected public exporter auth error")
	}
}

func TestSensitiveAttributeFiltering(t *testing.T) {
	attrs := Attrs{
		"tenant":        "acme",
		"authorization": "Bearer secret",
		"cookie":        "session=secret",
		"token":         "secret",
		"secret":        "value",
	}
	filtered := FilterSensitiveAttrs(attrs, nil)
	if filtered["tenant"] != "acme" {
		t.Fatalf("tenant attr = %q, want acme", filtered["tenant"])
	}
	for _, key := range []string{"authorization", "cookie", "token", "secret"} {
		if _, ok := filtered[key]; ok {
			t.Fatalf("sensitive key %q was not filtered: %#v", key, filtered)
		}
	}
}

func minimalValidCollectorConfig() CollectorConfig {
	return CollectorConfig{
		Name:         "collector",
		Distribution: "otelcol",
		Topology:     "external",
		Signals:      []string{"traces", "metrics", "logs"},
		Receivers: map[string]ReceiverConfig{
			"otlp": {Type: "otlp", Protocols: []string{"grpc", "http"}},
		},
		Exporters: map[string]ExporterConfig{
			"debug": {Type: "debug"},
		},
		Routes: []RouteConfig{
			{Signals: []string{"traces", "metrics", "logs"}, Receivers: []string{"otlp"}, Exporters: []string{"debug"}},
		},
	}
}
