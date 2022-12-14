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

from .managed_cluster import Cluster

class ManagedK8sCluster(Cluster):
    def __init__(self) -> None:
        config.load_kube_config()
        self.v1 = client.AppsV1Api()
        self._scale_master(up=True)

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
            

@pytest.fixture
def k8s_managed_cluster() -> ManagedK8sCluster:
    cluster = ManagedK8sCluster()
    cluster._scale_master(up=True)
    yield cluster
    cluster._scale_master(up=True)
