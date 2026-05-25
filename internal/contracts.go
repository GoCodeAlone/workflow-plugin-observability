package internal

import (
	observabilityv1 "github.com/GoCodeAlone/workflow-plugin-observability/gen/observability/v1"
	pb "github.com/GoCodeAlone/workflow/plugin/external/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const observabilityProtoPackage = "workflow.plugin.observability.v1."

func (p *observabilityPlugin) ContractRegistry() *pb.ContractRegistry {
	return observabilityContractRegistry
}

var observabilityContractRegistry = &pb.ContractRegistry{
	FileDescriptorSet: &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(structpb.File_google_protobuf_struct_proto),
			protodesc.ToFileDescriptorProto(timestamppb.File_google_protobuf_timestamp_proto),
			protodesc.ToFileDescriptorProto(observabilityv1.File_observability_v1_observability_proto),
		},
	},
	Contracts: []*pb.ContractDescriptor{
		{
			Kind:          pb.ContractKind_CONTRACT_KIND_MODULE,
			ModuleType:    "observability.collector",
			ConfigMessage: observabilityProtoPackage + "CollectorConfig",
			Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
		},
		{
			Kind:          pb.ContractKind_CONTRACT_KIND_MODULE,
			ModuleType:    "observability.telemetry",
			ConfigMessage: observabilityProtoPackage + "TelemetryConfig",
			Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
		},
		{
			Kind:          pb.ContractKind_CONTRACT_KIND_SERVICE,
			ModuleType:    "observability.collector",
			ServiceName:   "observability.collector",
			Method:        "plan",
			InputMessage:  observabilityProtoPackage + "Empty",
			OutputMessage: observabilityProtoPackage + "CollectorPlanOutput",
			Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
		},
		{
			Kind:          pb.ContractKind_CONTRACT_KIND_SERVICE,
			ModuleType:    "observability.collector",
			ServiceName:   "observability.collector",
			Method:        "renderConfig",
			InputMessage:  observabilityProtoPackage + "Empty",
			OutputMessage: observabilityProtoPackage + "RenderConfigOutput",
			Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
		},
		{
			Kind:          pb.ContractKind_CONTRACT_KIND_SERVICE,
			ModuleType:    "observability.telemetry",
			ServiceName:   "observability.telemetry",
			Method:        "recordMetrics",
			InputMessage:  observabilityProtoPackage + "RecordMetricsInput",
			OutputMessage: observabilityProtoPackage + "AcceptedOutput",
			Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
		},
		{
			Kind:          pb.ContractKind_CONTRACT_KIND_SERVICE,
			ModuleType:    "observability.telemetry",
			ServiceName:   "observability.telemetry",
			Method:        "recordLogs",
			InputMessage:  observabilityProtoPackage + "RecordLogsInput",
			OutputMessage: observabilityProtoPackage + "AcceptedOutput",
			Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
		},
		{
			Kind:          pb.ContractKind_CONTRACT_KIND_SERVICE,
			ModuleType:    "observability.telemetry",
			ServiceName:   "observability.telemetry",
			Method:        "recordSpanEvents",
			InputMessage:  observabilityProtoPackage + "RecordSpanEventsInput",
			OutputMessage: observabilityProtoPackage + "AcceptedOutput",
			Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
		},
		{
			Kind:          pb.ContractKind_CONTRACT_KIND_SERVICE,
			ModuleType:    "observability.telemetry",
			ServiceName:   "observability.telemetry",
			Method:        "snapshot",
			InputMessage:  observabilityProtoPackage + "Empty",
			OutputMessage: observabilityProtoPackage + "SnapshotOutput",
			Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
		},
	},
}

var _ interface{ ContractRegistry() *pb.ContractRegistry } = (*observabilityPlugin)(nil)
