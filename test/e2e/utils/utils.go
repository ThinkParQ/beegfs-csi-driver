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
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"
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
func VerifyNoOrphanedMounts(cs clientset.Interface) {
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
// test suite completes. It may also make sense to call it if a test will intentionally redeploy Pods (as this action
// results in a reset of container logs).
func ArchiveServiceLogs(cs clientset.Interface, reportPath string) {
	// Get controller and node pod information
	controllerPod := GetRunningControllerPod(cs)
	nodePods := GetRunningNodePods(cs)

	// Dump logs of each node pod's beegfs container
	for _, nodePod := range nodePods {
		logs, err := e2epod.GetPodLogs(cs, nodePod.Namespace, nodePod.Name, "beegfs")
		if err != nil {
			e2eframework.ExpectNoError(err, "failed to get beegfs logs for node container beegfs")
		}

		filePath := path.Join(reportPath, fmt.Sprintf("container-logs-%s-beegfs.log", nodePod.Name))
		AppendBytesToFile(filePath, []byte(logs))
	}

	// Dump logs of controller pod's beegfs and csi-provisioner containers
	for _, containerName := range []string{"beegfs", "csi-provisioner"} {
		logs, err := e2epod.GetPodLogs(cs, controllerPod.Namespace, controllerPod.Name, containerName)
		if err != nil {
			e2eframework.ExpectNoError(err, fmt.Sprintf("failed to get logs for controller container %s", containerName))
		}

		filePath := path.Join(reportPath, fmt.Sprintf("container-logs-%s-%s.log", controllerPod.Name, containerName))
		AppendBytesToFile(filePath, []byte(logs))
	}
}

// AppendBytesToFile appends to a file if it already exists or creates it if it does not.
func AppendBytesToFile(filePath string, logs []byte) {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		e2eframework.ExpectNoError(err, "failed to write to open file %s", filePath)
	}
	if _, err = f.Write(logs); err != nil {
		f.Close()
		e2eframework.ExpectNoError(err, "failed to write logs to file %s", filePath)
	}
	if err := f.Close(); err != nil {
		e2eframework.ExpectNoError(err, "failed to close file %s", filePath)
	}
}

// GetRunningControllerPod waits PodStartTimeout for exactly one controller service Pod to be running (in any namespace)
// and returns it. A consuming test fails if this is not possible.
func GetRunningControllerPod(cs clientset.Interface) corev1.Pod {
	controllerPods, err := e2epod.WaitForPodsWithLabelRunningReady(cs, "",
		labels.SelectorFromSet(map[string]string{"app": "csi-beegfs-controller"}), 1, e2eframework.PodStartTimeout)
	e2eframework.ExpectNoError(err, "expected to find exactly one controller pod")
	return controllerPods.Items[0]
}

// GetRunningNodePods waits PodStartTimeout for at least one node service Pod to be running (in any namespace) and
// returns all node service pods it finds. A consuming test fails if it cannot find at least one running Pod.
func GetRunningNodePods(cs clientset.Interface) []corev1.Pod {
	selector := labels.SelectorFromSet(map[string]string{"app": "csi-beegfs-node"})
	// Get node service Pods that should be running (we don't know how many up front).
	nodePods, err := e2epod.WaitForPodsWithLabelScheduled(cs, "", selector)
	if err != nil {
		e2eframework.ExpectNoError(err, "failed to get any node pods")
	}
	// Get node service Pods one they are all running.
	nodePods, err = e2epod.WaitForPodsWithLabelRunningReady(cs, "", selector, len(nodePods.Items), e2eframework.PodStartTimeout)
	if err != nil {
		e2eframework.ExpectNoError(err, "node pods took too long to run")
	}
	return nodePods.Items
}
