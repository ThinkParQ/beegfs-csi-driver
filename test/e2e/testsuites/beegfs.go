/*
Copyright 2019 The Kubernetes Authors.

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

// The general structure of this file is inspired by
// https://github.com/kubernetes/kubernetes/blob/v1.19.0/test/e2e/storage/testsuites/multivolume.go.

package testsuites

// The tests in this suite are in addition to the ones provided by the Kubernetes community, but based on the community
// developed framework. See k8s.io/kubernetes/test/e2e/storage/testsuites for example suites. Where possible, we have
// used framework functionality (and even entire tests, albeit with alternative setup or teardown).

// If/when test setup/cleanup for some subset of these tests becomes significantly different from another subset of
// these tests, the two subsets should be broken out into separate suites within the testsuites directory.

import (
	"fmt"
	"path"

	"github.com/netapp/beegfs-csi-driver/test/e2e/driver"
	"github.com/netapp/beegfs-csi-driver/test/e2e/utils"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/errors"
	e2eframework "k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	e2eskipper "k8s.io/kubernetes/test/e2e/framework/skipper"
	e2evolume "k8s.io/kubernetes/test/e2e/framework/volume"
	storageframework "k8s.io/kubernetes/test/e2e/storage/framework"
	storagesuites "k8s.io/kubernetes/test/e2e/storage/testsuites"
)

// Verify interface is properly implemented at compile time.
var _ storageframework.TestSuite = &beegfsTestSuite{}

type beegfsTestSuite struct {
	tsInfo storageframework.TestSuiteInfo
}

// beegfsTestSuite implements the storageframework.TestSuite interface.
func (b *beegfsTestSuite) GetTestSuiteInfo() storageframework.TestSuiteInfo {
	return b.tsInfo
}

// beegfsTestSuite implements the storageframework.TestSuite interface.
func (b *beegfsTestSuite) SkipUnsupportedTests(driver storageframework.TestDriver, pattern storageframework.TestPattern) {
	// Intentionally empty.
}

// InitBeegfsTestSuite returns a beegfsTestSuite that implements storageframework.TestSuite interface
func InitBeegfsTestSuite() storageframework.TestSuite {
	return &beegfsTestSuite{
		tsInfo: storageframework.TestSuiteInfo{
			Name: "beegfs-suite",
			TestPatterns: []storageframework.TestPattern{
				storageframework.DefaultFsDynamicPV,
				storageframework.DefaultFsPreprovisionedPV,
			},
			SupportedSizeRange: e2evolume.SizeRange{
				Min: "1Mi",
			},
		},
	}
}

// beegfsTestSuite implements the storageframework.TestSuiteInfo interface.
func (b *beegfsTestSuite) DefineTests(tDriver storageframework.TestDriver, pattern storageframework.TestPattern) {
	f := e2eframework.NewDefaultFramework("beegfs")

	// We can use a single BeegfsDriver for multiple tests because of the way Ginkgo performs parallelization
	// See test/e2e/README.md for details
	var (
		d         *driver.BeegfsDriver
		resources []*storageframework.VolumeResource
	)

	init := func() {
		var ok bool
		d, ok = tDriver.(*driver.BeegfsDriver) // These tests use BeegfsDriver specific methods.
		if !ok {
			e2eskipper.Skipf("This test only works with a BeegfsDriver -- skipping")
		}
		d.SetFSIndex(0)
		resources = make([]*storageframework.VolumeResource, 0)
	}

	// This block is heavily adapted from the cleanup in the k8s.io/kubernetes/test/e2e/storage/testsuites
	// multivolumeTestSuite DefineTests function
	// (https://github.com/kubernetes/kubernetes/blob/v1.19.0/test/e2e/storage/testsuites/multivolume.go#L113-L117).
	cleanup := func() {
		var errs []error
		for _, resource := range resources {
			errs = append(errs, resource.CleanupResource())
		}
		e2eframework.ExpectNoError(errors.NewAggregate(errs), "while cleaning up resources")
	}

	// This test runs for DynamicPV and PreprovisionedPV patterns.
	ginkgo.It("should access to two volumes from different file systems with the same volume mode and retain "+
		"data across pod recreation on the same node", func() {
		init()
		defer cleanup()
		cfg, _ := d.PrepareTest(f)
		testVolumeSizeRange := b.GetTestSuiteInfo().SupportedSizeRange

		// We can't check/skip until d is instantiated in init().
		if d.GetNumFS() < 2 {
			e2eskipper.Skipf("This test requires at least two distinct file systems")
		}

		// Create storage classes that reference all available file systems.
		var pvcs []*corev1.PersistentVolumeClaim
		for i := 0; i < d.GetNumFS(); i++ {
			d.SetFSIndex(i)
			resource := storageframework.CreateVolumeResource(d, cfg, pattern, testVolumeSizeRange)
			resources = append(resources, resource) // Allow for cleanup.
			pvcs = append(pvcs, resource.Pvc)
		}

		// There is already a Kubernetes end-to-end test that tests this behavior (and more).
		storagesuites.TestAccessMultipleVolumesAcrossPodRecreation(f, f.ClientSet, f.Namespace.Name, cfg.ClientNodeSelection, pvcs, true)
	})

	ginkgo.It("should correctly interpret a storage class stripe pattern", func() {
		if pattern.VolType != storageframework.DynamicPV {
			e2eskipper.Skipf("This test only works with dynamic volumes -- skipping")
		}

		// Don't do expensive test setup until we know we'll run the test.
		init()
		defer cleanup()
		cfg, _ := d.PrepareTest(f)
		testVolumeSizeRange := b.GetTestSuiteInfo().SupportedSizeRange

		// Create an FSExec with a storage resource including a StorageClass with non-standard striping params.
		d.SetStorageClassParams(map[string]string{
			"stripePattern/storagePoolID": "2",
			"stripePattern/chunkSize":     "1m",
			"stripePattern/numTargets":    "2",
		})
		defer d.UnsetStorageClassParams()
		fsExec := utils.NewFSExec(cfg, d, testVolumeSizeRange)
		defer func() {
			e2eframework.ExpectNoError(fsExec.Cleanup())
		}()

		// Execute beegfs-ctl getentryinfo command.
		volDirPathBeegfsRoot := path.Join(fsExec.Resource.Sc.Parameters["volDirBasePath"], fsExec.Resource.Pv.Name)
		result, err := fsExec.IssueCommandWithBeegfsPaths("beegfs-ctl --mount=%s --getentryinfo %s",
			"", volDirPathBeegfsRoot)

		e2eframework.ExpectNoError(err)
		gomega.Expect(string(result)).To(gomega.ContainSubstring("Storage Pool: 2"))
		gomega.Expect(string(result)).To(gomega.ContainSubstring("Chunksize: 1M"))
		gomega.Expect(string(result)).To(gomega.ContainSubstring("Number of storage targets: desired: 2"))
	})

	ginkgo.It("should use RDMA to connect", func() {
		init()
		defer cleanup()
		cfg, _ := d.PrepareTest(f)
		testVolumeSizeRange := b.GetTestSuiteInfo().SupportedSizeRange

		// We can't check/skip until d is instantiated in init().
		if !d.SetFSIndexForRDMA() {
			e2eskipper.Skipf("No available RDMA capable file systems -- skipping")
		}

		// Create an FSExec with a storage resource.
		fsExec := utils.NewFSExec(cfg, d, testVolumeSizeRange)
		defer func() {
			e2eframework.ExpectNoError(fsExec.Cleanup())
		}()

		// Query /proc for connection information associated with this storage resource.
		cmd := fmt.Sprintf("cat $(dirname $(grep -l %s /proc/fs/beegfs/*/config))/storage_nodes", fsExec.Resource.Pv.Name)
		result, err := fsExec.IssueCommandWithResult(cmd)
		e2eframework.ExpectNoError(err)

		// Output looks like:
		// localhost [ID: 1]
		//    Connections: TCP: 4 (10.193.113.4:8003);
		gomega.Expect(result).To(gomega.ContainSubstring("RDMA"))
	})

	ginkgo.It("should not be able to write to the /host file system", func() {
		if pattern.VolType != storageframework.DynamicPV {
			e2eskipper.Skipf("This test is covered with the dynamic volume pattern -- skipping")
		}

		init()
		defer cleanup()

		// Get the controller Pod, which could be running in any namespace.
		pods, err := e2epod.WaitForPodsWithLabelRunningReady(f.ClientSet, "",
			labels.SelectorFromSet(map[string]string{"app": "csi-beegfs-controller"}), 1, e2eframework.PodStartTimeout)
		e2eframework.ExpectNoError(err, "There should be exactly one controller pod")
		controllerPod := pods.Items[0]

		// Get a node Pod, which could be running in any namespace.
		pods, err = e2epod.WaitForPodsWithLabel(f.ClientSet, "",
			labels.SelectorFromSet(map[string]string{"app": "csi-beegfs-node"}))
		e2eframework.ExpectNoError(err, "There should be at least one node pod")
		nodePod := pods.Items[0]

		for _, pod := range []corev1.Pod{controllerPod, nodePod} {
			execOptions := e2eframework.ExecOptions{
				Command:       []string{"touch", "/host/test-file"},
				PodName:       pod.Name,
				Namespace:     pod.Namespace,
				ContainerName: "beegfs",
				CaptureStdout: true, // stdOut must be captured to avoid a timeout.
				CaptureStderr: true,
			}
			// There are other framework functions that seem more appropriate (e.g. LookForStringInPodExecToContainer),
			// but they do not work because they ignore stdErr, which we want to read.
			_, stdErr, err := f.ExecWithOptions(execOptions)
			e2eframework.ExpectError(err) // The touch should not be successful.
			gomega.Expect(stdErr).To(gomega.ContainSubstring("Read-only file system"))
		}
	})

	ginkgo.It("should correctly set permissions specified as storage class parameters", func() {
		if pattern.VolType != storageframework.DynamicPV {
			e2eskipper.Skipf("This test only works with dynamic volumes -- skipping")
		}

		// Don't do expensive test setup until we know we'll run the test.
		init()
		defer cleanup()
		cfg, _ := d.PrepareTest(f)
		testVolumeSizeRange := b.GetTestSuiteInfo().SupportedSizeRange

		// Create volume resource including a StorageClass with permissions params.
		const (
			uid          = "1000"
			gid          = "2000"
			mode         = "0755"
			expectedMode = "drwxr-xr-x" // `ls` representation of the expected octal mode for a directory
		)
		d.SetStorageClassParams(map[string]string{
			"permissions/uid":  uid,
			"permissions/gid":  gid,
			"permissions/mode": mode,
		})
		defer d.UnsetStorageClassParams()
		resource := storageframework.CreateVolumeResource(d, cfg, pattern, testVolumeSizeRange)
		resources = append(resources, resource) // Allow for cleanup.

		// Create a pod to consume the storage resource.
		podConfig := e2epod.Config{
			NS:      cfg.Framework.Namespace.Name,
			PVCs:    []*corev1.PersistentVolumeClaim{resource.Pvc},
			ImageID: e2epod.GetDefaultTestImageID(),
		}
		pod, err := e2epod.CreateSecPodWithNodeSelection(f.ClientSet, &podConfig, e2eframework.PodStartTimeout)
		defer func() {
			// ExpectNoError() must be wrapped in a func() or it will be evaluated (and the pod will be deleted) now.
			e2eframework.ExpectNoError(e2epod.DeletePodWithWait(f.ClientSet, pod))
		}()
		e2eframework.ExpectNoError(err)

		// Verify permissions.
		utils.VerifyDirectoryModeUidGidInPod(f, "/mnt/volume1", expectedMode, uid, gid, pod)
	})

	ginkgo.It("should correctly set default permissions", func() {
		if pattern.VolType != storageframework.DynamicPV {
			e2eskipper.Skipf("This test only works with dynamic volumes -- skipping")
		}

		// Don't do expensive test setup until we know we'll run the test.
		init()
		defer cleanup()
		cfg, _ := d.PrepareTest(f)
		testVolumeSizeRange := b.GetTestSuiteInfo().SupportedSizeRange

		// Create volume resource including a StorageClass without permissions params.
		const (
			expectedMode = "drwxrwxrwx" // `ls` representation of the expected octal mode for a directory
			expectedUid  = "root"
			expectedGid  = "root"
		)
		resource := storageframework.CreateVolumeResource(d, cfg, pattern, testVolumeSizeRange)
		resources = append(resources, resource) // Allow for cleanup.

		// Create a pod to consume the storage resource.
		podConfig := e2epod.Config{
			NS:      cfg.Framework.Namespace.Name,
			PVCs:    []*corev1.PersistentVolumeClaim{resource.Pvc},
			ImageID: e2epod.GetDefaultTestImageID(),
		}
		pod, err := e2epod.CreateSecPodWithNodeSelection(f.ClientSet, &podConfig, e2eframework.PodStartTimeout)
		defer func() {
			// ExpectNoError() must be wrapped in a func() or it will be evaluated (and the pod will be deleted) now.
			e2eframework.ExpectNoError(e2epod.DeletePodWithWait(f.ClientSet, pod))
		}()
		e2eframework.ExpectNoError(err)

		// Verify permissions.
		utils.VerifyDirectoryModeUidGidInPod(f, "/mnt/volume1", expectedMode, expectedUid, expectedGid, pod)
	})

	ginkgo.It("should delete only the anticipated directory", func() {
		// This test creates the following directory structure on the BeeGFS file system:
		// /
		// |-- e2e-test
		//     |-- dynamic (already exists from other tests)
		//     |-- static (already exists from other tests)
		//     |-- delete
		//         |-- beegfs-xxxx (potential unique volDirBasePath from another test)
		//         |-- beegfs-yyyy (unique volDirBasePath created by this test)
		//             |-- before.tar (archive created during this test)
		//             |-- pvc-######## (PVC created by this test)
		//             |-- pvc-######## (PVC created by this test)
		//             |-- pvc-######## (PVC created by this test)
		//
		// The test creates a tar archive of the beegfs-yyyy directory. It then creates another PVC with volDirBasePath
		// /e2e-test/delete/beegfs-yyyy and immediately deletes it. Finally, it confirms that the beegfs-yyyy directory
		// structure matches the original archive.
		//
		// This test confirms:
		// * DeleteVolume results in the removal of the expected directory.
		// * DeleteVolume does not result in the modification or removal of other directories within volDirBasePath.
		// * DeleteVolume does not result in the deletion of volDirBasePath or its parents.
		//
		// This test does not confirm that some arbitrary file or directory from elsewhere in the file system is not
		// deleted. That test would require a guarantee that nothing else could access the file system and would not be
		// significantly more useful (as we cannot anticipate what pattern or directory structure might trigger an
		// error and thus can't set up an appropriate test environment to catch that error).

		if pattern.VolType != storageframework.DynamicPV {
			e2eskipper.Skipf("This test only works with dynamic volumes -- skipping")
		}

		// Don't do expensive test setup until we know we'll run the test.
		init()
		defer cleanup()
		cfg, _ := d.PrepareTest(f)
		testVolumeSizeRange := b.GetTestSuiteInfo().SupportedSizeRange

		// Create an FSExec using a "standard" volDirBasePathBeegfsRoot so we can use it to clean up later. If we used
		// the uniquely named volDirBasePathBeegfsRoot we will test with, our FSExec would include a PVC in that
		// directory and couldn't be used to delete it.
		fsExec := utils.NewFSExec(cfg, d, testVolumeSizeRange)
		defer func() {
			e2eframework.ExpectNoError(fsExec.Cleanup())
		}()

		// Use a uniquely named volDirBasePathBeegfsRoot for isolation.
		volDirBasePathBeegfsRoot := path.Join("e2e-test", "delete", f.UniqueName)
		tarPathBeegfsRoot := path.Join(volDirBasePathBeegfsRoot, "before.tar")
		// The volDirBasePath storage class parameter is the volDirBasePathBeegfsRoot pkg/beegfs parameter.
		d.SetStorageClassParams(map[string]string{"volDirBasePath": volDirBasePathBeegfsRoot})

		// Prepare to delete the uniquely named volDirBasePath at the end of the test.
		defer func() {
			_, err := fsExec.IssueCommandWithBeegfsPaths("rm -rf %s", volDirBasePathBeegfsRoot)
			e2eframework.ExpectNoError(err)
		}()

		// Create three volumes using the unique volDirBasePathBeegfsRoot. These will provide provide the "before"
		// for our test.
		for i := 0; i < 3; i++ {
			resource := storageframework.CreateVolumeResource(d, cfg, pattern, testVolumeSizeRange)
			resources = append(resources, resource) // Allow for cleanup.
		}

		// Create an archive of the volDirBasePath as it currently exists.
		_, err := fsExec.IssueCommandWithBeegfsPaths("tar -cf %s %s", tarPathBeegfsRoot, volDirBasePathBeegfsRoot)
		e2eframework.ExpectNoError(err)

		// Create and then delete a new PVC with this same unique volDirBasePathBeegfsRoot.
		resource := storageframework.CreateVolumeResource(d, cfg, pattern, testVolumeSizeRange)
		e2eframework.ExpectNoError(resource.CleanupResource())

		// Verify that the current state of volDirBasePath matches our original archive.
		_, err = fsExec.IssueCommandWithBeegfsPaths("tar --diff -f %s", tarPathBeegfsRoot)
		e2eframework.ExpectNoError(err)
	})
}
