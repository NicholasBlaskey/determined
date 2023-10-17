package expconf

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
)

// LogPatternPoliciesConfigV0 is a list of log pattern actions.
//
//go:generate ../gen.sh
type LogPatternPoliciesConfigV0 []LogPatternPolicyV0

// Merge implemenets the mergable interface.
func (b LogPatternPoliciesConfigV0) Merge(
	other LogPatternPoliciesConfigV0,
) LogPatternPoliciesConfigV0 {
	var out LogPatternPoliciesConfigV0
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

// LogPatternPolicyV0 is an action to take if we match against trial logs.
//
//go:generate ../gen.sh
type LogPatternPolicyV0 struct {
	RawPattern string `json:"pattern"`

	RawPolicy *LogPolicyV0 `json:"policy"`
}

// LogPolicyType is a type for different log policies types.
// You will need to convert this to a "true" union if you need to
// allow configuring other options besides what type of policy it is.
type LogPolicyType string

// All the log policy types.
const (
	LogPolicyOnFailureDontRetry   LogPolicyType = "on_failure_dont_retry"
	LogPolicyOnFailureExcludeNode LogPolicyType = "on_failure_exclude_node"
)

// LogPolicyV0 is a policy to take after matching.
//
//go:generate ../gen.sh
type LogPolicyV0 struct {
	RawType LogPolicyType `json:"type"`
}
