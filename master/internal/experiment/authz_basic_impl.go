package user

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
)

// ExperimentAuthZBasic is basic OSS controls.
type ExperimentAuthZBasic struct{}

/*
func CanGetExperiment(curUser model.User) error {
}
*/

func (a *ExperimentAuthZBasic) CanDeleteExperiment(curUser model.User, e *model.Experiment) error {
	curUserIsOwner := e.OwnerID == nil || *e.OwnerID == curUser.ID
	if !curUser.Admin && !curUserIsOwner {
		return fmt.Errorf("non admin users may not delete other user's experiments")
	}
	return nil
}

func init() {
	AuthZProvider.Register("basic", &ExperimentAuthZBasic{})
}
