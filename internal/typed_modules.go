package internal

import (
	"fmt"
	"time"

	observabilityv1 "github.com/GoCodeAlone/workflow-plugin-observability/gen/observability/v1"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func (p *observabilityPlugin) TypedModuleTypes() []string {
	return p.ModuleTypes()
}

func (p *observabilityPlugin) CreateTypedModule(typeName, name string, config *anypb.Any) (sdk.ModuleInstance, error) {
	switch typeName {
	case "observability.collector":
		cfg, err := unpackTypedConfig(config, &observabilityv1.CollectorConfig{})
		if err != nil {
			return nil, err
		}
		collector := &collectorModule{
			name:   name,
			config: normalizeCollectorConfig(collectorConfigFromProto(name, cfg)),
		}
		if err := collector.config.Validate(); err != nil {
			return nil, err
		}
		return collector, nil
	case "observability.telemetry":
		cfg, err := unpackTypedConfig(config, &observabilityv1.TelemetryConfig{})
		if err != nil {
			return nil, err
		}
		converted, err := telemetryConfigFromProto(cfg)
		if err != nil {
			return nil, err
		}
		return &telemetryModule{name: name, config: converted}, nil
	default:
		return nil, fmt.Errorf("%w: module type %q", sdk.ErrTypedContractNotHandled, typeName)
	}
}

func unpackTypedConfig[T proto.Message](payload *anypb.Any, target T) (T, error) {
	if payload == nil {
		return target, nil
	}
	expected := target.ProtoReflect().Descriptor().FullName()
	if payload.MessageName() != expected {
		var zero T
		return zero, fmt.Errorf("typed config type mismatch: expected %s, got %s", expected, payload.MessageName())
	}
	if err := payload.UnmarshalTo(target); err != nil {
		var zero T
		return zero, fmt.Errorf("unpack typed config as %s: %w", expected, err)
	}
	return target, nil
}

func telemetryConfigFromProto(cfg *observabilityv1.TelemetryConfig) (TelemetryConfig, error) {
	if cfg == nil {
		return TelemetryConfig{}, nil
	}
	out := TelemetryConfig{
		ServiceName:        cfg.GetServiceName(),
		Environment:        cfg.GetEnvironment(),
		ResourceAttributes: copyStringMap(cfg.GetResource()),
		OTLPEndpoint:       cfg.GetOtlpEndpoint(),
	}
	if interval := cfg.GetMetricsInterval(); interval != "" {
		parsed, err := time.ParseDuration(interval)
		if err != nil {
			return TelemetryConfig{}, fmt.Errorf("parse metricsInterval %q: %w", interval, err)
		}
		out.MetricsInterval = parsed
	}
	return out, nil
}

func collectorConfigFromProto(name string, cfg *observabilityv1.CollectorConfig) CollectorConfig {
	if cfg == nil {
		return CollectorConfig{Name: name}
	}
	out := CollectorConfig{
		Name:         defaultString(cfg.GetName(), name),
		Distribution: cfg.GetDistribution(),
		Topology:     cfg.GetTopology(),
		Signals:      append([]string(nil), cfg.GetSignals()...),
		Receivers:    receiverConfigsFromProto(cfg.GetReceivers()),
		Processors:   processorConfigsFromProto(cfg.GetProcessors()),
		Exporters:    exporterConfigsFromProto(cfg.GetExporters()),
		Routes:       routeConfigsFromProto(cfg.GetRoutes()),
	}
	endpoint := defaultString(cfg.GetEndpoint(), cfg.GetOtlpEndpoint())
	if endpoint != "" && len(out.Exporters) == 0 {
		out.Exporters = map[string]ExporterConfig{
			"otlp": {Type: "otlphttp", Endpoint: endpoint},
		}
	}
	return out
}

func receiverConfigsFromProto(values map[string]*observabilityv1.ReceiverConfig) map[string]ReceiverConfig {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]ReceiverConfig, len(values))
	for name, cfg := range values {
		out[name] = ReceiverConfig{
			Type:      cfg.GetType(),
			Endpoint:  cfg.GetEndpoint(),
			Protocols: append([]string(nil), cfg.GetProtocols()...),
			Public:    cfg.GetPublic(),
			AuthRef:   cfg.GetAuthRef(),
		}
	}
	return out
}

func processorConfigsFromProto(values map[string]*observabilityv1.ProcessorConfig) map[string]ProcessorConfig {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]ProcessorConfig, len(values))
	for name, cfg := range values {
		out[name] = ProcessorConfig{
			Type:   cfg.GetType(),
			Config: structToMap(cfg.GetConfig()),
		}
	}
	return out
}

func exporterConfigsFromProto(values map[string]*observabilityv1.ExporterConfig) map[string]ExporterConfig {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]ExporterConfig, len(values))
	for name, cfg := range values {
		converted := ExporterConfig{
			Type:        cfg.GetType(),
			Endpoint:    cfg.GetEndpoint(),
			Headers:     copyStringMap(cfg.GetHeaders()),
			APIKeyRef:   cfg.GetApiKeyRef(),
			AuthRef:     cfg.GetAuthRef(),
			Public:      cfg.GetPublic(),
			Compression: cfg.GetCompression(),
		}
		if cfg.GetTls() != nil {
			converted.TLS = &TLSConfig{
				Enabled: cfg.GetTls().GetEnabled(),
				CARef:   cfg.GetTls().GetCaRef(),
			}
		}
		out[name] = converted
	}
	return out
}

func routeConfigsFromProto(values []*observabilityv1.RouteConfig) []RouteConfig {
	if len(values) == 0 {
		return nil
	}
	out := make([]RouteConfig, 0, len(values))
	for _, cfg := range values {
		out = append(out, RouteConfig{
			Signals:    append([]string(nil), cfg.GetSignals()...),
			Receivers:  append([]string(nil), cfg.GetReceivers()...),
			Processors: append([]string(nil), cfg.GetProcessors()...),
			Exporters:  append([]string(nil), cfg.GetExporters()...),
		})
	}
	return out
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func structToMap(value *structpb.Struct) map[string]any {
	if value == nil {
		return nil
	}
	return value.AsMap()
}

var _ interface {
	TypedModuleTypes() []string
	CreateTypedModule(string, string, *anypb.Any) (sdk.ModuleInstance, error)
} = (*observabilityPlugin)(nil)
