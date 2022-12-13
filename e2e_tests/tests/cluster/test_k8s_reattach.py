import pytest

from .utils import (
    command_succeeded,
    get_command_info,
    run_command,
    wait_for_command_state,
    wait_for_task_state,
)

from kubernetes import client, config, watch


class ManagedK8sCluster:
    def __init__(self) -> None:
        # Configs can be set in Configuration class directly or using helper utility
        config.load_kube_config()
        self.v1 = client.AppsV1Api()
        #self._scale_master(up=True)
        self._scale_master(up=False)

    def master_down_for(self, downtime: int) -> None:
        pass

    def _scale_master(self, up: bool) -> None:
        desired_pods = 0
        if up:
            desired_pods = 1
            
        ret = self.v1.list_deployment_for_all_namespaces(watch=False)
        master_deployment = [
            d for d in ret.items if "determined-master-deployment" in d.metadata.name
        ][0]
        print(master_deployment)

        replicas = master_deployment.status.available_replicas
        if (up and replicas == 1) or (not up and replicas is None):        
            print("master already scaled up")
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
                print("oTHER EVENT")
                continue
            print("Got event")

            #self.v.read_namespaced_deployment_status(master_deployment.meta)

            
            replicas = event["object"].status.available_replicas
            print(replicas)
            if (up and replicas == 1) or (not up and replicas is None):
                print("master scaled to desired pods")
                w.stop()

        print("ENDL")
        # Wait for determined master to take requests.

                
        raise Exception("BLEH")
        # status: available_replicas
            
        

    # Wait until at replicas (timeout)
    
@pytest.fixture
def managed_k8s_cluster():
    return ManagedK8sCluster()
    

@pytest.mark.e2e_k8s
def test_k8s_master_restart_command(managed_k8s_cluster: ManagedK8sCluster) -> None:
    downtime = 10
    command_id = run_command(30, 1)
    wait_for_command_state(command_id, "RUNNING", 10)

    k8s_cluster.master_down_for(downtime)

    wait_for_command_state(command_id, "TERMINATED", 30)
    assert "success" in get_command_info(command_id)["exitStatus"]
