/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Modifications Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package utils

import (
	"fmt"
	"os"
	"path"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	e2eframework "k8s.io/kubernetes/test/e2e/framework"
	e2enode "k8s.io/kubernetes/test/e2e/framework/node"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	e2essh "k8s.io/kubernetes/test/e2e/framework/ssh"
	e2evolume "k8s.io/kubernetes/test/e2e/framework/volume"
)

// VerifyDirectoryModeUidGidInPod verifies expected mode, UID, and GID of the target directory
// This implementation is similar to [`VerifyFilePathGidInPod`](https://github.com/kubernetes/kubernetes/blob/v1.21.0/test/e2e/storage/utils/utils.go#L709).
func VerifyDirectoryModeUidGidInPod(f *e2eframework.Framework, directory, expectedMode, expectedUid, expectedGid string, pod *corev1.Pod) {
	cmd := fmt.Sprintf("ls -ld %s", directory)
	stdout, stderr, err := e2evolume.PodExec(f, pod, cmd)
	e2eframework.ExpectNoError(err)
	e2eframework.Logf("pod %s/%s exec for cmd %s, stdout: %s, stderr: %s", pod.Namespace, pod.Name, cmd, stdout, stderr)
	ll := strings.Fields(stdout)
	e2eframework.Logf("stdout split: %v, expected mode: %v, expected uid: %v, expected gid: %v ", ll, expectedMode, expectedUid, expectedGid)
	e2eframework.ExpectEqual(ll[0], expectedMode)
	e2eframework.ExpectEqual(ll[2], expectedUid)
	e2eframework.ExpectEqual(ll[3], expectedGid)
}

// VerifyNoOrphanedMounts uses SSH to access all cluster nodes and verify that none of them have a BeeGFS file system
// mounted as a PersistentVolume. VerifyNoOrphanedMounts could be used within a single test case, but mounts are orphaned
// intermittently and it would be unlikely to catch an orphan mount without including an extremely long stress test
// within the case. It is currently preferred to use VerifyNoOrphanedMounts before and after an entire suite of tests
// runs to ensure none of the tests within the suite causes a mount to be orphaned.
func VerifyNoOrphanedMounts(cs *kubernetes.Clientset) {
	// The external infrastructure taints nodes that cannot participate in the tests.
	nodes, err := e2enode.GetReadySchedulableNodes(cs)
	e2eframework.ExpectNoError(err)
	if len(nodes.Items) < 2 {
		e2eframework.Failf("expected more than %d ready nodes", len(nodes.Items))
	}
	var nodeAddresses []string
	for _, node := range nodes.Items {
		address, err := e2enode.GetInternalIP(&node)
		e2eframework.ExpectNoError(err)
		nodeAddresses = append(nodeAddresses, address)
	}
	for _, nodeAddress := range nodeAddresses {
		result, err := e2essh.SSH("mount | grep -e beegfs_nodev | grep pvc", nodeAddress, e2eframework.TestContext.Provider)
		e2eframework.ExpectNoError(err)
		e2eframework.ExpectEmpty(result.Stdout, "node with address %s has orphaned mounts", nodeAddress)
	}
}

// ArchiveServiceLogs collects the logs on the node and controller service pods and writes them in the specified report
// path. Logs will be collected from beegfs and csi-provisioner containers. This should typically be called after the
// test suite completes.
func ArchiveServiceLogs(cs *kubernetes.Clientset, reportPath string) {
	// Get controller and node pod information
	nodePods, err := e2epod.GetPods(cs, "", map[string]string{"app": "csi-beegfs-node"})
	if err != nil {
		e2eframework.ExpectNoError(err, "failed to get node pods")
	}
	controllerPods, err := e2epod.GetPods(cs, "", map[string]string{"app": "csi-beegfs-controller"})
	if err != nil {
		e2eframework.ExpectNoError(err, "failed to get controller pod")
	}

	// Verify that we have the expected number of node and controller pods
	e2eframework.ExpectNotEqual(len(nodePods), 0, "expected to find at least one node pod")
	e2eframework.ExpectEqual(len(controllerPods), 1, "expected to find exactly one controller pod")

	// Dump logs of each node pod's beegfs container
	for _, nodePod := range nodePods {
		logs, err := e2epod.GetPodLogs(cs, nodePod.Namespace, nodePod.Name, "beegfs")
		if err != nil {
			e2eframework.ExpectNoError(err, "failed to get beegfs logs for node container beegfs")
		}

		err = os.WriteFile(path.Join(reportPath, fmt.Sprintf("container-logs-%s-beegfs.log", nodePod.Name)), []byte(logs), 0666)
		if err != nil {
			e2eframework.ExpectNoError(err, "failed to write to file")
		}
	}

	// Dump logs of controller pod's beegfs and csi-provisioner containers
	for _, containerName := range []string{"beegfs", "csi-provisioner"} {
		logs, err := e2epod.GetPodLogs(cs, controllerPods[0].Namespace, controllerPods[0].Name, containerName)
		if err != nil {
			e2eframework.ExpectNoError(err, fmt.Sprintf("failed to get logs for controller container %s", containerName))
		}

		err = os.WriteFile(path.Join(reportPath, fmt.Sprintf("container-logs-%s-%s.log", controllerPods[0].Name, containerName)), []byte(logs), 0666)
		if err != nil {
			e2eframework.ExpectNoError(err, "failed to write to file")
		}
	}
}
