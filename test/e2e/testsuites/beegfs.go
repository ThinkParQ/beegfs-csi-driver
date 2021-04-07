package testsuites

// The tests in this suite are in addition to the ones provided by the Kubernetes community, but based on the community
// developed framework. See k8s.io/kubernetes/test/e2e/storage/testsuites for example suites. Where possible, we have
// used framework functionality (and even entire tests, albeit with alternative setup or teardown).

// If/when test setup/cleanup for some subset of these tests becomes significantly different from another subset of
// these tests, the two subsets should be broken out into separate suites within the testsuites directory.

import (
	"fmt"
	"os/exec"
	"path"

	"github.com/netapp/beegfs-csi-driver/test/e2e/driver"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	e2eskipper "k8s.io/kubernetes/test/e2e/framework/skipper"
	e2evolume "k8s.io/kubernetes/test/e2e/framework/volume"
	"k8s.io/kubernetes/test/e2e/storage/testpatterns"
	"k8s.io/kubernetes/test/e2e/storage/testsuites"
	"k8s.io/kubernetes/test/e2e/storage/utils"
)

// Verify interface is properly implemented at compile time.
var _ testsuites.TestSuite = &beegfsTestSuite{}

type beegfsTestSuite struct {
	tsInfo testsuites.TestSuiteInfo
}

// beegfsTestSuite implements the testsuites.TestSuiteInfo interface.
func (b *beegfsTestSuite) GetTestSuiteInfo() testsuites.TestSuiteInfo {
	return b.tsInfo
}

// beegfsTestSuite implements the testsuites.TestSuiteInfo interface.
func (b *beegfsTestSuite) SkipRedundantSuite(driver testsuites.TestDriver, pattern testpatterns.TestPattern) {
	// Intentionally empty.
}

// InitBeegfsTestSuite returns a beegfsTestSuite that implements TestSuite interface
func InitBeegfsTestSuite() testsuites.TestSuite {
	return &beegfsTestSuite{
		tsInfo: testsuites.TestSuiteInfo{
			Name: "beegfs-suite",
			TestPatterns: []testpatterns.TestPattern{
				testpatterns.DefaultFsDynamicPV,
				testpatterns.DefaultFsPreprovisionedPV,
			},
			SupportedSizeRange: e2evolume.SizeRange{
				Min: "1Mi",
			},
		},
	}
}

// beegfsTestSuite implements the testsuites.TestSuiteInfo interface.
func (b *beegfsTestSuite) DefineTests(tDriver testsuites.TestDriver, pattern testpatterns.TestPattern) {
	f := framework.NewDefaultFramework("beegfs")

	var (
		d         *driver.BeegfsDriver
		resources []*testsuites.VolumeResource
		hostExec  utils.HostExec
	)

	init := func() {
		var ok bool
		d, ok = tDriver.(*driver.BeegfsDriver) // These tests use BeegfsDriver specific methods.
		if !ok {
			e2eskipper.Skipf("This test only works with a BeegfsDriver")
		}
		d.SetFSIndex(0)
		resources = make([]*testsuites.VolumeResource, 0)
		hostExec = utils.NewHostExec(f)
	}

	cleanup := func() {
		var errs []error
		for _, resource := range resources {
			errs = append(errs, resource.CleanupResource())
		}
		framework.ExpectNoError(errors.NewAggregate(errs), "while cleaning up resources")
		hostExec.Cleanup()
	}

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
			resource := testsuites.CreateVolumeResource(d, cfg, pattern, testVolumeSizeRange)
			resources = append(resources, resource) // Allow for cleanup.
			pvcs = append(pvcs, resource.Pvc)
		}

		// There is already a Kubernetes end-to-end test that tests this behavior (and more).
		testsuites.TestAccessMultipleVolumesAcrossPodRecreation(f, f.ClientSet, f.Namespace.Name, cfg.ClientNodeSelection, pvcs, true)
	})

	ginkgo.It("should correctly interpret a storage class stripe pattern", func() {
		if pattern.VolType != testpatterns.DynamicPV {
			e2eskipper.Skipf("This test only works with dynamic volumes -- skipping")
		}

		// Don't do expensive test setup until we know we'll run the test.
		init()
		defer cleanup()
		cfg, _ := d.PrepareTest(f)
		testVolumeSizeRange := b.GetTestSuiteInfo().SupportedSizeRange

		// Create storage resources including a StorageClass with non-standard striping params.
		d.SetStorageClassParams(map[string]string{
			"stripePattern/storagePoolID": "2",
			"stripePattern/chunkSize":     "1m",
			"stripePattern/numTargets":    "2",
		})
		defer d.UnsetStorageClassParams()
		resource := testsuites.CreateVolumeResource(d, cfg, pattern, testVolumeSizeRange)
		resources = append(resources, resource) // Allow for cleanup.

		// Use beegfs-ctl to investigate the striping parameters on the created directory.
		// If we develop more tests that use beegfs-ctl, it would be better to streamline this (e.g. have the test
		// driver return a mostly complete beegfs-ctl command to run or pass beegfs-ctl arguments to the driver).
		mountPath := d.GetFSConfig().MountPath
		volDirPathBeegfsRoot := path.Join(d.GetFSConfig().VolDirBasePathBeegfsRoot, resource.Pv.Name)
		cmd := exec.Command("/usr/sbin/beegfs-ctl", fmt.Sprintf("--mount=%s", mountPath), "--unmounted",
			"--getentryinfo", volDirPathBeegfsRoot)
		output, err := cmd.CombinedOutput()

		framework.ExpectNoError(err, string(output))
		gomega.Expect(string(output)).To(gomega.ContainSubstring("Storage Pool: 2"))
		gomega.Expect(string(output)).To(gomega.ContainSubstring("Chunksize: 1M"))
		gomega.Expect(string(output)).To(gomega.ContainSubstring("Number of storage targets: desired: 2"))
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
		resource := testsuites.CreateVolumeResource(d, cfg, pattern, testVolumeSizeRange)
		resources = append(resources, resource) // Allow for cleanup.

		// Create a pod to consume the storage resource.
		podConfig := e2epod.Config{
			NS:      cfg.Framework.Namespace.Name,
			PVCs:    []*corev1.PersistentVolumeClaim{resource.Pvc},
			ImageID: e2evolume.GetDefaultTestImageID(),
		}
		pod, err := e2epod.CreateSecPodWithNodeSelection(f.ClientSet, &podConfig, framework.PodStartTimeout)
		defer func() {
			// ExpectNoError() must be wrapped in a func() or it will be evaluated (and the pod will be deleted) now.
			framework.ExpectNoError(e2epod.DeletePodWithWait(f.ClientSet, pod))
		}()
		framework.ExpectNoError(err)

		// Query /proc for connection information associated with this storage resource.
		node, err := f.ClientSet.CoreV1().Nodes().Get(context.TODO(), pod.Spec.NodeName, metav1.GetOptions{})
		framework.ExpectNoError(err)
		cmd := fmt.Sprintf("cat $(dirname $(grep -l %s /proc/fs/beegfs/*/config))/storage_nodes", resource.Pv.Name)
		result, err := hostExec.IssueCommandWithResult(cmd, node)
		framework.ExpectNoError(err)

		// Output looks like:
		// localhost [ID: 1]
		//    Connections: TCP: 4 (10.193.113.4:8003);
		gomega.Expect(result).To(gomega.ContainSubstring("RDMA"))
	})
}
