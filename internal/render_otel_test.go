package internal

import (
	"strings"
	"testing"
)

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
