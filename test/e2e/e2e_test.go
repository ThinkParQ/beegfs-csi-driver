package e2e

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/netapp/beegfs-csi-driver/pkg/beegfs"
	"github.com/netapp/beegfs-csi-driver/test/e2e/driver"
	beegfssuites "github.com/netapp/beegfs-csi-driver/test/e2e/testsuites"
	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	e2eframework "k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	storageframework "k8s.io/kubernetes/test/e2e/storage/framework"
	storagesuites "k8s.io/kubernetes/test/e2e/storage/testsuites"
)

// A pointer to a BeegfsDriver is kept here as a global variable so its perFSConfigs can be set with
// beegfsDriver.SetPerFSConfigs in BeforeSuite. Another option would be to SetPerFSConfigs in beegfsDriver.PrepareTest,
// but this would cause us to query the K8s API server for every test. There are likely additional strategies for
// setting the BeegfsDriver's perFSConfigs, but Ginkgo's "order or execution" makes things difficult.
var beegfsDriver *driver.BeegfsDriver
var beegfsDynamicDriver *driver.BeegfsDynamicDriver

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	e2eframework.RegisterCommonFlags(flag.CommandLine)
	e2eframework.RegisterClusterFlags(flag.CommandLine)
	e2eframework.AfterReadingAllFlags(&e2eframework.TestContext)

}

var beegfsSuitesToRun = []func() storageframework.TestSuite{
	beegfssuites.InitBeegfsTestSuite,
}

var k8sSuitesToRun = []func() storageframework.TestSuite{
	// This list of test results in 31 pass and 1 fail.
	storagesuites.InitDisruptiveTestSuite,
	//storagesuites.InitEphemeralTestSuite, // 2 fail when WaitForFirstConsumer is enabled. Currently disabled.
	//storagesuites.InitFsGroupChangePolicyTestSuite, // No specs run as long as Capabilities[CapFsGroup] = false.
	storagesuites.InitMultiVolumeTestSuite,
	//storagesuites.InitProvisioningTestSuite, // No specs run. TODO: Look into "should provision storage with mount options."
	//storagesuites.InitSnapshottableTestSuite, // No specs run.
	//storagesuites.InitSubPathTestSuite, // 17 pass, 1 fails (after a long time). TODO: Investigate failure.
	//storagesuites.InitTopologyTestSuite, // No specs run.
	//storagesuites.InitVolumeExpandTestSuite, // No specs run.
	storagesuites.InitVolumeIOTestSuite,
	storagesuites.InitVolumeStressTestSuite,
	//storagesuites.InitVolumeLimitsTestSuite, // No specs run.
	storagesuites.InitVolumeModeTestSuite,
}

var _ = ginkgo.BeforeSuite(func() {
	cs, err := e2eframework.LoadClientset()
	e2eframework.ExpectNoError(err, "expected to load a client set")

	// Get the controller Pod (usually csi-beegfs-controller-0 in default or kube-system namespace).
	controllerPods, err := e2epod.GetPods(cs, "", map[string]string{"app": "csi-beegfs-controller"})
	e2eframework.ExpectNoError(err, "expected to find controller Pod")
	e2eframework.ExpectEqual(len(controllerPods), 1, "expected only one controller pod")

	// Get the name of the ConfigMap from the controller Pod.
	var driverCMName string
	controllerNS := controllerPods[0].ObjectMeta.Namespace
	for _, volume := range controllerPods[0].Spec.Volumes {
		if volume.Name == "config-dir" {
			driverCMName = volume.ConfigMap.Name
		}
	}

	// Read the ConfigMap.
	driverCM, err := cs.CoreV1().ConfigMaps(controllerNS).Get(context.TODO(), driverCMName, metav1.GetOptions{})
	e2eframework.ExpectNoError(err, "expected to read ConfigMap")
	driverConfigString, ok := driverCM.Data["csi-beegfs-config.yaml"]
	e2eframework.ExpectEqual(ok, true, "expected a csi-beegfs-config.yaml in ConfigMap")

	// Unmarshal the ConfigMap and use it to populate the global BeegfsDriver's perFSConfigs.
	var pluginConfig beegfs.PluginConfig
	err = yaml.UnmarshalStrict([]byte(driverConfigString), &pluginConfig)
	e2eframework.ExpectNoError(err, "expected to successfully unmarshal ConfigMap")
	beegfsDriver.SetPerFSConfigs(pluginConfig.FileSystemSpecificConfigs)
	beegfsDynamicDriver.SetPerFSConfigs(pluginConfig.FileSystemSpecificConfigs)

})

var _ = ginkgo.Describe("E2E Tests", func() {
	beegfsDriver = driver.InitBeegfsDriver()
	ginkgo.Context(storageframework.GetDriverNameWithFeatureTags(beegfsDriver), func() {
		storageframework.DefineTestSuites(beegfsDriver, beegfsSuitesToRun)
	})

	beegfsDynamicDriver = driver.InitBeegfsDynamicDriver()
	ginkgo.Context(storageframework.GetDriverNameWithFeatureTags(beegfsDynamicDriver), func() {
		storageframework.DefineTestSuites(beegfsDynamicDriver, k8sSuitesToRun)
	})
})

func Test(t *testing.T) {
	// Much of the code in this function is copied directly from the RunE2ETests function in
	// the k8s.io/kubernetes/test/e2e package.

	config.DefaultReporterConfig.NoColor = true
	gomega.RegisterFailHandler(ginkgo.Fail)
	// Run tests through the Ginkgo runner with output to console + JUnit for Jenkins
	var r []ginkgo.Reporter
	if e2eframework.TestContext.ReportDir != "" {
		if err := os.MkdirAll(e2eframework.TestContext.ReportDir, 0755); err != nil {
			log.Fatalf("Failed creating report directory: %v", err)
		} else {
			r = append(r, reporters.NewJUnitReporter(path.Join(e2eframework.TestContext.ReportDir,
				fmt.Sprintf("junit_%v%02d.xml", e2eframework.TestContext.ReportPrefix,
					config.GinkgoConfig.ParallelNode))))
		}
	}
	log.Printf("Starting e2e run %q on Ginkgo node %d", e2eframework.RunID, config.GinkgoConfig.ParallelNode)
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "E2E Tests", r)
}
