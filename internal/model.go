package internal

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type Attrs map[string]string

type TelemetryConfig struct {
	ServiceName        string            `json:"serviceName" yaml:"serviceName"`
	Environment        string            `json:"environment" yaml:"environment"`
	ResourceAttributes map[string]string `json:"resource" yaml:"resource"`
	MetricsInterval    time.Duration     `json:"metricsInterval" yaml:"metricsInterval"`
	OTLPEndpoint       string            `json:"otlpEndpoint" yaml:"otlpEndpoint"`
}

type CollectorConfig struct {
	Name         string                     `json:"name" yaml:"name"`
	Distribution string                     `json:"distribution" yaml:"distribution"`
	Topology     string                     `json:"topology" yaml:"topology"`
	Signals      []string                   `json:"signals" yaml:"signals"`
	Receivers    map[string]ReceiverConfig  `json:"receivers" yaml:"receivers"`
	Processors   map[string]ProcessorConfig `json:"processors" yaml:"processors"`
	Exporters    map[string]ExporterConfig  `json:"exporters" yaml:"exporters"`
	Routes       []RouteConfig              `json:"routes" yaml:"routes"`
}

type ReceiverConfig struct {
	Type      string   `json:"type" yaml:"type"`
	Endpoint  string   `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Protocols []string `json:"protocols,omitempty" yaml:"protocols,omitempty"`
	Public    bool     `json:"public,omitempty" yaml:"public,omitempty"`
	AuthRef   string   `json:"authRef,omitempty" yaml:"authRef,omitempty"`
}

type ProcessorConfig struct {
	Type   string         `json:"type" yaml:"type"`
	Config map[string]any `json:"config,omitempty" yaml:"config,omitempty"`
}

type ExporterConfig struct {
	Type        string            `json:"type" yaml:"type"`
	Endpoint    string            `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Headers     map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	APIKeyRef   string            `json:"apiKeyRef,omitempty" yaml:"apiKeyRef,omitempty"`
	AuthRef     string            `json:"authRef,omitempty" yaml:"authRef,omitempty"`
	Public      bool              `json:"public,omitempty" yaml:"public,omitempty"`
	TLS         *TLSConfig        `json:"tls,omitempty" yaml:"tls,omitempty"`
	Compression string            `json:"compression,omitempty" yaml:"compression,omitempty"`
}

type TLSConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	CARef   string `json:"caRef,omitempty" yaml:"caRef,omitempty"`
}

type RouteConfig struct {
	Signals    []string `json:"signals" yaml:"signals"`
	Receivers  []string `json:"receivers" yaml:"receivers"`
	Processors []string `json:"processors,omitempty" yaml:"processors,omitempty"`
	Exporters  []string `json:"exporters" yaml:"exporters"`
}

type ObservabilityPlan struct {
	ServiceName   string                 `json:"serviceName" yaml:"serviceName"`
	Environment   string                 `json:"environment" yaml:"environment"`
	ResourceAttrs map[string]string      `json:"resourceAttrs" yaml:"resourceAttrs"`
	Collector     CollectorPlan          `json:"collector" yaml:"collector"`
	Pipelines     []TelemetryPipeline    `json:"pipelines" yaml:"pipelines"`
	AppEnv        map[string]SecretValue `json:"appEnv" yaml:"appEnv"`
	Resources     []GeneratedResourceRef `json:"resources" yaml:"resources"`
}

type CollectorPlan struct {
	Name         string `json:"name" yaml:"name"`
	Distribution string `json:"distribution" yaml:"distribution"`
	Topology     string `json:"topology" yaml:"topology"`
	Config       string `json:"config,omitempty" yaml:"config,omitempty"`
}

type TelemetryPipeline struct {
	Signals   []string `json:"signals" yaml:"signals"`
	Receivers []string `json:"receivers" yaml:"receivers"`
	Exporters []string `json:"exporters" yaml:"exporters"`
}

