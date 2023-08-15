
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
    det_cmd(["e", "continue", str(exp_id), "--config", "hyperparameters.metrics_sigma=1.0"],
            check=True)
    exp.wait_for_experiment_state(exp_id, experimentv1State.COMPLETED)

    trials = exp.experiment_trials(exp_id)
    assert len(trials) == 1

    # Trial logs show both failed and sucessful run.
    trial_logs = "\n".join(exp.trial_logs(trials[0].trial.id))
    assert "assert 0 <= self.metrics_sigma" in trial_logs

@pytest.mark.e2e_cpu
def test_continue_max_restart() -> None:
    exp_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        ["--config", "hyperparameters.metrics_sigma=-1.0", "--config", "max_restarts=2"],
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)

    trials = exp.experiment_trials(exp_id)
    assert len(trials) == 1
    def count_times_ran():
        return "\n".join(exp.trial_logs(trials[0].trial.id)).count("assert 0 <= self.metrics_sigma")
    def get_trial_restarts():
        experiment_trials = exp.experiment_trials(exp_id)
        assert len(experiment_trials) == 1
        return experiment_trials[0].trial.restarts
        
    assert count_times_ran() == 3
    assert get_trial_restarts() == 2
    
    det_cmd(["e", "continue", str(exp_id)], check=True)
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)
    assert count_times_ran() == 6
    assert get_trial_restarts() == 2
    
    det_cmd(["e", "continue", str(exp_id), "--config", "max_restarts=1"], check=True)
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)
    assert count_times_ran() == 8
    assert get_trial_restarts() == 1

    
    # TODO assert
    # that
    # 
    # in logs aka we continue logs...
