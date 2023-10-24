package expconf

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
)

// LogPoliciesConfigV0 is a list of log policies.
//
//go:generate ../gen.sh
type LogPoliciesConfigV0 []LogPolicyV0

// Merge implemenets the mergable interface.
func (b LogPoliciesConfigV0) Merge(
	other LogPoliciesConfigV0,
) LogPoliciesConfigV0 {
	var out LogPoliciesConfigV0
	seen := make(map[string]bool)
	for _, p := range append(other, b...) {
		json, err := json.Marshal(p)
		if err != nil {
			log.Errorf("marshaling error %+v %v", p, err)
		}
		if seen[string(json)] {
			continue
		}
		seen[string(json)] = true

		out = append(out, p)
	}
	return out
}

// LogPolicyV0 is an action to take if we match against trial logs.
//
//go:generate ../gen.sh
type LogPolicyV0 struct {
	RawPattern string `json:"pattern"`

	RawAction LogActionV0 `json:"action"`
}

// LogActionType is a type for different log action types.
// You will need to convert this to a "true" union if you need to
// allow configuring other options besides what type of policy it is.
type LogActionType string

// All the log policy types.
const (
	LogActionCancelRetries LogActionType = "cancel_retries"
	LogActionExcludeNode   LogActionType = "exclude_node"
)

// LogActionV0 is an action to take after matching.
//
//go:generate ../gen.sh
type LogActionV0 struct {
	RawType LogActionType `json:"type"`
}
