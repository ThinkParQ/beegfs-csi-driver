/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package utils

import (
	"fmt"
	"path"

	"github.com/netapp/beegfs-csi-driver/test/e2e/driver"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	e2eframework "k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	e2evolume "k8s.io/kubernetes/test/e2e/framework/volume"
	storageframework "k8s.io/kubernetes/test/e2e/storage/framework"
	storageutils "k8s.io/kubernetes/test/e2e/storage/utils"
)

// FSExec makes it easy to execute commands on a node that has a BeeGFS file system mounted. FSExec handles creating a
// volume resource, creating a pod to mount it, and tracking the node that has it mounted. This frees the caller to
// worry only about what command to execute.
type FSExec struct {
	cfg       *storageframework.PerTestConfig
	hostExec  storageutils.HostExec
	mountPath string
	pod       *corev1.Pod
	Resource  *storageframework.VolumeResource // Export for easy inspection.
}

// NewFSExec initializes an FSExec.
func NewFSExec(cfg *storageframework.PerTestConfig, driver *driver.BeegfsDriver, sizeRange e2evolume.SizeRange) FSExec {
	thisExec := FSExec{
		cfg:      cfg,
		hostExec: storageutils.NewHostExec(cfg.Framework),
	}
	var err error

	// Create a storageframework.VolumeResource.
	// As is always the case with storageframework.CreateVolumeResource(), there is no way for us to get at any of the
	// components of the created VolumeResource during creation. If an error occurs, an embedded ExpectNoError() will
	// register a failure and launch us back up the stack. Namespaced resources will be cleaned up by AfterEach, but
	// non-namespaced resources will leak.
	thisExec.Resource = storageframework.CreateVolumeResource(driver, cfg, storageframework.DefaultFsDynamicPV,
		sizeRange)

	// Create a Pod to mount the storageframework.VolumeResource.
	podConfig := e2epod.Config{
		NS:      cfg.Framework.Namespace.Name,
		PVCs:    []*corev1.PersistentVolumeClaim{thisExec.Resource.Pvc},
		ImageID: e2epod.GetDefaultTestImageID(),
	}
	thisExec.pod, err = e2epod.CreateSecPodWithNodeSelection(cfg.Framework.ClientSet, &podConfig,
		e2eframework.PodStartTimeout)
	if err != nil {
		// If an error occurs during Pod creation, we can clean up the storageframework.VolumeResource, which has
		// non-namespaced objects, as well as the half-created Pod.
		errs := []error{err}
		errs = append(errs, thisExec.Cleanup())
		e2eframework.ExpectNoError(errors.Flatten(errors.NewAggregate(errs)))
	}

	return thisExec
}

// IssueCommandWithBeegfsPaths takes a format string (like the ones passed to fmt.Printf, fmt.Sprintf, etc.) and any
// number of BeeGFS relative paths. It converts the relative paths to absolute paths on the host that will execute a
// command.
func (e *FSExec) IssueCommandWithBeegfsPaths(cmdFmtString string, beegfsPaths ...string) (string, error) {
	mountPath := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/pv/%s/globalmount/mount", e.Resource.Pv.Name)
	var absPaths []interface{}
	for _, relPath := range beegfsPaths {
		absPaths = append(absPaths, path.Join(mountPath, relPath))
	}
	return e.IssueCommandWithResult(fmt.Sprintf(cmdFmtString, absPaths...))
}

// IssueCommandWithResult works exactly like hostExec.IssueCommandWithResult, except that it identifies the
// correct node (the one with our BeeGFS file system mounted) on which to execute the passed command automatically.
func (e *FSExec) IssueCommandWithResult(cmd string) (string, error) {
	node, err := e.cfg.Framework.ClientSet.CoreV1().Nodes().Get(context.TODO(), e.pod.Spec.NodeName, metav1.GetOptions{})
	e2eframework.ExpectNoError(err)
	return e.hostExec.IssueCommandWithResult(cmd, node)
}

// Cleanup removes the resources used by FSExec from the cluster.
func (e *FSExec) Cleanup() error {
	var errs []error
	errs = append(errs, e2epod.DeletePodWithWait(e.cfg.Framework.ClientSet, e.pod))
	errs = append(errs, e.Resource.CleanupResource())
	e.hostExec.Cleanup()
	// Return an error instead of asserting (e.g. e2eframework.ExpectNoError()) to give the calling code the option to
	// further aggregate cleanup errors.
	return errors.NewAggregate(errs)
}
