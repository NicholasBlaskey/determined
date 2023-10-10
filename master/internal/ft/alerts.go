package ft // rename ft

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/task/tasklogger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/set"

	"github.com/uptrace/bun"
)

var (
	blockListCache = make(map[model.TaskID]*set.Set[string])
	mu             sync.RWMutex
)

// There are two reasons for this using a cache
//  1. Avoid the possibility this feature causes a major slowdown to Scheduler
//     that won't be obvious till it run at scale.
//  2. Avoid putting possible transient db errors in the path of the Scheduler.
//
// I think there is going to be a decent chance this cache approach will somehow leak tasks
// in the future but I think even if we never removed items from the cache
// we would still probaly be okay.
func InitializeLogPatternPolicies(ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()

	var blockedNodes []*retryOnDifferentNode
	if err := db.Bun().NewSelect().Model(&blockedNodes).
		Where("task_ended = true").
		Scan(ctx, &blockedNodes); err != nil {
		return fmt.Errorf("getting blocked nodes: %w", err)
	}

	blockListCache = make(map[model.TaskID]*set.Set[string])
	for _, b := range blockedNodes {
		if _, ok := blockListCache[b.TaskID]; !ok {
			blockListCache[b.TaskID] = ptrs.Ptr(set.New[string]())
		}
		blockListCache[b.TaskID].Insert(b.NodeName)
	}

	return nil
}

// DisallowedNodes returns a list of nodes that should be blacklisted for the given allocation
func DisallowedNodes(taskID model.TaskID) *set.Set[string] {
	mu.RLock()
	defer mu.RUnlock()

	fmt.Println("DISALKLOWED NODES", taskID, blockListCache[taskID])

	disallowedNodes := blockListCache[taskID]
	if disallowedNodes != nil {
		return disallowedNodes
	}

	return ptrs.Ptr(set.New[string]())
}

// ReportTaskDone cleans up taskID to disallowed nodes cache.
// This is safe to call multiple times and on tasks without disallowed nodes.
func ReportTaskDone(taskID model.TaskID) {
	mu.Lock()
	defer mu.Unlock()

	delete(blockListCache, taskID)
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

// AddRetryOnDifferentNode comment.
func AddRetryOnDifferentNode(
	ctx context.Context, taskID model.TaskID, nodeName, regex, triggeringLog string,
) error {
	mu.Lock()
	defer mu.Unlock()

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

	// TODO make master log function in tasklogger
	tasklogger.Insert(&model.TaskLog{
		TaskID:    string(taskID),
		Timestamp: ptrs.Ptr(time.Now().UTC()),
		Level:     ptrs.Ptr(model.LogLevelError),
		Source:    ptrs.Ptr("master"),
		StdType:   ptrs.Ptr("stdout"),
		Log: fmt.Sprintf("(log '%q' matched regex %s) therefore will not schedule on %s\n",
			triggeringLog, regex, nodeName),
	})

	// TODO actually maybe here we should do the cap check on the taskID.
	// Like getAgents and decide this should be killed?

	if _, ok := blockListCache[taskID]; !ok {
		blockListCache[taskID] = ptrs.Ptr(set.New[string]())
	}
	blockListCache[taskID].Insert(nodeName)
	return nil
}

type sendWebhook struct {
	bun.BaseModel `bun:"table:log_policy_send_webhook"`

	ID            int          `bun:"id,pk,autoincrement"`
	TaskID        model.TaskID `bun:"task_id"`
	Regex         string       `bun:"regex"`
	NodeName      string       `bun:"node_name"`
	WebhookName   string       `bun:"webhook_name"`
	TriggeringLog string       `bun:"triggering_log"`
}

func AddWebhookAlert(
	ctx context.Context, taskID model.TaskID, webhookName, nodeName, regex, triggeringLog string,
) error {
	m := &sendWebhook{
		TaskID:        taskID,
		NodeName:      nodeName,
		Regex:         regex,
		TriggeringLog: triggeringLog,
		WebhookName:   webhookName,
	}
	res, err := db.Bun().NewInsert().Model(m).
		On("CONFLICT (task_id, regex, webhook_name) DO NOTHING"). // Only care about the first log.
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("adding send webhook policy %+v: %w", m, err)
	}
	if num, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("retry different node rows affected: %w", err)
	} else if num == 0 {
		return nil
	}

	// TODO make master log function in tasklogger
	tasklogger.Insert(&model.TaskLog{
		TaskID:    string(taskID),
		Timestamp: ptrs.Ptr(time.Now().UTC()),
		Level:     ptrs.Ptr(model.LogLevelError),
		Source:    ptrs.Ptr("master"),
		StdType:   ptrs.Ptr("stdout"),
		Log: fmt.Sprintf("(log '%q' matched regex %s) therefore sent webhook to webhook name %s\n",
			triggeringLog, regex, webhookName),
	})

	// TODO actually send this webhook

	return nil
}

type dontRetry struct {
	bun.BaseModel `bun:"table:log_policy_dont_retry"`

	ID            int          `bun:"id,pk,autoincrement"`
	TaskID        model.TaskID `bun:"task_id"`
	Regex         string       `bun:"regex"`
	NodeName      string       `bun:"node_name"`
	TriggeringLog string       `bun:"triggering_log"`
}

// AddDontRetry comment.
func AddDontRetry(
	ctx context.Context, taskID model.TaskID, nodeName, regex, triggeringLog string,
) error {
	// First taskID, nodeName, regex, triggeringLog combo?
	// How do we dedupe it? We only really want one trigger per config???
	m := &dontRetry{
		TaskID:        taskID,
		NodeName:      nodeName,
		Regex:         regex,
		TriggeringLog: triggeringLog,
	}
	if _, err := db.Bun().NewInsert().Model(m).Exec(ctx); err != nil {
		return fmt.Errorf("adding don't retry policy %+v: %w", m, err)
	}

	return nil
}

type RetryInfo struct {
	Regex         string
	TriggeringLog string // TODO this could be a model.Log but just the string I think is fine for now.
}

// ShouldRetry comment.
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
			Regex: m.Regex,
			// model.Log would be cool since it has like containerID. and nodeName / podID.
			// I think this is fine for now.
			TriggeringLog: m.TriggeringLog,
		})
	}

	return out, nil
}

/*
type RetryInfo struct {
	Regex string
	Log   string // TODO this could be a model.Log but just the string I think is fine for now.
}

func ShouldRetryOnDifferentNode(taskID model.TaskID) ([]RetryDifferentNodeInfo, error) {
}
*/
