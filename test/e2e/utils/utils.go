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
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"
	e2eframework "k8s.io/kubernetes/test/e2e/framework"
	e2enode "k8s.io/kubernetes/test/e2e/framework/node"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	e2epv "k8s.io/kubernetes/test/e2e/framework/pv"
	e2essh "k8s.io/kubernetes/test/e2e/framework/ssh"
	e2evolume "k8s.io/kubernetes/test/e2e/framework/volume"
)

// VerifyDirectoryModeUidGidInPod verifies expected mode, UID, and GID of the target directory
// This implementation is similar to [`VerifyFilePathGidInPod`](https://github.com/kubernetes/kubernetes/blob/v1.21.0/test/e2e/storage/utils/utils.go#L709).
func VerifyDirectoryModeUidGidInPod(f *e2eframework.Framework, directory, expectedMode, expectedUID, expectedGID string, pod *corev1.Pod) {
	cmd := fmt.Sprintf("ls -ld %s", directory)
	stdout, stderr, err := e2evolume.PodExec(f, pod, cmd)
	e2eframework.ExpectNoError(err)
	e2eframework.Logf("pod %s/%s exec for cmd %s, stdout: %s, stderr: %s", pod.Namespace, pod.Name, cmd, stdout, stderr)
	ll := strings.Fields(stdout)
	e2eframework.Logf("stdout split: %v, expected mode: %v, expected uid: %v, expected gid: %v ", ll, expectedMode, expectedUID, expectedGID)
	e2eframework.ExpectEqual(ll[0], expectedMode)
	e2eframework.ExpectEqual(ll[2], expectedUID)
	e2eframework.ExpectEqual(ll[3], expectedGID)
}

// VerifyNoOrphanedMounts uses SSH to access all cluster nodes and verify that none of them have a BeeGFS file system
// mounted by the CSI driver. VerifyNoOrphanedMounts could be used within a single test case, but mounts are orphaned
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
		cmd := "findmnt -l -n -t beegfs | grep '/var/lib/kubelet/'"
		result, err := e2essh.SSH(cmd, nodeAddress, e2eframework.TestContext.Provider)
		e2eframework.ExpectNoError(err)
		e2eframework.ExpectEmpty(result.Stdout, "node with address %s has orphaned mounts", nodeAddress)
	}
}

// ArchiveServiceLogs collects the logs on the node and controller service pods and writes them in the specified report
// path. Logs will be collected from beegfs and csi-provisioner containers. This should typically be called after the
// test suite completes. It may also make sense to call it if a test will intentionally redeploy pods (as this action
// results in a reset of container logs). ArchiveServiceLogs returns the first error it generates instead of failing in
// case the caller wants to proceed anyway.
func ArchiveServiceLogs(cs clientset.Interface, reportPath string) error {
	var returnError error = nil

	// Dump logs of each node pod's beegfs container.
	nodePods, err := GetRunningNodePods(cs)
	if err != nil {
		e2eframework.Logf("Failed to get node pods due to error: %w", err)
		returnError = err
	} else { // Only proceed if we have node pods to get get logs from.
		for _, nodePod := range nodePods {
			logs, err := e2epod.GetPodLogs(cs, nodePod.Namespace, nodePod.Name, "beegfs")
			if err != nil {
				e2eframework.Logf("Failed to get logs for node container beegfs due to error: %w", err)
				if returnError == nil {
					returnError = err
				}
			}
			filePath := path.Join(reportPath, fmt.Sprintf("container-logs-%s-beegfs.log", nodePod.Name))
			AppendBytesToFile(filePath, []byte(logs))
		}
	}

	// Dump logs of controller pod's beegfs and csi-provisioner containers
	controllerPod, err := GetRunningControllerPod(cs)
	if err != nil {
		e2eframework.Logf("Failed to get controller pod due to error: %w", err)
		returnError = err
	} else { // Only proceed if we have a controller pod to get get logs from.
		for _, containerName := range []string{"beegfs", "csi-provisioner"} {
			logs, err := e2epod.GetPodLogs(cs, controllerPod.Namespace, controllerPod.Name, containerName)
			if err != nil {
				e2eframework.Logf("Failed to get logs for controller container %s due to error: %w", containerName, err)
				if returnError == nil {
					returnError = err
				}
			}
			filePath := path.Join(reportPath, fmt.Sprintf("container-logs-%s-%s.log", controllerPod.Name, containerName))
			AppendBytesToFile(filePath, []byte(logs))
		}
	}

	return returnError
}

// AppendBytesToFile appends to a file if it already exists or creates it if it does not. AppendBytesToFile returns an
// error instead of failing if the write doesn't work in case the caller wants to proceed anyway.
func AppendBytesToFile(filePath string, bytes []byte) error {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer func() {
		if err = f.Close(); err != nil {
			// Don't return an error on close failure as there isn't anything for the caller to do.
			e2eframework.Logf("Failed to close file %s", filePath)
		}
	}()
	if _, err = f.Write(bytes); err != nil {
		return err
	}
	return nil
}

