
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


# TODO continue core API sucess
    
@pytest.mark.e2e_cpu
def test_continue_batches() -> None:
    # Experiment fails before first checkpoint.
    exp_id = exp.create_experiment(
        conf.fixtures_path("mnist_pytorch/failable.yaml"),
        conf.fixtures_path("mnist_pytorch"),
        ["--config", "environment.environment_variables=['FAIL_AT_BATCH=2']"],
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
    det_cmd(["e", "continue", str(exp_id),
             "--config", "environment.environment_variables=['FAIL_AT_BATCH=-1']"], check=True)
    exp.wait_for_experiment_state(exp_id, experimentv1State.COMPLETED)

    metrics = get_metric_list()
    assert len(metrics) == 8
    i = 1
    for m in metrics:
        if m.totalBatches <= 3:
            assert m.id in second_metric_ids
        else:
            assert m.id not in second_metric_ids
        assert m.totalBatches == i
        i += 1

        
@pytest.mark.e2e_cpu
@pytest.mark.parametrize("continue_max_length,expected_total_batches", [(2, 4), (3, 6), (6, 6)])
def test_continue_completed_searcher(continue_max_length: int, expected_total_batches: int) -> None:
    exp_id = exp.create_experiment(
        conf.fixtures_path("mnist_pytorch/failable.yaml"),
        conf.fixtures_path("mnist_pytorch"),
        ["--config", "searcher.max_length.batches=3"],
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.COMPLETED)
    
    sess = api_utils.determined_test_session()
    trials = exp.experiment_trials(exp_id)
    assert len(trials) == 1
    trial_id = trials[0].trial.id
    
    def max_total_batches():
        resp_list = bindings.get_GetValidationMetrics(sess, trialIds=[trial_id])
        return max([metric for resp in resp_list for metric in resp.metrics],
                   key=lambda x: x.totalBatches).totalBatches
    assert max_total_batches() == 3

    # Train for less time.
    det_cmd(["e", "continue", str(exp_id),
             # This merging feels really bad. TODO fix this. SOMETHING is wrong
             "--config", f"searcher.max_length.batches={continue_max_length}",
             "--config", "searcher.name=single"], check=True)
    exp.wait_for_experiment_state(exp_id, experimentv1State.COMPLETED)

    # TODO should both be 3. I don't know why we keep training. This might be okay as a follow
    # up ticket.
    assert max_total_batches() == expected_total_batches
    
    # This feels super bad.
    # CAN WE PR this? seperately

    '''
2023-08-21T20:31:31.061122Z] 4d80fe39 || BATCH_IDX 3 EPOCH IDX 0
[2023-08-21T20:31:31.077307Z] 4d80fe39 || /opt/conda/lib/python3.8/site-packages/torch/nn/functional.py:1331: UserWarning: dropout2d: Received a 2-D input to dropout2d, which is deprecated and will result in an error in a future release. To retain the behavior and silence this warning, please use dropout instead. Note that dropout2d exists to provide channel-wise dropout on inputs with 2 spatial dimensions, a channel dimension, and an optional batch dimension (i.e. 3D or 4D inputs).
[2023-08-21T20:31:31.077318Z] 4d80fe39 ||   warnings.warn(warn_msg)
[2023-08-21T20:31:31.077718Z] 4d80fe39 || /opt/conda/lib/python3.8/site-packages/torch/nn/modules/container.py:139: UserWarning: Implicit dimension choice for log_softmax has been deprecated. Change the call to include dim=X as an argument.
[2023-08-21T20:31:31.077723Z] 4d80fe39 ||   input = module(input)
[2023-08-21T20:31:31.112191Z] 4d80fe39 || PRITNING value / step_num 2 4
[2023-08-21T20:31:31.112200Z] 4d80fe39 || PRITNING value / step_num 1 4
[2023-08-21T20:31:31.112202Z] 4d80fe39 || PRITNING value / step_num 3 4
[2023-08-21T20:31:31.112203Z] 4d80fe39 || PRITNING value / step_num 100 4
[2023-08-21T20:31:31.323216Z] 4d80fe39 || 2023-08-21 20:31:31.323051: I tensorflow/core/pl
    '''

    '''
[2023-08-21T20:32:41.411412Z] 834ade79 || BATCH_IDX 3 EPOCH IDX 0
[2023-08-21T20:32:41.428153Z] 834ade79 || /opt/conda/lib/python3.8/site-packages/torch/nn/functional.py:1331: UserWarning: dropout2d: Received a 2-D input to dropout2d, which is deprecated and will result in an error in a future release. To retain the behavior and silence this warning, please use dropout instead. Note that dropout2d exists to provide channel-wise dropout on inputs with 2 spatial dimensions, a channel dimension, and an optional batch dimension (i.e. 3D or 4D inputs).
[2023-08-21T20:32:41.428165Z] 834ade79 ||   warnings.warn(warn_msg)
[2023-08-21T20:32:41.428618Z] 834ade79 || /opt/conda/lib/python3.8/site-packages/torch/nn/modules/container.py:139: UserWarning: Implicit dimension choice for log_softmax has been deprecated. Change the call to include dim=X as an argument.
[2023-08-21T20:32:41.428621Z] 834ade79 ||   input = module(input)
[2023-08-21T20:32:41.461256Z] 834ade79 || PRITNING value / step_num 3 4
[2023-08-21T20:32:41.461266Z] 834ade79 || PRITNING value / step_num 1 4
[2023-08-21T20:32:41.461268Z] 834ade79 || PRITNING value / step_num 3 4
[2023-08-21T20:32:41.461436Z] 834ade79 || PRITNING value / step_num 100 4
[2023-08-21T20:32:41.668721Z] 834ade79 || 2023-08-21 20:32:41.668550: I tensorflow/core/platform/cpu_feature_guard.cc:193] This TensorFlow binary is optimized with oneAPI Deep Neural Network Library (oneDNN) to use the following CPU instructions in performance-critical operations:  AVX2 FMA
[2023-08-21T20:32:41.668728Z] 834ade79 || To enable them in other operations, rebuild TensorFlow with the appropriate compiler flags.
[2023-08-21T20:32:43.277179Z] 834ade79 || /opt/conda/lib/python3.8/site-packages/torch/nn/functional.py:1331: UserWarning: dropout2d: Received a 2-D input to dropout2d, which is deprecated and will result in an error in a future release. To retain the behavior and silence this warning, please use dropout instead. Note that dropout2d exists to provide channel-wise dropout on inputs with 2 spatial dimensions, a channel dimension, and an optional batch dimension (i.e. 3D or 4D inputs).
[2023-08-21T20:32:43.277185Z] 834ade79 ||   warnings.warn(warn_msg)
[2023-08-21T20:32:43.277225Z] 834ade79 || /opt/conda/lib/python3.8/site-packages/torch/nn/modules/container.py:139: UserWarning: Implicit dimension choice for log_softmax has been deprecated. Change the call to include dim=X as an argument.
[2023-08-21T20:32:43.277229Z] 834ade79 ||   input = module(input)
[2023-08-21T20:32:47.336381Z] 834ade79 || INFO: [33] root: validated: 10000 records in 4.08s (2451.0 records/s), in 157 batches (38.48 batches/s)
[2023-08-21T20:32:47.396243Z] 834ade79 || BATCH_IDX 4 EPOCH IDX 0
[2023-08-21T20:32:47.478026Z] 834ade79 || PRITNING value / step_num 3 5
[2023-08-21T20:32:47.478037Z] 834ade79 || PRITNING value / step_num 1 5
[2023-08-21T20:32:47.478038Z] 834ade79 || PRITNING value / step_num 3 5
[2023-08-21T20:32:47.478040Z] 834ade79 || PRITNING value / step_num 100 5
[2023-08-21T20:32:53.275287Z] 834ade79 || INFO: [33] root: validated: 10000 records in 5.778s (1731.0 records/s), in 157 batches (27.17 batches/s)
[2023-08-21T20:32:53.319643Z] 834ade79 || BATCH_IDX 5 EPOCH IDX 0
[2023-08-21T20:32:53.400110Z] 834ade79 || PRITNING value / step_num 3 6
[2023-08-21T20:32:53.400120Z] 834ade79 || PRITNING value / step_num 1 6
[2023-08-21T20:32:53.400121Z] 834ade79 || PRITNING value / step_num 3 6
[2023-08-21T20:32:53.400245Z] 834ade79 || PRITNING value / step_num 100 6
[2023-08-21T20:32:59.140884Z] 834ade79 || INFO: [33] root: validated: 10000 records in 5.718s (1749.0 records/s), in 157 batches (27.46 batches/s)
[2023-08-21T20:32:59.211753Z] 834ade79 || INFO: [33] determined.core: Reported checkpoint to master 86dca8dd-b3de-42f9-844c-7992276b4834
[2023-08-21T20:33:00.170613Z] 834ade79 || INFO: resources exited successfully with a zero exit code
[2023-08-21T20:33:00.175160Z]          || INFO: Trial 478 (Experiment 194) was terminated: allocation preempted after all resources exited: resources exited successfully with a zero exit code
Trial log stream ended. To reopen log stream, run: det trial logs -f 478
    '''
    
