
import pytest

import time

from determined.common.api import authentication, bindings, certs
from determined.common.api.bindings import experimentv1State
from tests import experiment as exp
from tests import config as conf
from tests import api_utils

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

@pytest.mark.e2e_cpu
def test_continue_trial_time() -> None:
    exp_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        ["--config", "hyperparameters.metrics_sigma=-1.0"], # TODO why does this fail on continue?
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)

    sess = api_utils.determined_test_session()
    def exp_start_end_time():
        e = bindings.get_GetExperiment(sess, experimentId=exp_id).experiment
        return e.startTime, e.endTime
    def trial_start_end_time():
        experiment_trials = exp.experiment_trials(exp_id)
        assert len(experiment_trials) == 1
        return experiment_trials[0].trial.startTime, experiment_trials[0].trial.endTime
    

    exp_orig_start, exp_orig_end = exp_start_end_time()
    trial_orig_start, trial_orig_end = trial_start_end_time()

    # This is technically a race. So maybe do IN PROGRESS then completed? IDK Oh it isn't a race
    # We might be good.
    det_cmd(["e", "continue", str(exp_id)], check=True)
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)

    exp_new_start, exp_new_end = exp_start_end_time()
    trial_new_start, trial_new_end = trial_start_end_time()

    # I don't love this behaviour.
    assert exp_orig_start == exp_new_start
    assert trial_orig_start == trial_new_start
    
    assert exp_new_end > exp_orig_end
    assert trial_new_end > trial_orig_end

    # Task times are updated.
    experiment_trials = exp.experiment_trials(exp_id)
    assert len(experiment_trials) == 1
    taskIds = experiment_trials[0].trial.taskIds
    assert len(taskIds) == 2
    
    task = bindings.get_GetTask(sess, taskId=taskIds[1]).task
    assert task.startTime > exp_orig_end
    assert task.endTime > task.startTime


@pytest.mark.e2e_cpu
def test_continue_batches() -> None:
    # Experiment fails before first checkpoint.
    exp_id = exp.create_experiment(
        conf.fixtures_path("mnist_pytorch/failable.yaml"),
        conf.fixtures_path("mnist_pytorch"),
        ["--config", "environment.environment_variables=['FAIL_AT_BATCH=2']"],
         # TODO why does this fail on continue?
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)

    sess = api_utils.determined_test_session()
    trials = exp.experiment_trials(exp_id)
    assert len(trials) == 1
    trial_id = trials[0].trial.id
    
    def assert_exited_at(batch_idx):
        assert f"failed at this batch {batch_idx}" in "\n".join(exp.trial_logs(trial_id))
    assert_exited_at(2)

    def get_metric_list():
        resp_list = bindings.get_GetValidationMetrics(sess, trialIds=[trial_id])
        return [metric for resp in resp_list for metric in resp.metrics]
    metrics = get_metric_list()
    assert len(metrics) == 2

    first_metric_ids = []
    i = 1
    for m in metrics:
        first_metric_ids.append(m.id)
        assert m.totalBatches == i
        i += 1

    # Experiment has to start over since we didn't checkpoint.
    # We must invalidate all previous reported metrics.
    # This time experiment makes it a validation after the first checkpoint.
    det_cmd(["e", "continue", str(exp_id),
             "--config", "environment.environment_variables=['FAIL_AT_BATCH=5']"], check=True)
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)
    assert_exited_at(5)

    # We don't want to checkpoint any of this...
    
    second_metric_ids = []
    metrics = get_metric_list()
    assert len(metrics) == 5
    i = 1
    for m in metrics:
        assert m.id not in first_metric_ids # Invalidated first metrics.
        second_metric_ids.append(m.id)
        assert m.totalBatches == i
        i += 1
        
    

    # We lose one metric since we are continuing from first checkpoint.
    # We correctly stop at total_batches.


    # Our validations will get invalidated when rerun.
    # TODO get validations

    # TODO checkpoint at 101
    # REMOVE the 101 fail at environemtn variable (CAN we do that???) (override to 0?)
    #
    # Then like just see what we do... based on metrics. Really we just want intutive behaviour
    # Like really the searcher should be as good as we can be...
