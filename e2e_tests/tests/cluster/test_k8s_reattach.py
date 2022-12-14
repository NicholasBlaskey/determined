import pytest
import requests

from .utils import (
    command_succeeded,
    get_command_info,
    run_command,
    wait_for_command_state,
    wait_for_task_state,
)
from .managed_cluster import get_agent_data
from tests import config as conf
from tests import command as cmd
from tests.cluster.test_users import det_spawn
from determined.common.api import authentication
from determined.common import constants
from tests import experiment as exp
from determined.common.api.bindings import determinedexperimentv1State as EXP_STATE

import time
from kubernetes import client, config, watch

class ManagedK8sCluster:
    def __init__(self) -> None:
        # Configs can be set in Configuration class directly or using helper utility
        config.load_kube_config()
        self.v1 = client.AppsV1Api()
        #self._scale_master(up=True)
        #self._scale_master(up=False)

    def master_down_for(self, downtime: int) -> None:
        pass

    def kill_master(self) -> None:
        self._scale_master(up=False)

    def restart_master(self) -> None:
        self._scale_master(up=True)
    
    def _scale_master(self, up: bool) -> None:
        desired_pods = 0
        if up:
            desired_pods = 1
            
        ret = self.v1.list_deployment_for_all_namespaces(watch=False)
        master_deployment_list = [
            d for d in ret.items if "determined-master-deployment" in d.metadata.name
        ]
        assert len(master_deployment_list) == 1
        master_deployment = master_deployment_list[0]

        replicas = master_deployment.status.available_replicas
        if (up and replicas == 1) or (not up and replicas is None):        
            print(f"master already scaled {'up' if up else 'down'}")
            return
        
        patch = [{"op": "add", "path": "/spec/replicas", "value": desired_pods}]
        self.v1.patch_namespaced_deployment_scale(
            name=master_deployment.metadata.name,
            namespace=master_deployment.metadata.namespace,
            body=patch,
        )

        # Wait for pod to complete scale.
        w = watch.Watch()
        for event in w.stream(self.v1.list_deployment_for_all_namespaces, _request_timeout=360):
            if event["object"].metadata.name != master_deployment.metadata.name:
                continue

            replicas = event["object"].status.available_replicas
            print(f"Got event of master deployment updated available_replicas = {replicas}")
            if (up and replicas == 1) or (not up and replicas is None):
                print(f"master pods scaled {'up' if up else 'down'}")                
                w.stop()

        if not up:
            return 

        # Wait for determined to be up.
        WAIT_FOR_UP = 30
        for _ in range(WAIT_FOR_UP):
            try:
                assert len(get_agent_data(conf.make_master_url())) > 0
                return
            except Exception as e:
                print(f"Can't reach master, retrying again {e}")
                time.sleep(1)
        pytest.fail(f"Unable to reach master after {WAIT_FOR_UP} seconds")
            
    # Wait until at replicas (timeout)
    
@pytest.fixture
def restartable_managed_cluster() -> ManagedK8sCluster:
    cluster = ManagedK8sCluster()
    cluster._scale_master(up=True)
    yield cluster
    cluster._scale_master(up=True)
    

@pytest.mark.e2e_k8s
@pytest.mark.parametrize("slots", [0])
@pytest.mark.parametrize("downtime", [60])
def test_k8s_master_restart_command(
    restartable_managed_cluster: ManagedK8sCluster, slots: int, downtime: int,
) -> None:
    command_id = run_command(30, slots)
    wait_for_command_state(command_id, "RUNNING", 10)

    if downtime >= 0:
        managed_cluster.kill_master()
        time.sleep(downtime)
        managed_cluster.restart_master()

    wait_for_command_state(command_id, "TERMINATED", 30)
    succeeded = "success" in get_command_info(command_id)["exitStatus"]
    assert succeeded

    