// GetRunningControllerPod waits PodStartTimeout for exactly one controller service Pod to be running (in any namespace)
// and returns it or an error.
func GetRunningControllerPod(cs clientset.Interface) (corev1.Pod, error) {
	controllerPods, err := e2epod.WaitForPodsWithLabelRunningReady(cs, "",
		labels.SelectorFromSet(map[string]string{"app": "csi-beegfs-controller"}), 1, e2eframework.PodStartTimeout)
	if err != nil {
		return corev1.Pod{}, err
	}
	return controllerPods.Items[0], nil
}

// GetRunningControllerPodOrFail waits PodStartTimeout for exactly one controller service Pod to be running (in any
// namespace) and returns it or fails.
func GetRunningControllerPodOrFail(cs clientset.Interface) corev1.Pod {
	controllerPod, err := GetRunningControllerPod(cs)
	if err != nil {
		e2eframework.ExpectNoError(err, "expected to find exactly one controller pod")
	}
	return controllerPod
}

// GetRunningNodePods waits PodStartTimeout for at least one node service Pod to be running (in any namespace) and
// returns all node service pods it finds or an error.
func GetRunningNodePods(cs clientset.Interface) ([]corev1.Pod, error) {
	selector := labels.SelectorFromSet(map[string]string{"app": "csi-beegfs-node"})
	// Get node service Pods that should be running (we don't know how many up front).
	nodePods, err := e2epod.WaitForPodsWithLabelScheduled(cs, "", selector)
	if err != nil {
		return []corev1.Pod{}, err
	}
	// Get node service Pods one they are all running.
	nodePods, err = e2epod.WaitForPodsWithLabelRunningReady(cs, "", selector, len(nodePods.Items),
		e2eframework.PodStartTimeout)
	if err != nil {
		return []corev1.Pod{}, err
	}
	return nodePods.Items, nil
}

// GetRunningNodePodsOrFail waits PodStartTimeout for at least one node service Pod to be running (in any namespace) and
// returns all node service pods it finds or an error.
func GetRunningNodePodsOrFail(cs clientset.Interface) []corev1.Pod {
	nodePods, err := GetRunningNodePods(cs)
	if err != nil {
		e2eframework.ExpectNoError(err, "expected to find at least one node pod and for all node pods to run")
	}
	return nodePods
}

// GetUnusedPoolID will return a string representing a pool id that is not in
// the list of existing configured pool ids. If no such string can be found then
// an empty string is returned.
func GetUnusedPoolId(configuredPools []string) string {
	poolId := ""
	for id := 5; id < 199; id++ {
		if !ContainsString(configuredPools, fmt.Sprint(id)) {
			return fmt.Sprint(id)
		}
	}
	return poolId
}

// The ContainsString function returns true if the passed in data slice contains
// an element with the value val. If the value is not found in the slice
// then false is returned.
func ContainsString(data []string, val string) bool {
	for _, entry := range data {
		if entry == val {
			return true
		}
	}
	return false
}

// CreatePVCFromStorageClass is based on createPVCPVFromDynamicProvisionSC which is part of
// the kubernetes e2e/storage/framework/volume_resource. The purpose of this function is to
// create a PVC from a given storage class without the built in assertions of the framework's
// CreateVolumeResource function and it's related functions.
// This function does not handle cleanup. Be warned!
func CreatePVCFromStorageClass(
	f *e2eframework.Framework,
	name string,
	claimSize string,
	sc *storagev1.StorageClass,
	volMode corev1.PersistentVolumeMode,
	accessModes []corev1.PersistentVolumeAccessMode,
	claimProvisionTimeout time.Duration,
) (*corev1.PersistentVolumeClaim, error) {
	cs := f.ClientSet
	ns := f.Namespace.Name
	pvcCfg := e2epv.PersistentVolumeClaimConfig{
		NamePrefix:       name,
		ClaimSize:        claimSize,
		StorageClassName: &(sc.Name),
		AccessModes:      accessModes,
		VolumeMode:       &volMode,
	}

	pvc := e2epv.MakePersistentVolumeClaim(pvcCfg, ns)
	var err error
	pvc, err = e2epv.CreatePVC(cs, ns, pvc)
	if err != nil {
		return pvc, err
	}

	isDelayedBinding := false
	if sc.VolumeBindingMode != nil {
		isDelayedBinding = *sc.VolumeBindingMode == storagev1.VolumeBindingWaitForFirstConsumer
	}

	if !isDelayedBinding {
		err = e2epv.WaitForPersistentVolumeClaimPhase(corev1.ClaimBound, cs, pvc.Namespace, pvc.Name, e2eframework.Poll, claimProvisionTimeout)
		return pvc, err
	}

	return pvc, nil
}
