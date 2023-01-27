package command

import (
	"fmt"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// LaunchWarning represents warnings related to launching commands.
type LaunchWarning int

const (
	// CurrentSlotsExceeded represents a resource pool having insufficient slots.
	CurrentSlotsExceeded LaunchWarning = 1

	// AgentLabelWithoutMatchingAgent represents a task submitted with
	// resources.agent_label with no matching agent.
	AgentLabelWithoutMatchingAgent LaunchWarning = 2
)

func toProtoEnum(l LaunchWarning) apiv1.LaunchWarning {
	switch l {
	case CurrentSlotsExceeded:
		return apiv1.LaunchWarning_LAUNCH_WARNING_CURRENT_SLOTS_EXCEEDED
	case AgentLabelWithoutMatchingAgent:
		return apiv1.LaunchWarning_LAUNCH_WARNING_AGENT_LABEL_WITHOUT_MATCHING_AGENT
	default:
		panic(fmt.Sprintf("Unknown LaunchWarning value %v", l))
	}
}

// LaunchWarningToProto converts LaunchWarnings to their protobuf representation.
func LaunchWarningToProto(lw []LaunchWarning) []apiv1.LaunchWarning {
	res := make([]apiv1.LaunchWarning, 0, len(lw))
	for _, w := range lw {
		res = append(res, toProtoEnum(w))
	}
	return res
}
