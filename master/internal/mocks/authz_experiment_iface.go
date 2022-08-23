// Code generated by mockery v2.13.1. DO NOT EDIT.

package mocks

import (
	bun "github.com/uptrace/bun"

	checkpointv1 "github.com/determined-ai/determined/proto/pkg/checkpointv1"

	mock "github.com/stretchr/testify/mock"

	model "github.com/determined-ai/determined/master/pkg/model"

	projectv1 "github.com/determined-ai/determined/proto/pkg/projectv1"
)

// ExperimentAuthZ is an autogenerated mock type for the ExperimentAuthZ type
type ExperimentAuthZ struct {
	mock.Mock
}

// CanActivateExperiment provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanActivateExperiment(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanArchiveExperiment provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanArchiveExperiment(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanCancelExperiment provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanCancelExperiment(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanComputeHPImportance provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanComputeHPImportance(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanCreateExperiment provides a mock function with given fields: curUser, proj, e
func (_m *ExperimentAuthZ) CanCreateExperiment(curUser model.User, proj *projectv1.Project, e *model.Experiment) error {
	ret := _m.Called(curUser, proj, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *projectv1.Project, *model.Experiment) error); ok {
		r0 = rf(curUser, proj, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanDeleteExperiment provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanDeleteExperiment(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanForkFromExperiment provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanForkFromExperiment(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanGetBestSearcherValidationMetric provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanGetBestSearcherValidationMetric(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanGetExperiment provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanGetExperiment(curUser model.User, e *model.Experiment) (bool, error) {
	ret := _m.Called(curUser, e)

	var r0 bool
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) bool); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(model.User, *model.Experiment) error); ok {
		r1 = rf(curUser, e)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CanGetExperimentValidationHistory provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanGetExperimentValidationHistory(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanGetExperimentsCheckpointsToGC provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanGetExperimentsCheckpointsToGC(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanGetHPImportance provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanGetHPImportance(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanGetMetricBatches provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanGetMetricBatches(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanGetMetricNames provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanGetMetricNames(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanGetModelDef provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanGetModelDef(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanGetModelDefFile provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanGetModelDefFile(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanGetModelDefTree provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanGetModelDefTree(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanGetTrialsSample provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanGetTrialsSample(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanGetTrialsSnapshot provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanGetTrialsSnapshot(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanKillExperiment provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanKillExperiment(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanMoveExperiment provides a mock function with given fields: curUser, from, to, e
func (_m *ExperimentAuthZ) CanMoveExperiment(curUser model.User, from *projectv1.Project, to *projectv1.Project, e *model.Experiment) error {
	ret := _m.Called(curUser, from, to, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *projectv1.Project, *projectv1.Project, *model.Experiment) error); ok {
		r0 = rf(curUser, from, to, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanPauseExperiment provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanPauseExperiment(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanPreviewHPSearch provides a mock function with given fields: curUser
func (_m *ExperimentAuthZ) CanPreviewHPSearch(curUser model.User) error {
	ret := _m.Called(curUser)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User) error); ok {
		r0 = rf(curUser)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanSetExperimentsCheckpointGCPolicy provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanSetExperimentsCheckpointGCPolicy(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanSetExperimentsDescription provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanSetExperimentsDescription(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanSetExperimentsLabels provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanSetExperimentsLabels(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanSetExperimentsMaxSlots provides a mock function with given fields: curUser, e, slots
func (_m *ExperimentAuthZ) CanSetExperimentsMaxSlots(curUser model.User, e *model.Experiment, slots int) error {
	ret := _m.Called(curUser, e, slots)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment, int) error); ok {
		r0 = rf(curUser, e, slots)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanSetExperimentsName provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanSetExperimentsName(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanSetExperimentsNotes provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanSetExperimentsNotes(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanSetExperimentsPriority provides a mock function with given fields: curUser, e, priority
func (_m *ExperimentAuthZ) CanSetExperimentsPriority(curUser model.User, e *model.Experiment, priority int) error {
	ret := _m.Called(curUser, e, priority)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment, int) error); ok {
		r0 = rf(curUser, e, priority)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanSetExperimentsWeight provides a mock function with given fields: curUser, e, weight
func (_m *ExperimentAuthZ) CanSetExperimentsWeight(curUser model.User, e *model.Experiment, weight float64) error {
	ret := _m.Called(curUser, e, weight)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment, float64) error); ok {
		r0 = rf(curUser, e, weight)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanUnarchiveExperiment provides a mock function with given fields: curUser, e
func (_m *ExperimentAuthZ) CanUnarchiveExperiment(curUser model.User, e *model.Experiment) error {
	ret := _m.Called(curUser, e)

	var r0 error
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment) error); ok {
		r0 = rf(curUser, e)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// FilterCheckpoints provides a mock function with given fields: curUser, e, checkpoints
func (_m *ExperimentAuthZ) FilterCheckpoints(curUser model.User, e *model.Experiment, checkpoints []*checkpointv1.Checkpoint) ([]*checkpointv1.Checkpoint, error) {
	ret := _m.Called(curUser, e, checkpoints)

	var r0 []*checkpointv1.Checkpoint
	if rf, ok := ret.Get(0).(func(model.User, *model.Experiment, []*checkpointv1.Checkpoint) []*checkpointv1.Checkpoint); ok {
		r0 = rf(curUser, e, checkpoints)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*checkpointv1.Checkpoint)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(model.User, *model.Experiment, []*checkpointv1.Checkpoint) error); ok {
		r1 = rf(curUser, e, checkpoints)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FilterExperimentLabelsQuery provides a mock function with given fields: curUser, proj, query
func (_m *ExperimentAuthZ) FilterExperimentLabelsQuery(curUser model.User, proj *projectv1.Project, query *bun.SelectQuery) (*bun.SelectQuery, error) {
	ret := _m.Called(curUser, proj, query)

	var r0 *bun.SelectQuery
	if rf, ok := ret.Get(0).(func(model.User, *projectv1.Project, *bun.SelectQuery) *bun.SelectQuery); ok {
		r0 = rf(curUser, proj, query)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bun.SelectQuery)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(model.User, *projectv1.Project, *bun.SelectQuery) error); ok {
		r1 = rf(curUser, proj, query)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FilterExperiments provides a mock function with given fields: curUser, project, experiments
func (_m *ExperimentAuthZ) FilterExperiments(curUser model.User, project *projectv1.Project, experiments []*model.Experiment) ([]*model.Experiment, error) {
	ret := _m.Called(curUser, project, experiments)

	var r0 []*model.Experiment
	if rf, ok := ret.Get(0).(func(model.User, *projectv1.Project, []*model.Experiment) []*model.Experiment); ok {
		r0 = rf(curUser, project, experiments)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*model.Experiment)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(model.User, *projectv1.Project, []*model.Experiment) error); ok {
		r1 = rf(curUser, project, experiments)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewExperimentAuthZ interface {
	mock.TestingT
	Cleanup(func())
}

// NewExperimentAuthZ creates a new instance of ExperimentAuthZ. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewExperimentAuthZ(t mockConstructorTestingTNewExperimentAuthZ) *ExperimentAuthZ {
	mock := &ExperimentAuthZ{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
