
import pytest


from determined.common.api.bindings import experimentv1State
from tests import experiment as exp
from tests import config as conf

from .test_groups import det_cmd

@pytest.mark.e2e_cpu
def test_continue_fixing_broken_config() -> None:
    exp_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        ["--config", "hyperparameters.metrics_sigma=-1.0"],
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)

    # TODO make a continue expect error / continue expect pass thing that logs task logs
    # for circleci.
    
    # RUNNING exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR
    det_cmd(["e", "continue", str(exp_id), "--config", "hyperparameters.metrics_sigma=-1.0"],
            check=True)
    exp.wait_for_experiment_state(exp_id, experimentv1State.COMPLETED)
