package internal

import (
	"errors"
	"testing"
	"time"

	observabilityv1 "github.com/GoCodeAlone/workflow-plugin-observability/gen/observability/v1"
	pb "github.com/GoCodeAlone/workflow/plugin/external/proto"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestContractRegistryAdvertisesStrictProtoModules(t *testing.T) {
	provider, ok := NewPlugin().(interface {
		ContractRegistry() *pb.ContractRegistry
	})
	if !ok {
		t.Fatal("NewPlugin() does not implement ContractRegistry")
	}

	registry := provider.ContractRegistry()
	if registry == nil {
		t.Fatal("ContractRegistry() returned nil")
	}
	if registry.FileDescriptorSet == nil || len(registry.FileDescriptorSet.File) == 0 {
		t.Fatal("ContractRegistry() must include protobuf descriptors")
	}

	contracts := map[string]*pb.ContractDescriptor{}
	for _, contract := range registry.Contracts {
		if contract.Kind == pb.ContractKind_CONTRACT_KIND_MODULE {
			contracts[contract.ModuleType] = contract
		}
	}

	expected := map[string]string{
		"observability.collector": observabilityProtoPackage + "CollectorConfig",
		"observability.telemetry": observabilityProtoPackage + "TelemetryConfig",
	}
	for moduleType, configMessage := range expected {
		contract := contracts[moduleType]
		if contract == nil {
			t.Fatalf("missing contract for module type %q", moduleType)
		}
		if contract.Mode != pb.ContractMode_CONTRACT_MODE_STRICT_PROTO {
			t.Fatalf("%s mode = %v, want STRICT_PROTO", moduleType, contract.Mode)
		}
		if contract.ConfigMessage != configMessage {
			t.Fatalf("%s config message = %q, want %q", moduleType, contract.ConfigMessage, configMessage)
		}
	}
}

func TestContractRegistryAdvertisesStrictProtoServiceMethods(t *testing.T) {
	registry := NewPlugin().(interface {
		ContractRegistry() *pb.ContractRegistry
	}).ContractRegistry()

	contracts := map[string]*pb.ContractDescriptor{}
	for _, contract := range registry.Contracts {
		if contract.Kind == pb.ContractKind_CONTRACT_KIND_SERVICE {
			contracts[contract.ModuleType+"/"+contract.Method] = contract
		}
	}

	expected := map[string][2]string{
		"observability.collector/plan":             {observabilityProtoPackage + "Empty", observabilityProtoPackage + "CollectorPlanOutput"},
		"observability.collector/renderConfig":     {observabilityProtoPackage + "Empty", observabilityProtoPackage + "RenderConfigOutput"},
		"observability.telemetry/recordMetrics":    {observabilityProtoPackage + "RecordMetricsInput", observabilityProtoPackage + "AcceptedOutput"},
		"observability.telemetry/recordLogs":       {observabilityProtoPackage + "RecordLogsInput", observabilityProtoPackage + "AcceptedOutput"},
		"observability.telemetry/recordSpanEvents": {observabilityProtoPackage + "RecordSpanEventsInput", observabilityProtoPackage + "AcceptedOutput"},
		"observability.telemetry/snapshot":         {observabilityProtoPackage + "Empty", observabilityProtoPackage + "SnapshotOutput"},
	}
	for key, messages := range expected {
		contract := contracts[key]
		if contract == nil {
			t.Fatalf("missing service contract %q", key)
		}
		if contract.Mode != pb.ContractMode_CONTRACT_MODE_STRICT_PROTO {
			t.Fatalf("%s mode = %v, want STRICT_PROTO", key, contract.Mode)
		}
		if contract.InputMessage != messages[0] || contract.OutputMessage != messages[1] {
			t.Fatalf("%s messages = (%q, %q), want (%q, %q)", key, contract.InputMessage, contract.OutputMessage, messages[0], messages[1])
		}
	}
}

func TestContractRegistryDescriptorsResolveAllMessages(t *testing.T) {
	registry := NewPlugin().(interface {
		ContractRegistry() *pb.ContractRegistry
	}).ContractRegistry()
	files, err := protodesc.NewFiles(registry.FileDescriptorSet)
	if err != nil {
		t.Fatalf("FileDescriptorSet cannot be loaded: %v", err)
	}

	for _, contract := range registry.Contracts {
		for _, message := range []string{contract.ConfigMessage, contract.InputMessage, contract.OutputMessage} {
			if message == "" {
				continue
			}
			if _, err := files.FindDescriptorByName(protoreflect.FullName(message)); err != nil {
				t.Fatalf("contract message %q does not resolve: %v", message, err)
			}
		}
	}
}

func TestCreateTypedModuleAcceptsStrictProtoConfigs(t *testing.T) {
	provider, ok := NewPlugin().(interface {
		CreateTypedModule(string, string, *anypb.Any) (sdk.ModuleInstance, error)
	})
	if !ok {
		t.Fatal("NewPlugin() does not implement CreateTypedModule")
	}

	telemetry, err := provider.CreateTypedModule(
		"observability.telemetry",
		"telemetry",
		emptyTypedConfig(observabilityProtoPackage+"TelemetryConfig"),
	)
	if err != nil {
		t.Fatalf("CreateTypedModule(observability.telemetry): %v", err)
	}
	if _, ok := telemetry.(*telemetryModule); !ok {
		t.Fatalf("observability.telemetry typed module = %T, want *telemetryModule", telemetry)
	}

	collector, err := provider.CreateTypedModule(
		"observability.collector",
		"collector",
		emptyTypedConfig(observabilityProtoPackage+"CollectorConfig"),
	)
	if err != nil {
		t.Fatalf("CreateTypedModule(observability.collector): %v", err)
	}
	if _, ok := collector.(*collectorModule); !ok {
		t.Fatalf("observability.collector typed module = %T, want *collectorModule", collector)
	}
}

func TestTypedTelemetryMethodsRoundTripStrictProto(t *testing.T) {
	module := &telemetryModule{name: "telemetry"}
	now := time.Date(2026, 5, 24, 12, 30, 0, 0, time.UTC)
	input := mustPack(t, &observabilityv1.RecordMetricsInput{
		Metrics: []*observabilityv1.MetricRecord{{
			Name:      "requests",
			Kind:      "counter",
			Value:     7,
			Attrs:     map[string]string{"route": "/healthz", "token": "secret"},
			Timestamp: timestamppb.New(now),
		}},
	})

	acceptedAny, err := module.InvokeTypedMethod("recordMetrics", input)
	if err != nil {
		t.Fatalf("InvokeTypedMethod(recordMetrics): %v", err)
	}
	accepted := unpackForTest[*observabilityv1.AcceptedOutput](t, acceptedAny, &observabilityv1.AcceptedOutput{})
	if !accepted.GetAccepted() || accepted.GetCount() != 1 {
		t.Fatalf("accepted = %#v, want accepted count 1", accepted)
	}

	snapshotAny, err := module.InvokeTypedMethod("snapshot", mustPack(t, &observabilityv1.Empty{}))
	if err != nil {
		t.Fatalf("InvokeTypedMethod(snapshot): %v", err)
	}
	snapshot := unpackForTest[*observabilityv1.SnapshotOutput](t, snapshotAny, &observabilityv1.SnapshotOutput{})
	if len(snapshot.GetMetrics()) != 1 {
		t.Fatalf("snapshot metrics = %d, want 1", len(snapshot.GetMetrics()))
	}
	if _, ok := snapshot.GetMetrics()[0].GetAttrs()["token"]; ok {
		t.Fatal("snapshot leaked sensitive token attribute")
	}
	if got := snapshot.GetMetrics()[0].GetTimestamp().AsTime(); !got.Equal(now) {
		t.Fatalf("snapshot timestamp = %s, want %s", got, now)
	}
}

func TestTypedCollectorMethodsRoundTripStrictProto(t *testing.T) {
	module, err := newCollectorModule("collector", nil)
	if err != nil {
		t.Fatalf("newCollectorModule: %v", err)
	}
	renderedAny, err := module.InvokeTypedMethod("renderConfig", mustPack(t, &observabilityv1.Empty{}))
	if err != nil {
		t.Fatalf("InvokeTypedMethod(renderConfig): %v", err)
	}
	rendered := unpackForTest[*observabilityv1.RenderConfigOutput](t, renderedAny, &observabilityv1.RenderConfigOutput{})
	if rendered.GetConfig() == "" {
		t.Fatal("renderConfig returned empty config")
	}
}

func TestCreateTypedModuleDeclinesUnknownTypes(t *testing.T) {
	provider, ok := NewPlugin().(interface {
		CreateTypedModule(string, string, *anypb.Any) (sdk.ModuleInstance, error)
	})
	if !ok {
		t.Fatal("NewPlugin() does not implement CreateTypedModule")
	}

	_, err := provider.CreateTypedModule("observability.unknown", "unknown", nil)
	if !errors.Is(err, sdk.ErrTypedContractNotHandled) {
		t.Fatalf("CreateTypedModule unknown error = %v, want ErrTypedContractNotHandled", err)
	}
}

func mustPack(t *testing.T, msg proto.Message) *anypb.Any {
	t.Helper()
	payload, err := anypb.New(msg)
	if err != nil {
		t.Fatalf("pack %T: %v", msg, err)
	}
	return payload
}

func unpackForTest[T proto.Message](t *testing.T, payload *anypb.Any, target T) T {
	t.Helper()
	if err := payload.UnmarshalTo(target); err != nil {
		t.Fatalf("unpack %s: %v", payload.GetTypeUrl(), err)
	}
	return target
}

func emptyTypedConfig(messageName string) *anypb.Any {
	return &anypb.Any{
		TypeUrl: "type.googleapis.com/" + messageName,
		Value:   []byte{},
	}
}
