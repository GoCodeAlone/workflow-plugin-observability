package internal

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type telemetryModule struct {
	name   string
	config TelemetryConfig

	mu       sync.RWMutex
	snapshot TelemetrySnapshot
}

type TelemetrySnapshot struct {
	Metrics    []MetricRecord `json:"metrics"`
	Logs       []LogRecord    `json:"logs"`
	SpanEvents []SpanEvent    `json:"spanEvents"`
}

type MetricRecord struct {
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	Value     float64   `json:"value"`
	Attrs     Attrs     `json:"attrs,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

type LogRecord struct {
	Timestamp time.Time `json:"timestamp,omitempty"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Module    string    `json:"module,omitempty"`
	Attrs     Attrs     `json:"attrs,omitempty"`
}

type SpanEvent struct {
	Name      string    `json:"name"`
	Attrs     Attrs     `json:"attrs,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

func newTelemetryModule(name string, cfg map[string]any) (*telemetryModule, error) {
	mod := &telemetryModule{name: name}
	if cfg != nil {
		mod.config = TelemetryConfig{
			ServiceName:        stringValue(cfg["serviceName"]),
			Environment:        stringValue(cfg["environment"]),
			ResourceAttributes: attrsValue(cfg["resource"]),
			OTLPEndpoint:       stringValue(cfg["otlpEndpoint"]),
		}
	}
	return mod, nil
}

func (m *telemetryModule) Init() error {
	return nil
}

func (m *telemetryModule) Start(context.Context) error {
	return nil
}

func (m *telemetryModule) Stop(context.Context) error {
	return nil
}

func (m *telemetryModule) InvokeMethod(method string, args map[string]any) (map[string]any, error) {
	return m.InvokeMethodContext(context.Background(), method, args)
}

func (m *telemetryModule) InvokeMethodContext(_ context.Context, method string, args map[string]any) (map[string]any, error) {
	switch method {
	case "recordMetrics":
		records, err := metricRecordsFromArgs(args["metrics"])
		if err != nil {
			return nil, err
		}
		m.mu.Lock()
		m.snapshot.Metrics = append(m.snapshot.Metrics, records...)
		m.mu.Unlock()
		return map[string]any{"accepted": true, "count": len(records)}, nil
	case "recordLogs":
		records, err := logRecordsFromArgs(args["logs"])
		if err != nil {
			return nil, err
		}
		m.mu.Lock()
		m.snapshot.Logs = append(m.snapshot.Logs, records...)
		m.mu.Unlock()
		return map[string]any{"accepted": true, "count": len(records)}, nil
	case "recordSpanEvents":
		records, err := spanEventsFromArgs(args["events"])
		if err != nil {
			return nil, err
		}
		m.mu.Lock()
		m.snapshot.SpanEvents = append(m.snapshot.SpanEvents, records...)
		m.mu.Unlock()
		return map[string]any{"accepted": true, "count": len(records)}, nil
	case "snapshot":
		return map[string]any{"snapshot": m.Snapshot()}, nil
	default:
		return nil, fmt.Errorf("observability.telemetry %q: unknown method %q", m.name, method)
	}
}

func (m *telemetryModule) Snapshot() TelemetrySnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := TelemetrySnapshot{
		Metrics:    make([]MetricRecord, len(m.snapshot.Metrics)),
		Logs:       make([]LogRecord, len(m.snapshot.Logs)),
		SpanEvents: make([]SpanEvent, len(m.snapshot.SpanEvents)),
	}
	copy(out.Metrics, m.snapshot.Metrics)
	copy(out.Logs, m.snapshot.Logs)
	copy(out.SpanEvents, m.snapshot.SpanEvents)
	for i := range out.Metrics {
		out.Metrics[i].Attrs = FilterSensitiveAttrs(out.Metrics[i].Attrs, nil)
	}
	for i := range out.Logs {
		out.Logs[i].Attrs = FilterSensitiveAttrs(out.Logs[i].Attrs, nil)
	}
	for i := range out.SpanEvents {
		out.SpanEvents[i].Attrs = FilterSensitiveAttrs(out.SpanEvents[i].Attrs, nil)
	}
	return out
}

func metricRecordsFromArgs(value any) ([]MetricRecord, error) {
	items, err := sliceOfMaps(value)
	if err != nil {
		return nil, err
	}
	records := make([]MetricRecord, 0, len(items))
	for _, item := range items {
		records = append(records, MetricRecord{
			Name:      stringValue(item["name"]),
			Kind:      stringValue(item["kind"]),
			Value:     floatValue(item["value"]),
			Attrs:     FilterSensitiveAttrs(attrsValue(item["attrs"]), nil),
			Timestamp: timeValue(item["timestamp"]),
		})
	}
	return records, nil
}

func logRecordsFromArgs(value any) ([]LogRecord, error) {
	items, err := sliceOfMaps(value)
	if err != nil {
		return nil, err
	}
	records := make([]LogRecord, 0, len(items))
	for _, item := range items {
		records = append(records, LogRecord{
			Timestamp: timeValue(item["timestamp"]),
			Level:     stringValue(item["level"]),
			Message:   stringValue(item["message"]),
			Module:    stringValue(item["module"]),
			Attrs:     FilterSensitiveAttrs(attrsValue(item["attrs"]), nil),
		})
	}
	return records, nil
}

func spanEventsFromArgs(value any) ([]SpanEvent, error) {
	items, err := sliceOfMaps(value)
	if err != nil {
		return nil, err
	}
	records := make([]SpanEvent, 0, len(items))
	for _, item := range items {
		records = append(records, SpanEvent{
			Name:      stringValue(item["name"]),
			Attrs:     FilterSensitiveAttrs(attrsValue(item["attrs"]), nil),
			Timestamp: timeValue(item["timestamp"]),
		})
	}
	return records, nil
}

func sliceOfMaps(value any) ([]map[string]any, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case []map[string]any:
		return typed, nil
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			mapped, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("expected map item, got %T", item)
			}
			out = append(out, mapped)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected records slice, got %T", value)
	}
}

func attrsValue(value any) Attrs {
	switch typed := value.(type) {
	case nil:
		return nil
	case Attrs:
		return typed
	case map[string]string:
		out := make(Attrs, len(typed))
		for k, v := range typed {
			out[k] = v
		}
		return out
	case map[string]any:
		out := make(Attrs, len(typed))
		for k, v := range typed {
			out[k] = stringValue(v)
		}
		return out
	default:
		return nil
	}
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(value)
	}
}

func floatValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		parsed, _ := strconv.ParseFloat(typed, 64)
		return parsed
	default:
		return 0
	}
}

func timeValue(value any) time.Time {
	switch typed := value.(type) {
	case time.Time:
		return typed
	case string:
		if typed == "" {
			return time.Time{}
		}
		parsed, _ := time.Parse(time.RFC3339Nano, typed)
		return parsed
	default:
		return time.Time{}
	}
}
