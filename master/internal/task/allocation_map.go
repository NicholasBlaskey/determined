package task

import (
	"sync"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

var allocationMap map[model.AllocationID]*actor.Ref
var allocationMapMutex sync.RWMutex

func InitAllocationMap() {
	allocationMap = map[model.AllocationID]*actor.Ref{}
}

func GetAllocation(allocationID model.AllocationID) *actor.Ref {
	allocationMapMutex.RLock()
	defer allocationMapMutex.RUnlock()
	return allocationMap[allocationID]
}

func RegisterAllocation(allocationID model.AllocationID, ref *actor.Ref) {
	allocationMapMutex.Lock()
	defer allocationMapMutex.Unlock()
	allocationMap[allocationID] = ref
}

func DeregisterAllocation(allocationID model.AllocationID) {
	allocationMapMutex.Lock()
	defer allocationMapMutex.Unlock()
	delete(allocationMap, allocationID)
}
