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
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	e2eframework "k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	e2eskipper "k8s.io/kubernetes/test/e2e/framework/skipper"
	e2evolume "k8s.io/kubernetes/test/e2e/framework/volume"
	storageframework "k8s.io/kubernetes/test/e2e/storage/framework"
	storagesuites "k8s.io/kubernetes/test/e2e/storage/testsuites"
	storageutils "k8s.io/kubernetes/test/e2e/storage/utils"
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

	var (
		d         *driver.BeegfsDriver
		resources []*storageframework.VolumeResource
		hostExec  storageutils.HostExec
	)

	init := func() {
		var ok bool
		d, ok = tDriver.(*driver.BeegfsDriver) // These tests use BeegfsDriver specific methods.
		if !ok {
			e2eskipper.Skipf("This test only works with a BeegfsDriver")
		}
		d.SetFSIndex(0)
		resources = make([]*storageframework.VolumeResource, 0)
		hostExec = storageutils.NewHostExec(f)
	}

	cleanup := func() {
		var errs []error
		for _, resource := range resources {
			errs = append(errs, resource.CleanupResource())
		}
		e2eframework.ExpectNoError(errors.NewAggregate(errs), "while cleaning up resources")
		hostExec.Cleanup()
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

		// Create storage resource including a StorageClass with non-standard striping params.
		d.SetStorageClassParams(map[string]string{
			"stripePattern/storagePoolID": "2",
			"stripePattern/chunkSize":     "1m",
			"stripePattern/numTargets":    "2",
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

		// Construct necessary beegfs-ctl parameters.
		globalMountPath := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/pv/%s/globalmount/mount", resource.Pv.Name)
		volDirPathBeegfsRoot := path.Join(resource.Sc.Parameters["volDirBasePath"], resource.Pv.Name)

		// Get striping information using beegfs-ctl on the appropriate hose.
		node, err := f.ClientSet.CoreV1().Nodes().Get(context.TODO(), pod.Spec.NodeName, metav1.GetOptions{})
		e2eframework.ExpectNoError(err)
		cmd := fmt.Sprintf("beegfs-ctl --mount=%s --unmounted --getentryinfo %s", globalMountPath, volDirPathBeegfsRoot)
		result, err := hostExec.IssueCommandWithResult(cmd, node)

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

		// Create a single storage resource to be consumed by a Pod.
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

		// Query /proc for connection information associated with this storage resource.
		node, err := f.ClientSet.CoreV1().Nodes().Get(context.TODO(), pod.Spec.NodeName, metav1.GetOptions{})
		e2eframework.ExpectNoError(err)
		cmd := fmt.Sprintf("cat $(dirname $(grep -l %s /proc/fs/beegfs/*/config))/storage_nodes", resource.Pv.Name)
		result, err := hostExec.IssueCommandWithResult(cmd, node)
		e2eframework.ExpectNoError(err)

		// Output looks like:
		// localhost [ID: 1]
		//    Connections: TCP: 4 (10.193.113.4:8003);
		gomega.Expect(result).To(gomega.ContainSubstring("RDMA"))
	})
}
