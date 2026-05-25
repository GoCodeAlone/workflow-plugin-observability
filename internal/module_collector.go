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
		Receivers:    receiverConfigsValue(cfg["receivers"]),
		Processors:   processorConfigsValue(cfg["processors"]),
		Exporters:    exporterConfigsValue(cfg["exporters"]),
		Routes:       routeConfigsValue(cfg["routes"]),
	}
	endpoint := stringValue(cfg["endpoint"])
	if endpoint == "" {
		endpoint = stringValue(cfg["otlpEndpoint"])
	}
	if endpoint != "" && len(out.Exporters) == 0 {
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

func receiverConfigsValue(value any) map[string]ReceiverConfig {
	items := mapValue(value)
	if len(items) == 0 {
		return nil
	}
	out := make(map[string]ReceiverConfig, len(items))
	for name, raw := range items {
		cfg := mapValue(raw)
		out[name] = ReceiverConfig{
			Type:      stringValue(cfg["type"]),
			Endpoint:  stringValue(cfg["endpoint"]),
			Protocols: stringSliceValue(cfg["protocols"]),
			Public:    boolValue(cfg["public"]),
			AuthRef:   stringValue(cfg["authRef"]),
		}
	}
	return out
}

func processorConfigsValue(value any) map[string]ProcessorConfig {
	items := mapValue(value)
	if len(items) == 0 {
		return nil
	}
	out := make(map[string]ProcessorConfig, len(items))
	for name, raw := range items {
		cfg := mapValue(raw)
		out[name] = ProcessorConfig{
			Type:   stringValue(cfg["type"]),
			Config: mapValue(cfg["config"]),
		}
	}
	return out
}

func exporterConfigsValue(value any) map[string]ExporterConfig {
	items := mapValue(value)
	if len(items) == 0 {
		return nil
	}
	out := make(map[string]ExporterConfig, len(items))
	for name, raw := range items {
		cfg := mapValue(raw)
		out[name] = ExporterConfig{
			Type:        stringValue(cfg["type"]),
			Endpoint:    stringValue(cfg["endpoint"]),
			Headers:     stringMapValue(cfg["headers"]),
			APIKeyRef:   stringValue(cfg["apiKeyRef"]),
			AuthRef:     stringValue(cfg["authRef"]),
			Public:      boolValue(cfg["public"]),
			Compression: stringValue(cfg["compression"]),
		}
	}
	return out
}

func routeConfigsValue(value any) []RouteConfig {
	items, err := sliceOfMaps(value)
	if err != nil || len(items) == 0 {
		return nil
	}
	out := make([]RouteConfig, 0, len(items))
	for _, item := range items {
		out = append(out, RouteConfig{
			Signals:    stringSliceValue(item["signals"]),
			Receivers:  stringSliceValue(item["receivers"]),
			Processors: stringSliceValue(item["processors"]),
			Exporters:  stringSliceValue(item["exporters"]),
		})
	}
	return out
}

func mapValue(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[string]string:
		out := make(map[string]any, len(typed))
		for k, v := range typed {
			out[k] = v
		}
		return out
	default:
		return nil
	}
}

func stringMapValue(value any) map[string]string {
	switch typed := value.(type) {
	case map[string]string:
		return typed
	case map[string]any:
		out := make(map[string]string, len(typed))
		for k, v := range typed {
			out[k] = stringValue(v)
		}
		return out
	default:
		return nil
	}
}

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed == "true"
	default:
		return false
	}
}
