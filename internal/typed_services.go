package internal

import (
	"fmt"
	"time"

	observabilityv1 "github.com/GoCodeAlone/workflow-plugin-observability/gen/observability/v1"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (m *collectorModule) InvokeTypedMethod(method string, input *anypb.Any) (*anypb.Any, error) {
	if _, err := unpackTypedConfig(input, &observabilityv1.Empty{}); err != nil {
		return nil, err
	}
	switch method {
	case "plan":
		rendered, err := RenderOTelCollectorYAML(m.config)
		if err != nil {
			return nil, err
		}
		return anypb.New(&observabilityv1.CollectorPlanOutput{
			Collector: &observabilityv1.CollectorPlan{
				Name:         m.name,
				Distribution: m.config.Distribution,
				Topology:     m.config.Topology,
				Config:       rendered,
			},
			Env:       m.appEnv(),
			Resources: generatedResourcesToProto(m.generatedResources()),
		})
	case "renderConfig":
		rendered, err := RenderOTelCollectorYAML(m.config)
		if err != nil {
			return nil, err
		}
		return anypb.New(&observabilityv1.RenderConfigOutput{Config: rendered})
	default:
		return nil, fmt.Errorf("observability.collector %q: unknown typed method %q", m.name, method)
	}
}

func (m *telemetryModule) InvokeTypedMethod(method string, input *anypb.Any) (*anypb.Any, error) {
	switch method {
	case "recordMetrics":
		req, err := unpackTypedConfig(input, &observabilityv1.RecordMetricsInput{})
		if err != nil {
			return nil, err
		}
		records := metricRecordsFromProto(req.GetMetrics())
		m.mu.Lock()
		m.snapshot.Metrics = append(m.snapshot.Metrics, records...)
		m.mu.Unlock()
		return anypb.New(&observabilityv1.AcceptedOutput{Accepted: true, Count: int32(len(records))})
	case "recordLogs":
		req, err := unpackTypedConfig(input, &observabilityv1.RecordLogsInput{})
		if err != nil {
			return nil, err
		}
		records := logRecordsFromProto(req.GetLogs())
		m.mu.Lock()
		m.snapshot.Logs = append(m.snapshot.Logs, records...)
		m.mu.Unlock()
		return anypb.New(&observabilityv1.AcceptedOutput{Accepted: true, Count: int32(len(records))})
	case "recordSpanEvents":
		req, err := unpackTypedConfig(input, &observabilityv1.RecordSpanEventsInput{})
		if err != nil {
			return nil, err
		}
		records := spanEventsFromProto(req.GetEvents())
		m.mu.Lock()
		m.snapshot.SpanEvents = append(m.snapshot.SpanEvents, records...)
		m.mu.Unlock()
		return anypb.New(&observabilityv1.AcceptedOutput{Accepted: true, Count: int32(len(records))})
	case "snapshot":
		if _, err := unpackTypedConfig(input, &observabilityv1.Empty{}); err != nil {
			return nil, err
		}
		return anypb.New(snapshotToProto(m.Snapshot()))
	default:
		return nil, fmt.Errorf("observability.telemetry %q: unknown typed method %q", m.name, method)
	}
}

func generatedResourcesToProto(values []map[string]any) []*observabilityv1.GeneratedResourceRef {
	out := make([]*observabilityv1.GeneratedResourceRef, 0, len(values))
	for _, value := range values {
		out = append(out, &observabilityv1.GeneratedResourceRef{
			Kind:   stringValue(value["kind"]),
			Name:   stringValue(value["name"]),
			Labels: stringMapValue(value["labels"]),
		})
	}
	return out
}

func metricRecordsFromProto(values []*observabilityv1.MetricRecord) []MetricRecord {
	out := make([]MetricRecord, 0, len(values))
	for _, value := range values {
		out = append(out, MetricRecord{
			Name:      value.GetName(),
			Kind:      value.GetKind(),
			Value:     value.GetValue(),
			Attrs:     FilterSensitiveAttrs(copyAttrs(value.GetAttrs()), nil),
			Timestamp: timeFromProto(value.GetTimestamp()),
		})
	}
	return out
}

func logRecordsFromProto(values []*observabilityv1.LogRecord) []LogRecord {
	out := make([]LogRecord, 0, len(values))
	for _, value := range values {
		out = append(out, LogRecord{
			Timestamp: timeFromProto(value.GetTimestamp()),
			Level:     value.GetLevel(),
			Message:   value.GetMessage(),
			Module:    value.GetModule(),
			Attrs:     FilterSensitiveAttrs(copyAttrs(value.GetAttrs()), nil),
		})
	}
	return out
}

func spanEventsFromProto(values []*observabilityv1.SpanEvent) []SpanEvent {
	out := make([]SpanEvent, 0, len(values))
	for _, value := range values {
		out = append(out, SpanEvent{
			Name:      value.GetName(),
			Attrs:     FilterSensitiveAttrs(copyAttrs(value.GetAttrs()), nil),
			Timestamp: timeFromProto(value.GetTimestamp()),
		})
	}
	return out
}

func snapshotToProto(snapshot TelemetrySnapshot) *observabilityv1.SnapshotOutput {
	return &observabilityv1.SnapshotOutput{
		Metrics:    metricRecordsToProto(snapshot.Metrics),
		Logs:       logRecordsToProto(snapshot.Logs),
		SpanEvents: spanEventsToProto(snapshot.SpanEvents),
	}
}

func metricRecordsToProto(values []MetricRecord) []*observabilityv1.MetricRecord {
	out := make([]*observabilityv1.MetricRecord, 0, len(values))
	for _, value := range values {
		out = append(out, &observabilityv1.MetricRecord{
			Name:      value.Name,
			Kind:      value.Kind,
			Value:     value.Value,
			Attrs:     copyStringMap(value.Attrs),
			Timestamp: timeToProto(value.Timestamp),
		})
	}
	return out
}

func logRecordsToProto(values []LogRecord) []*observabilityv1.LogRecord {
	out := make([]*observabilityv1.LogRecord, 0, len(values))
	for _, value := range values {
		out = append(out, &observabilityv1.LogRecord{
			Timestamp: timeToProto(value.Timestamp),
			Level:     value.Level,
			Message:   value.Message,
			Module:    value.Module,
			Attrs:     copyStringMap(value.Attrs),
		})
	}
	return out
}

func spanEventsToProto(values []SpanEvent) []*observabilityv1.SpanEvent {
	out := make([]*observabilityv1.SpanEvent, 0, len(values))
	for _, value := range values {
		out = append(out, &observabilityv1.SpanEvent{
			Name:      value.Name,
			Attrs:     copyStringMap(value.Attrs),
			Timestamp: timeToProto(value.Timestamp),
		})
	}
	return out
}

func copyAttrs(values map[string]string) Attrs {
	if len(values) == 0 {
		return nil
	}
	out := make(Attrs, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func timeToProto(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value)
}

func timeFromProto(value *timestamppb.Timestamp) time.Time {
	if value == nil {
		return time.Time{}
	}
	if err := value.CheckValid(); err != nil {
		return time.Time{}
	}
	return value.AsTime()
}

var (
	_ interface {
		InvokeTypedMethod(string, *anypb.Any) (*anypb.Any, error)
	} = (*collectorModule)(nil)
	_ interface {
		InvokeTypedMethod(string, *anypb.Any) (*anypb.Any, error)
	} = (*telemetryModule)(nil)
)