@pytest.mark.e2e_k8s
@pytest.mark.parametrize("downtime", [5])
def test_k8s_master_restart_shell(
        restartable_managed_cluster: ManagedK8sCluster, downtime: int,
) -> None:
    managed_cluster = restartable_managed_cluster

    with cmd.interactive_command("shell", "start", "--detach") as shell:
        task_id = shell.task_id

        assert task_id is not None
        wait_for_task_state("shell", task_id, "RUNNING")

        if downtime >= 0:
            managed_cluster.kill_master()
            time.sleep(downtime)
            managed_cluster.restart_master()

        wait_for_task_state("shell", task_id, "RUNNING")

        child = det_spawn(["shell", "open", task_id])
        child.setecho(True)
        child.expect(r".*Permanently added.+([0-9a-f-]{36}).+known hosts\.")
        child.sendline("det user whoami")
        child.expect("You are logged in as user \\'(.*)\\'", timeout=10)
        child.sendline("exit")
        child.read()
        child.wait()
        assert child.exitstatus == 0

def _get_auth_token_for_curl() -> str:
    token = authentication.TokenStore(conf.make_master_url()).get_token(
        constants.DEFAULT_DETERMINED_USER
    )
    assert token is not None
    return token


def _check_web_url(url: str, name: str) -> None:
    token = _get_auth_token_for_curl()
    bad_status_codes = []
    for _ in range(10):
        r = requests.get(url, headers={"Authorization": f"Bearer {token}"}, allow_redirects=True)
        # Sometimes the TB/JL take a bit of time to stand up, returning 502.
        # Sometimes it takes a bit of time for master to register the proxy, returning 404.
        if r.status_code == 502 or r.status_code == 404:
            time.sleep(1)
            bad_status_codes.append(r.status_code)
            continue
        r.raise_for_status()
        html = r.content.decode("utf-8")
        assert name in html  # Brutal.
        break
    else:
        error_msg = ",".join(str(v) for v in bad_status_codes)
        pytest.fail(f"{name} {url} got error codes: {error_msg}")


def _check_notebook_url(url: str) -> None:
    return _check_web_url(url, "JupyterLab")


def _check_tb_url(url: str) -> None:
    return _check_web_url(url, "TensorBoard")

#TODO abstract
@pytest.mark.e2e_k8s
@pytest.mark.parametrize("downtime", [5])
def test_k8s_master_restart_notebook(
    restartable_managed_cluster: ManagedK8sCluster, downtime: int
) -> None:
    managed_cluster = restartable_managed_cluster
    with cmd.interactive_command("notebook", "start", "--detach") as notebook:
        task_id = notebook.task_id
        assert task_id is not None
        wait_for_task_state("notebook", task_id, "RUNNING")
        notebook_url = f"{conf.make_master_url()}proxy/{task_id}/"
        _check_notebook_url(notebook_url)

        if downtime >= 0:
            managed_cluster.kill_master()
            time.sleep(downtime)
            managed_cluster.restart_master()

        _check_notebook_url(notebook_url)

        print("notebook ok")


@pytest.mark.e2e_k8s
@pytest.mark.parametrize("downtime", [5])
def test_k8s_master_restart_tensorboard(
    restartable_managed_cluster: ManagedK8sCluster, downtime: int
) -> None:
    managed_cluster = restartable_managed_cluster

    exp_id = exp.create_experiment(
        conf.fixtures_path("no_op/single.yaml"),
        conf.fixtures_path("no_op"),
        None,
    )

    exp.wait_for_experiment_state(exp_id, EXP_STATE.STATE_COMPLETED, max_wait_secs=60)

    with cmd.interactive_command("tensorboard", "start", "--detach", str(exp_id)) as tb:
        task_id = tb.task_id
        assert task_id is not None
        wait_for_task_state("tensorboard", task_id, "RUNNING")

        tb_url = f"{conf.make_master_url()}proxy/{task_id}/"
        _check_tb_url(tb_url)

        if downtime >= 0:
            managed_cluster.kill_master()
            time.sleep(downtime)
            managed_cluster.restart_master()

        _check_tb_url(tb_url)

        print("tensorboard ok")
        
