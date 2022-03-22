import time
import logging

import pytest

from determined.common.api.bindings import determinedexperimentv1State as EXP_STATE
from tests import config as conf
from tests import experiment as exp

from .managed_cluster import ManagedCluster
from .utils import command_succeeded, run_command, wait_for_command_state

logger = logging.getLogger(__name__)


def _sanity_check(managed_cluster: ManagedCluster) -> None:
    if not managed_cluster.reattach:
        pytest.skip()

    managed_cluster.ensure_agent_ok()

@pytest.mark.managed_devcluster
def test_master_restart_ok(managed_cluster: ManagedCluster) -> None:
    # - Kill master
    # - Restart master
    # - Schedule something.
    # Do it twice to ensure nothing gets stuck.
    _sanity_check(managed_cluster)

    try:
        for i in range(2):
            print("test_master_restart_ok stage %s start" % i)
            managed_cluster.kill_master()
            managed_cluster.restart_master()

            cmd_ids = [run_command(1, slots) for slots in [0, 1]]

            for cmd_id in cmd_ids:
                wait_for_command_state(cmd_id, "TERMINATED", 10)
                assert command_succeeded(cmd_id)
            print("test_master_restart_ok stage %s done" % i)
    except Exception:
        managed_cluster.restart_master()
        managed_cluster.restart_agent()
        raise


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("downtime", [0, 20, 60])
def test_master_restart_reattach_recover_experiment(
        managed_cluster: ManagedCluster, downtime: int) -> None:
    _sanity_check(managed_cluster)

    try:
        exp_id = exp.create_experiment(
            conf.fixtures_path("no_op/single-medium-train-step.yaml"),
            conf.fixtures_path("no_op"),
            None,
        )

        # TODO(ilia): don't wait for progress.
        exp.wait_for_experiment_workload_progress(exp_id)

        if downtime >= 0:
            managed_cluster.kill_master()
            time.sleep(downtime)
            managed_cluster.restart_master()

        exp.wait_for_experiment_state(
            exp_id, EXP_STATE.STATE_COMPLETED, max_wait_secs=downtime + 60)
        trials = exp.experiment_trials(exp_id)

        assert len(trials) == 1
        train_wls = exp.workloads_with_training(trials[0].workloads)
        assert len(train_wls) == 5
    except Exception:
        managed_cluster.restart_master()
        managed_cluster.restart_agent()
        raise
