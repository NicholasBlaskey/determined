// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// source: determined/agent/v1/agent.proto

package agentv1

import (
	containerv1 "github.com/determined-ai/determined/proto/pkg/containerv1"
	devicev1 "github.com/determined-ai/determined/proto/pkg/devicev1"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// An scheduler action.
// DISCUSS: we can furhter expand to other categories.
type Action int32

const (
	// An unspecified action. No op.
	Action_ACTION_UNSPECIFIED Action = 0
	// Run on another device.
	Action_ACTION_RETRY_DIFFERENT_DEVICE Action = 1
	// Run on different device type.
	// Scheduler will not be able to accomodate in the same RP for now.
	Action_ACTION_RETRY_DIFFERENT_DEVICE_TYPE Action = 2
)

// Enum value maps for Action.
var (
	Action_name = map[int32]string{
		0: "ACTION_UNSPECIFIED",
		1: "ACTION_RETRY_DIFFERENT_DEVICE",
		2: "ACTION_RETRY_DIFFERENT_DEVICE_TYPE",
	}
	Action_value = map[string]int32{
		"ACTION_UNSPECIFIED":                 0,
		"ACTION_RETRY_DIFFERENT_DEVICE":      1,
		"ACTION_RETRY_DIFFERENT_DEVICE_TYPE": 2,
	}
)

func (x Action) Enum() *Action {
	p := new(Action)
	*p = x
	return p
}

func (x Action) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Action) Descriptor() protoreflect.EnumDescriptor {
	return file_determined_agent_v1_agent_proto_enumTypes[0].Descriptor()
}

func (Action) Type() protoreflect.EnumType {
	return &file_determined_agent_v1_agent_proto_enumTypes[0]
}

func (x Action) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Action.Descriptor instead.
func (Action) EnumDescriptor() ([]byte, []int) {
	return file_determined_agent_v1_agent_proto_rawDescGZIP(), []int{0}
}

// Agent is a pool of resources where containers are run.
type Agent struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The unique id of the agent.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// The time when the agent registered with the master.
	RegisteredTime *timestamp.Timestamp `protobuf:"bytes,2,opt,name=registered_time,json=registeredTime,proto3" json:"registered_time,omitempty"`
	// A map of slot id to each slot of this agent.
	Slots map[string]*Slot `protobuf:"bytes,3,rep,name=slots,proto3" json:"slots,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// A map of container id to all containers assigned to this agent.
	Containers map[string]*containerv1.Container `protobuf:"bytes,4,rep,name=containers,proto3" json:"containers,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// This field has been deprecated and will be empty.
	Label string `protobuf:"bytes,5,opt,name=label,proto3" json:"label,omitempty"`
	// The addresses of the agent.
	Addresses []string `protobuf:"bytes,7,rep,name=addresses,proto3" json:"addresses,omitempty"`
	// Flag notifying if containers can be scheduled on this agent.
	Enabled bool `protobuf:"varint,8,opt,name=enabled,proto3" json:"enabled,omitempty"`
	// Flag notifying if this agent is in the draining mode: current containers
	// will be allowed to finish but no new ones will be scheduled.
	Draining bool `protobuf:"varint,9,opt,name=draining,proto3" json:"draining,omitempty"`
	// The Determined version that this agent was built from.
	Version string `protobuf:"bytes,10,opt,name=version,proto3" json:"version,omitempty"`
	// The name of the resource pools the agent is in. Only slurm can contain
	// multiples.
	ResourcePools []string `protobuf:"bytes,6,rep,name=resource_pools,json=resourcePools,proto3" json:"resource_pools,omitempty"`
}

func (x *Agent) Reset() {
	*x = Agent{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_agent_v1_agent_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Agent) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Agent) ProtoMessage() {}

func (x *Agent) ProtoReflect() protoreflect.Message {
	mi := &file_determined_agent_v1_agent_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Agent.ProtoReflect.Descriptor instead.
func (*Agent) Descriptor() ([]byte, []int) {
	return file_determined_agent_v1_agent_proto_rawDescGZIP(), []int{0}
}

func (x *Agent) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Agent) GetRegisteredTime() *timestamp.Timestamp {
	if x != nil {
		return x.RegisteredTime
	}
	return nil
}

func (x *Agent) GetSlots() map[string]*Slot {
	if x != nil {
		return x.Slots
	}
	return nil
}

func (x *Agent) GetContainers() map[string]*containerv1.Container {
	if x != nil {
		return x.Containers
	}
	return nil
}

func (x *Agent) GetLabel() string {
	if x != nil {
		return x.Label
	}
	return ""
}

func (x *Agent) GetAddresses() []string {
	if x != nil {
		return x.Addresses
	}
	return nil
}

func (x *Agent) GetEnabled() bool {
	if x != nil {
		return x.Enabled
	}
	return false
}

