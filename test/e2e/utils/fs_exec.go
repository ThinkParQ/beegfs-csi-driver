/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package utils

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path"
	"strings"

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
	cfg      *storageframework.PerTestConfig
	hostExec storageutils.HostExec
	pod      *corev1.Pod
	Resource *storageframework.VolumeResource // Export for easy inspection.
}

// FSMountData is a type to represent the JSON object returned by findmnt -J. Each object represents one of
// the objects that exist within the filesystems list.
// Example: {"filesystems": [FSMountData1, FSMountData2]}
type FSMountData struct {
	Target  string
	Source  string
	FsType  string
	Options string
}

// FindmntData represents the top level JSON object returned by 'findmnt -J'.
type FindmntData struct {
	Filesystems []FSMountData
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

// getAllBeegfsMounts will return a slice of type FSMountData including data for all mounted
// BeeGFS filesystems found on the FSExec host when the command is run. Potential errors might
// arise from running the findmnt command or from decoding the JSON output of the command.
// This function will return an unfiltered list which will include any bind mounts if they exist
// on the system.
func (e *FSExec) getAllBeegfsMounts() (mounts []FSMountData, err error) {
	// This command will find all beegfs filesystem mounts and return a JSON string
	command := "findmnt -J -t beegfs"
	findOutput, err := e.IssueCommandWithResult(command)
	if err != nil {
		return mounts, err
	}
	var mountData FindmntData
	if jsonErr := json.Unmarshal([]byte(findOutput), &mountData); jsonErr != nil {
		return mounts, jsonErr
	}
	return mountData.Filesystems, nil

}

// getBeegfsMountsExcludingBindMounts will return a slice of type FSMountData. The slice will be filtered
// to exclude any bind mounts from the system so that only regular filesystem mounts will be returned.
func (e *FSExec) getBeegfsMountsExcludingBindMounts() (mounts []FSMountData, err error) {
	mounts, err = e.getAllBeegfsMounts()
	if err != nil {
		return mounts, err
	}
	var filteredMounts []FSMountData
	for _, fs := range mounts {
		// Bind mounts include '[/dir/path]' in the source value in addition to 'beegfs_nodev'
		if fs.Source == "beegfs_nodev" {
			filteredMounts = append(filteredMounts, fs)
		}
	}
	return filteredMounts, nil
}

// getBeegfsMountByValue will return the path of the BeeGFS mount point that is associated
// with the provided value. An error is returned if there is a problem getting the mount
// information or if no matching filesystem is found.
func (e *FSExec) getBeegfsMountByValue(value string) (mount string, err error) {
	mounts, err := e.getBeegfsMountsExcludingBindMounts()
	if err != nil {
		return mount, err
	}
	for _, fs := range mounts {
		if strings.Contains(fs.Target, value) {
			mount = fs.Target
			// Ignore any possible duplicate entries which we don't expect
			break
		}
	}
	if mount != "" {
		return mount, nil
	}
	return mount, fmt.Errorf("no beegfs filesystem found matching value %s", value)
}

// GetVolumeSHA256Checksum will return the SHA256 checksum of the volumeHandle for the volume
// associated with this FSExec. This assumes that there is a single volume associated with the
// FSExec. Starting in Kubernetes 1.24 this checksum is now used in CSI staging paths.
func (e *FSExec) GetVolumeSHA256Checksum() (checksum string) {
	sumData := sha256.Sum256([]byte(e.Resource.Pv.Spec.CSI.VolumeHandle))
	return fmt.Sprintf("%x", sumData)
}

// GetVolumeHostMountPath will return the path of the BeeGFS filesystem mount related to
// the volume associated with this FSExec. This assumes that there is a single volume
// associated with the FSExec.
func (e *FSExec) GetVolumeHostMountPath() (string, error) {
	volumeHash := e.GetVolumeSHA256Checksum()
	// First let's check for the mount path using the new sha256 based pathing method
	hashPath, hashErr := e.getBeegfsMountByValue(volumeHash)
	if hashErr != nil {
		err := fmt.Errorf("get beegfs mount by hash failed: %w", hashErr)
		// If the checksum method failed, try the older method using the pv name
		namePath, nameErr := e.getBeegfsMountByValue(e.Resource.Pv.Name)
		if nameErr != nil {
			// err = fmt.Errorf("get beegfs mount by name failed: %w", nameErr)
			return "", fmt.Errorf("get beegfs mount by name failed: %v %v", nameErr, err)
		}
		return namePath, nil
	}
	return hashPath, nil
}

// IssueCommandWithBeegfsPaths takes a format string (like the ones passed to fmt.Printf, fmt.Sprintf, etc.) and any
// number of BeeGFS relative paths. It converts the relative paths to absolute paths on the host that will execute a
// command.
func (e *FSExec) IssueCommandWithBeegfsPaths(cmdFmtString string, beegfsPaths ...string) (string, error) {
	mountPath, err := e.GetVolumeHostMountPath()
	if err != nil {
		return "", err
	}
	var absPaths []interface{}
	for _, relPath := range beegfsPaths {
		absPaths = append(absPaths, path.Join(mountPath, relPath))
	}
	return e.IssueCommandWithResult(fmt.Sprintf(cmdFmtString, absPaths...))
}

// IssueCommandWithResult works exactly like hostExec.IssueCommandWithResult, except that it identifies the
// correct node (the one with our BeeGFS file system mounted) on which to execute the passed command automatically.
func (e *FSExec) IssueCommandWithResult(cmd string) (string, error) {
	// Some automatic worker node prep workflows put BeeGFS utilities in the plugin-owned
	// /var/lib/kubelet/plugins/beegfs.csi.netapp.com/client/sbin directory to avoid base OS "contamination".
	cmd = "PATH=$PATH:/var/lib/kubelet/plugins/beegfs.csi.netapp.com/client/sbin " + cmd
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
