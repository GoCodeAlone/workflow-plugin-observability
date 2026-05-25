package internal

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

func RenderOTelCollectorYAML(cfg CollectorConfig) (string, error) {
	cfg = normalizeCollectorConfig(cfg)
	if cfg.Distribution != "otelcol" && cfg.Distribution != "external" {
		return "", fmt.Errorf("otel renderer does not support distribution %q", cfg.Distribution)
	}
	if err := cfg.Validate(); err != nil {
		return "", err
	}

	receivers := make(map[string]any, len(cfg.Receivers))
	for name, receiver := range cfg.Receivers {
		rendered := map[string]any{}
		if receiver.Type == "otlp" {
			protocols := map[string]any{}
			for _, protocol := range receiver.Protocols {
				protocols[protocol] = map[string]any{}
			}
			if len(protocols) == 0 {
				protocols["grpc"] = map[string]any{}
				protocols["http"] = map[string]any{}
			}
			rendered["protocols"] = protocols
		}
		if receiver.Endpoint != "" {
			rendered["endpoint"] = receiver.Endpoint
		}
		receivers[name] = rendered
	}

	processors := map[string]any{}
	for name, processor := range cfg.Processors {
		processors[name] = processor.Config
	}
	if len(processors) == 0 {
		processors["batch"] = map[string]any{}
	}

	exporters := make(map[string]any, len(cfg.Exporters))
	for name, exporter := range cfg.Exporters {
		rendered := map[string]any{}
		if exporter.Endpoint != "" {
			rendered["endpoint"] = exporter.Endpoint
		}
		if len(exporter.Headers) > 0 {
			rendered["headers"] = exporter.Headers
		}
		if exporter.Compression != "" {
			rendered["compression"] = exporter.Compression
		}
		exporters[name] = rendered
	}

	pipelines := map[string]any{}
	for _, route := range cfg.Routes {
		processorsForRoute := route.Processors
		if len(processorsForRoute) == 0 {
			processorsForRoute = []string{"batch"}
		}
		for _, signal := range route.Signals {
			pipelines[signal] = map[string]any{
				"receivers":  route.Receivers,
				"processors": processorsForRoute,
				"exporters":  route.Exporters,
			}
		}
	}

	doc := map[string]any{
		"receivers":  receivers,
		"processors": processors,
		"exporters":  exporters,
		"service": map[string]any{
			"pipelines": pipelines,
		},
	}
	data, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
