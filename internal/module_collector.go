package internal

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

const defaultOTLPEndpoint = "http://localhost:4318"

type collectorModule struct {
	name   string
	config CollectorConfig
}

func newCollectorModule(name string, cfg map[string]any) (*collectorModule, error) {
	collector := &collectorModule{
		name:   name,
		config: normalizeCollectorConfig(collectorConfigFromMap(name, cfg)),
	}
	if err := collector.config.Validate(); err != nil {
		return nil, err
	}
	return collector, nil
}

func (m *collectorModule) Init() error {
	return nil
}

func (m *collectorModule) Start(context.Context) error {
	return nil
}

func (m *collectorModule) Stop(context.Context) error {
	return nil
}

func (m *collectorModule) InvokeMethod(method string, args map[string]any) (map[string]any, error) {
	return m.InvokeMethodContext(context.Background(), method, args)
}

func (m *collectorModule) InvokeMethodContext(_ context.Context, method string, _ map[string]any) (map[string]any, error) {
	switch method {
	case "plan":
		return m.plan()
	case "renderConfig":
		return m.renderConfig()
	default:
		return nil, fmt.Errorf("observability.collector %q: unknown method %q", m.name, method)
	}
}

func (m *collectorModule) plan() (map[string]any, error) {
	rendered, err := RenderOTelCollectorYAML(m.config)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"collector": map[string]any{
			"name":         m.name,
			"distribution": m.config.Distribution,
			"topology":     m.config.Topology,
			"config":       rendered,
		},
		"env":       m.appEnv(),
		"resources": m.generatedResources(),
	}, nil
}

func (m *collectorModule) renderConfig() (map[string]any, error) {
	rendered, err := RenderOTelCollectorYAML(m.config)
	if err != nil {
		return nil, err
	}
	return map[string]any{"config": rendered}, nil
}

func (m *collectorModule) appEnv() map[string]string {
	env := map[string]string{
		"OTEL_EXPORTER_OTLP_ENDPOINT": defaultOTLPEndpoint,
		"OTEL_TRACES_EXPORTER":        "otlp",
		"OTEL_METRICS_EXPORTER":       "otlp",
		"OTEL_LOGS_EXPORTER":          "otlp",
	}
	if m.name != "" {
		env["OTEL_SERVICE_NAME"] = m.name
	}
	for _, exporter := range m.config.Exporters {
		if exporter.Endpoint != "" {
			env["OTEL_EXPORTER_OTLP_ENDPOINT"] = exporter.Endpoint
			break
		}
	}
	return env
}

func (m *collectorModule) generatedResources() []map[string]any {
	return []map[string]any{
		{
			"kind": "collector",
			"name": m.name,
			"labels": map[string]string{
				"workflow.gocodealone.io/managed-by": "workflow-plugin-observability",
				"workflow.gocodealone.io/collector":  m.name,
			},
		},
	}
}

func collectorConfigFromMap(name string, cfg map[string]any) CollectorConfig {
	out := CollectorConfig{
		Name:         name,
		Distribution: stringValue(cfg["distribution"]),
		Topology:     stringValue(cfg["topology"]),
		Signals:      stringSliceValue(cfg["signals"]),
	}
	endpoint := stringValue(cfg["endpoint"])
	if endpoint == "" {
		endpoint = stringValue(cfg["otlpEndpoint"])
	}
	if endpoint != "" {
		out.Exporters = map[string]ExporterConfig{
			"otlp": {Type: "otlphttp", Endpoint: endpoint},
		}
	}
	return out
}

func normalizeCollectorConfig(cfg CollectorConfig) CollectorConfig {
	if cfg.Name == "" {
		cfg.Name = "collector"
	}
	cfg.Distribution = defaultString(cfg.Distribution, "external")
	cfg.Topology = defaultString(cfg.Topology, "external")
	if len(cfg.Signals) == 0 {
		cfg.Signals = []string{"traces", "metrics", "logs"}
	}
	if len(cfg.Receivers) == 0 {
		cfg.Receivers = map[string]ReceiverConfig{
			"otlp": {Type: "otlp", Protocols: []string{"grpc", "http"}},
		}
	}
	if len(cfg.Exporters) == 0 {
		cfg.Exporters = map[string]ExporterConfig{
			"otlp": {Type: "otlphttp", Endpoint: defaultOTLPEndpoint},
		}
	}
	if len(cfg.Routes) == 0 {
		cfg.Routes = []RouteConfig{
			{Signals: cfg.Signals, Receivers: sortedKeys(cfg.Receivers), Exporters: sortedKeys(cfg.Exporters)},
		}
	}
	return cfg
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func stringSliceValue(value any) []string {
	switch typed := value.(type) {
	case nil:
		return nil
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, stringValue(item))
		}
		return out
	case string:
		if typed == "" {
			return nil
		}
		return strings.Split(typed, ",")
	default:
		return nil
	}
}
