package logpattern

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/task/tasklogger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/set"

	"github.com/uptrace/bun"
)

const regexCacheSize = 256

var p = &LogPatternPolicies{
	blockListCache: make(map[model.TaskID]*set.Set[string]),
}

// Default returns the default singleton log pattern type.
func Default() *LogPatternPolicies {
	return p
}

// LogPatternPolicies is the singleton type that holds blocklist and regex cache.
type LogPatternPolicies struct {
	blockListCache map[model.TaskID]*set.Set[string]
	mu             sync.RWMutex
	regexCache     *lru.Cache[string, *regexp.Regexp]
}

func (l *LogPatternPolicies) getCompiledRegex(regex string) (*regexp.Regexp, error) {
	var err error
	if l.regexCache == nil {
		l.regexCache, err = lru.New[string, *regexp.Regexp](regexCacheSize)
		if err != nil {
			return nil, fmt.Errorf("creating LRU cache for compiled regex: %w", err)
		}
	}

	compiledRegex, ok := l.regexCache.Get(regex)
	if !ok {
		compiledRegex, err = regexp.Compile(regex)
		if err != nil {
			return nil, fmt.Errorf("compiling regex '%s': %w", regex, err)
		}

		l.regexCache.Add(regex, compiledRegex)
	}

	return compiledRegex, nil
}

// Monitor checks for logs against any log_pattern_policies and takes action according to the policy.
func (l *LogPatternPolicies) Monitor(ctx context.Context,
	taskID model.TaskID, logs []*model.TaskLog, policies expconf.LogPatternPoliciesConfig,
) error {
	if len(policies) == 0 {
		return nil
	}

	// TODO when we add rm specific log grabbing we will need to also monitor them.
	for _, log := range logs {
		if log.AgentID == nil {
			return fmt.Errorf("agentID must be non nil to monitor logs")
		}

		for _, policy := range policies {
			// TODO we have this problem where a regex will always match itself.
			// Should we match the regex against itself and regex it in expconf?
			// This does this since the first line of logs is printing expconf
			// which has the regex pattern. Maybe we can censor or omit the pattern?
			// I'm not sure. Maybe this isn't an issue since
			// regexes matching themself can be avoided by users.
			regex := fmt.Sprintf("(.*)%s(.*)", policy.Pattern())
			compiledRegex, err := l.getCompiledRegex(regex)
			if err != nil {
				return err
			}

			if compiledRegex.MatchString(log.Log) {
				switch policy.Policy().Type() {
				case expconf.LogPolicyOnFailureDontRetry:
					if err := addDontRetry(
						ctx, model.TaskID(log.TaskID), *log.AgentID, policy.Pattern(), log.Log,
					); err != nil {
						return fmt.Errorf("adding don't retry: %w", err)
					}

				case expconf.LogPolicyOnFailureExcludeNode:
					if err := l.addRetryOnDifferentNode(
						ctx, model.TaskID(log.TaskID), *log.AgentID, policy.Pattern(), log.Log,
					); err != nil {
						return fmt.Errorf("adding retry on different node: %w", err)
					}

				default:
					return fmt.Errorf("unrecognized log pattern policy type")
				}
			}
		}
	}

	return nil
}

// Initialize the blocked node list.
// There are two reasons for this using a cache
//  1. Avoid the possibility this feature causes a major slowdown to Scheduler
//     that won't be obvious till it run at scale.
//  2. Avoid putting possible transient db errors in the path of the Scheduler.
//
// I think there is going to be a decent chance this cache approach will somehow leak tasks
// in the future but I think even if we never removed items from the cache
// we would still probably be okay.
func (l *LogPatternPolicies) Initialize(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var blockedNodes []*retryOnDifferentNode
	if err := db.Bun().NewSelect().Model(&blockedNodes).
		Where("task_ended = false").
		Scan(ctx, &blockedNodes); err != nil {
		return fmt.Errorf("getting blocked nodes: %w", err)
	}

	l.blockListCache = make(map[model.TaskID]*set.Set[string])
	for _, b := range blockedNodes {
		if _, ok := l.blockListCache[b.TaskID]; !ok {
			l.blockListCache[b.TaskID] = ptrs.Ptr(set.New[string]())
		}
		l.blockListCache[b.TaskID].Insert(b.NodeName)
	}

	return nil
}