func (x *Agent) GetDraining() bool {
	if x != nil {
		return x.Draining
	}
	return false
}

func (x *Agent) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *Agent) GetResourcePools() []string {
	if x != nil {
		return x.ResourcePools
	}
	return nil
}

// Slot wraps a single device on the agent.
type Slot struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The unqiue id of the slot for a given agent.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// The individual resource this slot wraps.
	Device *devicev1.Device `protobuf:"bytes,2,opt,name=device,proto3" json:"device,omitempty"`
	// Flag notifying if containers can be scheduled on this slot.
	Enabled bool `protobuf:"varint,3,opt,name=enabled,proto3" json:"enabled,omitempty"`
	// Container that is currently running on this agent. It is unset if there is
	// no container currently running on this slot.
	Container *containerv1.Container `protobuf:"bytes,4,opt,name=container,proto3" json:"container,omitempty"`
	// Flag notifying if this slot is in the draining mode: current containers
	// will be allowed to finish but no new ones will be scheduled.
	Draining bool `protobuf:"varint,5,opt,name=draining,proto3" json:"draining,omitempty"`
}

func (x *Slot) Reset() {
	*x = Slot{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_agent_v1_agent_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Slot) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Slot) ProtoMessage() {}

func (x *Slot) ProtoReflect() protoreflect.Message {
	mi := &file_determined_agent_v1_agent_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Slot.ProtoReflect.Descriptor instead.
func (*Slot) Descriptor() ([]byte, []int) {
	return file_determined_agent_v1_agent_proto_rawDescGZIP(), []int{1}
}

func (x *Slot) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Slot) GetDevice() *devicev1.Device {
	if x != nil {
		return x.Device
	}
	return nil
}

func (x *Slot) GetEnabled() bool {
	if x != nil {
		return x.Enabled
	}
	return false
}

func (x *Slot) GetContainer() *containerv1.Container {
	if x != nil {
		return x.Container
	}
	return nil
}

func (x *Slot) GetDraining() bool {
	if x != nil {
		return x.Draining
	}
	return false
}

// RunAlert describes a task exception.
// TODO: move to task?
type RunAlert struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The job that the alert was detected on. REMOVEME?
	JobId string `protobuf:"bytes,1,opt,name=job_id,json=jobId,proto3" json:"job_id,omitempty"`
	// Agent or node identifier.
	NodeId string `protobuf:"bytes,2,opt,name=node_id,json=nodeId,proto3" json:"node_id,omitempty"`
	// The task that the alert was detected on.
	TaskId string `protobuf:"bytes,3,opt,name=task_id,json=taskId,proto3" json:"task_id,omitempty"`
	// The suspected devices that failed, if known.
	Devices []*devicev1.Device `protobuf:"bytes,4,rep,name=devices,proto3" json:"devices,omitempty"`
	// The requested scheduling action.
	// DISCUSS: could be a list of actions based on available actions.
	Action Action `protobuf:"varint,5,opt,name=action,proto3,enum=determined.agent.v1.Action" json:"action,omitempty"`
}

func (x *RunAlert) Reset() {
	*x = RunAlert{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_agent_v1_agent_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RunAlert) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RunAlert) ProtoMessage() {}

