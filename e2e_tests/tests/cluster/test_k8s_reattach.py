import pytest

from .utils import (
    command_succeeded,
    get_command_info,
    run_command,
    wait_for_command_state,
    wait_for_task_state,
)
from .managed_cluster import get_agent_data
from tests import config as conf


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
def managed_cluster():
    cluster = ManagedK8sCluster()
    cluster._scale_master(up=True)
    yield cluster
    cluster._scale_master(up=True)
    

@pytest.mark.e2e_k8s
@pytest.mark.parametrize("slots", [0])
@pytest.mark.parametrize("downtime", [60])
def test_k8s_master_restart_command(
    managed_cluster: ManagedK8sCluster, slots: int, downtime: int,
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