type SecretValue struct {
	Value     string `json:"value,omitempty" yaml:"value,omitempty"`
	SecretRef string `json:"secretRef,omitempty" yaml:"secretRef,omitempty"`
}

type GeneratedResourceRef struct {
	Kind   string            `json:"kind" yaml:"kind"`
	Name   string            `json:"name" yaml:"name"`
	Labels map[string]string `json:"labels" yaml:"labels"`
}

var (
	validDistributions = []string{"otelcol", "alloy", "datadog-agent", "datadog-otlp", "external"}
	validTopologies    = []string{"sidecar", "service", "daemonset", "app-component", "external"}
	validSignals       = []string{"traces", "metrics", "logs"}
)

var defaultSensitiveKeys = map[string]struct{}{
	"authorization": {},
	"cookie":        {},
	"password":      {},
	"secret":        {},
	"set-cookie":    {},
	"token":         {},
}

func (c CollectorConfig) Validate() error {
	distribution := defaultString(c.Distribution, "external")
	if !slices.Contains(validDistributions, distribution) {
		return fmt.Errorf("invalid collector distribution %q", c.Distribution)
	}
	topology := defaultString(c.Topology, "external")
	if !slices.Contains(validTopologies, topology) {
		return fmt.Errorf("invalid collector topology %q", c.Topology)
	}
	if len(c.Signals) == 0 {
		return fmt.Errorf("collector signals are required")
	}
	for _, signal := range c.Signals {
		if !slices.Contains(validSignals, signal) {
			return fmt.Errorf("invalid collector signal %q", signal)
		}
	}
	if len(c.Routes) == 0 {
		return fmt.Errorf("at least one telemetry route is required")
	}

	for name, receiver := range c.Receivers {
		if receiver.Type == "" {
			return fmt.Errorf("receiver %q type is required", name)
		}
		if receiver.Public && receiver.AuthRef == "" {
			return fmt.Errorf("public receiver %q requires authRef", name)
		}
	}
	for name, exporter := range c.Exporters {
		if exporter.Type == "" {
			return fmt.Errorf("exporter %q type is required", name)
		}
		if exporter.Public && exporter.APIKeyRef == "" && exporter.AuthRef == "" {
			return fmt.Errorf("public exporter %q requires apiKeyRef or authRef", name)
		}
	}
	for i, route := range c.Routes {
		if err := c.validateRoute(i, route); err != nil {
			return err
		}
	}
	return nil
}

func (c CollectorConfig) validateRoute(index int, route RouteConfig) error {
	if len(route.Signals) == 0 {
		return fmt.Errorf("route %d signals are required", index)
	}
	for _, signal := range route.Signals {
		if !slices.Contains(validSignals, signal) {
			return fmt.Errorf("route %d invalid signal %q", index, signal)
		}
	}
	if len(route.Receivers) == 0 {
		return fmt.Errorf("route %d receivers are required", index)
	}
	for _, name := range route.Receivers {
		if _, ok := c.Receivers[name]; !ok {
			return fmt.Errorf("route %d references unknown receiver %q", index, name)
		}
	}
	for _, name := range route.Processors {
		if _, ok := c.Processors[name]; !ok {
			return fmt.Errorf("route %d references unknown processor %q", index, name)
		}
	}
	if len(route.Exporters) == 0 {
		return fmt.Errorf("route %d exporters are required", index)
	}
	for _, name := range route.Exporters {
		if _, ok := c.Exporters[name]; !ok {
			return fmt.Errorf("route %d references unknown exporter %q", index, name)
		}
	}
	return nil
}

func FilterSensitiveAttrs(attrs Attrs, allow []string) Attrs {
	if len(attrs) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(allow))
	for _, key := range allow {
		allowed[strings.ToLower(key)] = struct{}{}
	}
	filtered := make(Attrs, len(attrs))
	for key, value := range attrs {
		normalized := strings.ToLower(key)
		if _, sensitive := defaultSensitiveKeys[normalized]; sensitive {
			if _, ok := allowed[normalized]; !ok {
				continue
			}
		}
		filtered[key] = value
	}
	return filtered
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