func (x *RunAlert) ProtoReflect() protoreflect.Message {
	mi := &file_determined_agent_v1_agent_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RunAlert.ProtoReflect.Descriptor instead.
func (*RunAlert) Descriptor() ([]byte, []int) {
	return file_determined_agent_v1_agent_proto_rawDescGZIP(), []int{2}
}

func (x *RunAlert) GetJobId() string {
	if x != nil {
		return x.JobId
	}
	return ""
}

func (x *RunAlert) GetNodeId() string {
	if x != nil {
		return x.NodeId
	}
	return ""
}

func (x *RunAlert) GetTaskId() string {
	if x != nil {
		return x.TaskId
	}
	return ""
}

func (x *RunAlert) GetDevices() []*devicev1.Device {
	if x != nil {
		return x.Devices
	}
	return nil
}

func (x *RunAlert) GetAction() Action {
	if x != nil {
		return x.Action
	}
	return Action_ACTION_UNSPECIFIED
}

var File_determined_agent_v1_agent_proto protoreflect.FileDescriptor

var file_determined_agent_v1_agent_proto_rawDesc = []byte{
	0x0a, 0x1f, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2f, 0x61, 0x67, 0x65,
	0x6e, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x13, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x61, 0x67,
	0x65, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x2c, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x2d,
	0x67, 0x65, 0x6e, 0x2d, 0x73, 0x77, 0x61, 0x67, 0x67, 0x65, 0x72, 0x2f, 0x6f, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x27, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65,
	0x64, 0x2f, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x63,
	0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x21,
	0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2f, 0x64, 0x65, 0x76, 0x69, 0x63,
	0x65, 0x2f, 0x76, 0x31, 0x2f, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0xd4, 0x04, 0x0a, 0x05, 0x41, 0x67, 0x65, 0x6e, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x43, 0x0a, 0x0f, 0x72,
	0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x65, 0x64, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x52, 0x0e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x65, 0x64, 0x54, 0x69, 0x6d, 0x65,
	0x12, 0x3b, 0x0a, 0x05, 0x73, 0x6c, 0x6f, 0x74, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x25, 0x2e, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x61, 0x67, 0x65,
	0x6e, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x53, 0x6c, 0x6f, 0x74,
	0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x05, 0x73, 0x6c, 0x6f, 0x74, 0x73, 0x12, 0x4a, 0x0a,
	0x0a, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x2a, 0x2e, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x61,
	0x67, 0x65, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x43, 0x6f,
	0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0a, 0x63,
	0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x6c, 0x61, 0x62,
	0x65, 0x6c, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x12,
	0x1c, 0x0a, 0x09, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x65, 0x73, 0x18, 0x07, 0x20, 0x03,
	0x28, 0x09, 0x52, 0x09, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x65, 0x73, 0x12, 0x18, 0x0a,
	0x07, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64, 0x18, 0x08, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07,
	0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x64, 0x72, 0x61, 0x69, 0x6e,
	0x69, 0x6e, 0x67, 0x18, 0x09, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x64, 0x72, 0x61, 0x69, 0x6e,
	0x69, 0x6e, 0x67, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x0a,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x25, 0x0a,
	0x0e, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x5f, 0x70, 0x6f, 0x6f, 0x6c, 0x73, 0x18,
	0x06, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0d, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x50,
	0x6f, 0x6f, 0x6c, 0x73, 0x1a, 0x53, 0x0a, 0x0a, 0x53, 0x6c, 0x6f, 0x74, 0x73, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x03, 0x6b, 0x65, 0x79, 0x12, 0x2f, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64,
	0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x6c, 0x6f, 0x74, 0x52, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x61, 0x0a, 0x0f, 0x43, 0x6f, 0x6e,
	0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03,
	0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x38,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x22, 0x2e,
	0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x63, 0x6f, 0x6e, 0x74, 0x61,
	0x69, 0x6e, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65,
	0x72, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x3a, 0x0a, 0x92, 0x41,
	0x07, 0x0a, 0x05, 0xd2, 0x01, 0x02, 0x69, 0x64, 0x22, 0xc4, 0x01, 0x0a, 0x04, 0x53, 0x6c, 0x6f,
	0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69,
	0x64, 0x12, 0x34, 0x0a, 0x06, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x1c, 0x2e, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x64,
	0x65, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x52,
	0x06, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x65, 0x6e, 0x61, 0x62, 0x6c,
	0x65, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65,
	0x64, 0x12, 0x40, 0x0a, 0x09, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65,
	0x64, 0x2e, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43,
	0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x52, 0x09, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69,
	0x6e, 0x65, 0x72, 0x12, 0x1a, 0x0a, 0x08, 0x64, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x64, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x22,
	0xee, 0x01, 0x0a, 0x08, 0x52, 0x75, 0x6e, 0x41, 0x6c, 0x65, 0x72, 0x74, 0x12, 0x15, 0x0a, 0x06,
	0x6a, 0x6f, 0x62, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6a, 0x6f,
	0x62, 0x49, 0x64, 0x12, 0x17, 0x0a, 0x07, 0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x6e, 0x6f, 0x64, 0x65, 0x49, 0x64, 0x12, 0x17, 0x0a, 0x07,
	0x74, 0x61, 0x73, 0x6b, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x74,
	0x61, 0x73, 0x6b, 0x49, 0x64, 0x12, 0x36, 0x0a, 0x07, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x73,
	0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69,
	0x6e, 0x65, 0x64, 0x2e, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x65,
	0x76, 0x69, 0x63, 0x65, 0x52, 0x07, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x73, 0x12, 0x33, 0x0a,
	0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1b, 0x2e,
	0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74,
	0x2e, 0x76, 0x31, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x06, 0x61, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x3a, 0x2c, 0x92, 0x41, 0x29, 0x0a, 0x27, 0xd2, 0x01, 0x07, 0x6e, 0x6f, 0x64, 0x65,
	0x5f, 0x69, 0x64, 0xd2, 0x01, 0x07, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x73, 0xd2, 0x01, 0x07,
	0x74, 0x61, 0x73, 0x6b, 0x5f, 0x69, 0x64, 0xd2, 0x01, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x2a, 0x6b, 0x0a, 0x06, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x16, 0x0a, 0x12, 0x41, 0x43,
	0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44,
	0x10, 0x00, 0x12, 0x21, 0x0a, 0x1d, 0x41, 0x43, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x52, 0x45, 0x54,
	0x52, 0x59, 0x5f, 0x44, 0x49, 0x46, 0x46, 0x45, 0x52, 0x45, 0x4e, 0x54, 0x5f, 0x44, 0x45, 0x56,
	0x49, 0x43, 0x45, 0x10, 0x01, 0x12, 0x26, 0x0a, 0x22, 0x41, 0x43, 0x54, 0x49, 0x4f, 0x4e, 0x5f,
	0x52, 0x45, 0x54, 0x52, 0x59, 0x5f, 0x44, 0x49, 0x46, 0x46, 0x45, 0x52, 0x45, 0x4e, 0x54, 0x5f,
	0x44, 0x45, 0x56, 0x49, 0x43, 0x45, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x10, 0x02, 0x42, 0x37, 0x5a,
	0x35, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x65, 0x74, 0x65,
	0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2d, 0x61, 0x69, 0x2f, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d,
	0x69, 0x6e, 0x65, 0x64, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x61,
	0x67, 0x65, 0x6e, 0x74, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_determined_agent_v1_agent_proto_rawDescOnce sync.Once
	file_determined_agent_v1_agent_proto_rawDescData = file_determined_agent_v1_agent_proto_rawDesc
)

func file_determined_agent_v1_agent_proto_rawDescGZIP() []byte {
	file_determined_agent_v1_agent_proto_rawDescOnce.Do(func() {
		file_determined_agent_v1_agent_proto_rawDescData = protoimpl.X.CompressGZIP(file_determined_agent_v1_agent_proto_rawDescData)
	})
	return file_determined_agent_v1_agent_proto_rawDescData
}

var file_determined_agent_v1_agent_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_determined_agent_v1_agent_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_determined_agent_v1_agent_proto_goTypes = []interface{}{
	(Action)(0),                   // 0: determined.agent.v1.Action
	(*Agent)(nil),                 // 1: determined.agent.v1.Agent
	(*Slot)(nil),                  // 2: determined.agent.v1.Slot
	(*RunAlert)(nil),              // 3: determined.agent.v1.RunAlert
	nil,                           // 4: determined.agent.v1.Agent.SlotsEntry
	nil,                           // 5: determined.agent.v1.Agent.ContainersEntry
	(*timestamp.Timestamp)(nil),   // 6: google.protobuf.Timestamp
	(*devicev1.Device)(nil),       // 7: determined.device.v1.Device
	(*containerv1.Container)(nil), // 8: determined.container.v1.Container
}
var file_determined_agent_v1_agent_proto_depIdxs = []int32{
	6, // 0: determined.agent.v1.Agent.registered_time:type_name -> google.protobuf.Timestamp
	4, // 1: determined.agent.v1.Agent.slots:type_name -> determined.agent.v1.Agent.SlotsEntry
	5, // 2: determined.agent.v1.Agent.containers:type_name -> determined.agent.v1.Agent.ContainersEntry
	7, // 3: determined.agent.v1.Slot.device:type_name -> determined.device.v1.Device
	8, // 4: determined.agent.v1.Slot.container:type_name -> determined.container.v1.Container
	7, // 5: determined.agent.v1.RunAlert.devices:type_name -> determined.device.v1.Device
	0, // 6: determined.agent.v1.RunAlert.action:type_name -> determined.agent.v1.Action
	2, // 7: determined.agent.v1.Agent.SlotsEntry.value:type_name -> determined.agent.v1.Slot
	8, // 8: determined.agent.v1.Agent.ContainersEntry.value:type_name -> determined.container.v1.Container
	9, // [9:9] is the sub-list for method output_type
	9, // [9:9] is the sub-list for method input_type
	9, // [9:9] is the sub-list for extension type_name
	9, // [9:9] is the sub-list for extension extendee
	0, // [0:9] is the sub-list for field type_name
}

func init() { file_determined_agent_v1_agent_proto_init() }
func file_determined_agent_v1_agent_proto_init() {
	if File_determined_agent_v1_agent_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_determined_agent_v1_agent_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Agent); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_determined_agent_v1_agent_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Slot); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_determined_agent_v1_agent_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RunAlert); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_determined_agent_v1_agent_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_determined_agent_v1_agent_proto_goTypes,
		DependencyIndexes: file_determined_agent_v1_agent_proto_depIdxs,
		EnumInfos:         file_determined_agent_v1_agent_proto_enumTypes,
		MessageInfos:      file_determined_agent_v1_agent_proto_msgTypes,
	}.Build()
	File_determined_agent_v1_agent_proto = out.File
	file_determined_agent_v1_agent_proto_rawDesc = nil
	file_determined_agent_v1_agent_proto_goTypes = nil
	file_determined_agent_v1_agent_proto_depIdxs = nil
}
