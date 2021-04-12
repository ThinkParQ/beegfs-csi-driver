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
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	"k8s.io/kubernetes/test/e2e/storage/testsuites"
)

// A pointer to a BeegfsDriver is kept here as a global variable so its perFSConfigs can be set with
// beegfsDriver.SetPerFSConfigs in BeforeSuite. Another option would be to SetPerFSConfigs in beegfsDriver.PrepareTest,
// but this would cause us to query the K8s API server for every test. There are likely additional strategies for
// setting the BeegfsDriver's perFSConfigs, but Ginkgo's "order or execution" makes things difficult.
var beegfsDriver *driver.BeegfsDriver
var beegfsDynamicDriver *driver.BeegfsDynamicDriver

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	framework.RegisterCommonFlags(flag.CommandLine)
	framework.RegisterClusterFlags(flag.CommandLine)
	framework.AfterReadingAllFlags(&framework.TestContext)

}

var beegfsSuitesToRun = []func() testsuites.TestSuite{
	beegfssuites.InitBeegfsTestSuite,
}

var k8sSuitesToRun = []func() testsuites.TestSuite{
	// This list of test results in 31 pass and 1 fail.
	testsuites.InitDisruptiveTestSuite,
	//testsuites.InitEphemeralTestSuite, // 2 fail when WaitForFirstConsumer is enabled. Currently disabled.
	//testsuites.InitFsGroupChangePolicyTestSuite, // No specs run as long as Capabilities[CapFsGroup] = false.
	testsuites.InitMultiVolumeTestSuite,
	//testsuites.InitProvisioningTestSuite, // No specs run. TODO: Look into "should provision storage with mount options."
	//testsuites.InitSnapshottableTestSuite, // No specs run.
	//testsuites.InitSubPathTestSuite, // 17 pass, 1 fails (after a long time). TODO: Investigate failure.
	//testsuites.InitTopologyTestSuite, // No specs run.
	//testsuites.InitVolumeExpandTestSuite, // No specs run.
	testsuites.InitVolumeIOTestSuite,
	testsuites.InitVolumeStressTestSuite,
	//testsuites.InitVolumeLimitsTestSuite, // No specs run.
	testsuites.InitVolumeModeTestSuite,
}

var _ = ginkgo.BeforeSuite(func() {
	cs, err := framework.LoadClientset()
	framework.ExpectNoError(err, "expected to load a client set")

	// Get the controller Pod (usually csi-beegfs-controller-0 in default or kube-system namespace).
	controllerPods, err := e2epod.GetPods(cs, "", map[string]string{"app": "csi-beegfs-controller"})
	framework.ExpectNoError(err, "expected to find controller Pod")
	framework.ExpectEqual(len(controllerPods), 1, "expected only one controller pod")

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
	framework.ExpectNoError(err, "expected to read ConfigMap")
	driverConfigString, ok := driverCM.Data["csi-beegfs-config.yaml"]
	framework.ExpectEqual(ok, true, "expected a csi-beegfs-config.yaml in ConfigMap")

	// Unmarshal the ConfigMap and use it to populate the global BeegfsDriver's perFSConfigs.
	var pluginConfig beegfs.PluginConfig
	err = yaml.UnmarshalStrict([]byte(driverConfigString), &pluginConfig)
	framework.ExpectNoError(err, "expected to successfully unmarshal ConfigMap")
	beegfsDriver.SetPerFSConfigs(pluginConfig.FileSystemSpecificConfigs)
	beegfsDynamicDriver.SetPerFSConfigs(pluginConfig.FileSystemSpecificConfigs)

})

var _ = ginkgo.Describe("E2E Tests", func() {
	beegfsDriver = driver.InitBeegfsDriver()
	ginkgo.Context(testsuites.GetDriverNameWithFeatureTags(beegfsDriver), func() {
		testsuites.DefineTestSuite(beegfsDriver, beegfsSuitesToRun)
	})

	beegfsDynamicDriver = driver.InitBeegfsDynamicDriver()
	ginkgo.Context(testsuites.GetDriverNameWithFeatureTags(beegfsDynamicDriver), func() {
		testsuites.DefineTestSuite(beegfsDynamicDriver, k8sSuitesToRun)
	})
})

func Test(t *testing.T) {
	// Much of the code in this function is copied directly from the RunE2ETests function in
	// the k8s.io/kubernetes/test/e2e package.

	gomega.RegisterFailHandler(ginkgo.Fail)
	// Run tests through the Ginkgo runner with output to console + JUnit for Jenkins
	var r []ginkgo.Reporter
	if framework.TestContext.ReportDir != "" {
		if err := os.MkdirAll(framework.TestContext.ReportDir, 0755); err != nil {
			log.Fatalf("Failed creating report directory: %v", err)
		} else {
			r = append(r, reporters.NewJUnitReporter(path.Join(framework.TestContext.ReportDir, fmt.Sprintf("junit_%v%02d.xml", framework.TestContext.ReportPrefix, config.GinkgoConfig.ParallelNode))))
		}
	}
	log.Printf("Starting e2e run %q on Ginkgo node %d", framework.RunID, config.GinkgoConfig.ParallelNode)
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "E2E Tests", r)
}