// DisallowedNodes returns a list of nodes that should be blocklisted for the given allocation.
func (l *LogPatternPolicies) DisallowedNodes(taskID model.TaskID) *set.Set[string] {
	l.mu.RLock()
	defer l.mu.RUnlock()

	disallowedNodes := l.blockListCache[taskID]
	if disallowedNodes != nil {
		return disallowedNodes
	}

	return ptrs.Ptr(set.New[string]())
}

// ReportTaskDone cleans up taskID to disallowed nodes cache.
// This is safe to call multiple times and on tasks without disallowed nodes.
func (l *LogPatternPolicies) ReportTaskDone(taskID model.TaskID) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.blockListCache, taskID)
}

type retryOnDifferentNode struct {
	bun.BaseModel `bun:"table:log_policy_retry_on_different_node"`

	ID            int          `bun:"id,pk,autoincrement"`
	TaskID        model.TaskID `bun:"task_id"`
	NodeName      string       `bun:"node_name"`
	Regex         string       `bun:"regex"`
	TriggeringLog string       `bun:"triggering_log"`
	TaskEnded     bool         `bun:"task_ended"`
}

func (l *LogPatternPolicies) addRetryOnDifferentNode(
	ctx context.Context, taskID model.TaskID, nodeName, regex, triggeringLog string,
) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	m := &retryOnDifferentNode{
		TaskID:        taskID,
		NodeName:      nodeName,
		Regex:         regex,
		TriggeringLog: triggeringLog,
		TaskEnded:     false,
	}
	res, err := db.Bun().NewInsert().Model(m).
		On("CONFLICT (task_id, node_name, regex) DO NOTHING"). // Only care about the first log.
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("inserting log policy retry on different node alert %+v: %w", m, err)
	}
	if num, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("retry different node rows affected: %w", err)
	} else if num == 0 {
		return nil
	}

	tasklogger.Insert(tasklogger.CreateLogFromMaster(taskID, model.LogLevelError,
		fmt.Sprintf("(log '%q' matched regex %s) therefore will not schedule on %s\n",
			triggeringLog, regex, nodeName)))

	if _, ok := l.blockListCache[taskID]; !ok {
		l.blockListCache[taskID] = ptrs.Ptr(set.New[string]())
	}
	l.blockListCache[taskID].Insert(nodeName)
	return nil
}

// RetryInfo has information about don't retry policies that have been triggered.
type RetryInfo struct {
	Regex         string
	TriggeringLog string
}

type dontRetry struct {
	bun.BaseModel `bun:"table:log_policy_dont_retry"`

	ID            int          `bun:"id,pk,autoincrement"`
	TaskID        model.TaskID `bun:"task_id"`
	Regex         string       `bun:"regex"`
	NodeName      string       `bun:"node_name"`
	TriggeringLog string       `bun:"triggering_log"`
}

func addDontRetry(
	ctx context.Context, taskID model.TaskID, nodeName, regex, triggeringLog string,
) error {
	m := &dontRetry{
		TaskID:        taskID,
		NodeName:      nodeName,
		Regex:         regex,
		TriggeringLog: triggeringLog,
	}
	if _, err := db.Bun().NewInsert().Model(m).
		On("CONFLICT (task_id, regex) DO NOTHING"). // Only care about the first log.
		Exec(ctx); err != nil {
		return fmt.Errorf("adding don't retry policy %+v: %w", m, err)
	}

	// We don't send a log to the trial. The trial will do it if it failed.
	return nil
}

// ShouldRetry returns a list of any triggered log policies that prevent retrying a trial.
// Returns an empty list if taskID doesn't exist. Order is not guaranteed.
// Only returns first log that triggered each regex. Multiple policies with the same regex
// will only have one RetryInfo.
func ShouldRetry(ctx context.Context, taskID model.TaskID) ([]RetryInfo, error) {
	var models []*dontRetry
	if err := db.Bun().NewSelect().Model(&models).
		Where("task_id = ?", taskID).
		Scan(ctx, &models); err != nil {
		return nil, fmt.Errorf("getting taskID %s should retry: %w", taskID, err)
	}

	var out []RetryInfo
	for _, m := range models {
		out = append(out, RetryInfo{
			Regex:         m.Regex,
			TriggeringLog: m.TriggeringLog,
		})
	}

	return out, nil
}
