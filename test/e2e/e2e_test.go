package e2e

import (
	"flag"
	"fmt"
	"github.com/netapp/beegfs-csi-driver/test/e2e/driver"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	beegfssuites "github.com/netapp/beegfs-csi-driver/test/e2e/testsuites"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/storage/testsuites"
)

var (
	perFSConfigsPath    string
	perFSConfigs        []driver.PerFSConfig
	beegfsDriver        *driver.BeegfsDriver
	beegfsDynamicDriver *driver.BeegfsDynamicDriver
	beegfsStaticDriver  *driver.BeegfsStaticDriver
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	framework.RegisterCommonFlags(flag.CommandLine)
	framework.RegisterClusterFlags(flag.CommandLine)
	registerFlags(flag.CommandLine)
	framework.AfterReadingAllFlags(&framework.TestContext)
}

func registerFlags(flags *flag.FlagSet) {
	flags.StringVar(&perFSConfigsPath, "per-fs-configs-path", "",
		"Absolute path to a configuration file defining the file systems to test.")
}

var csiTestSuites = []func() testsuites.TestSuite{
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

var _ = BeforeSuite(func() {
	framework.ExpectNotEqual(len(perFSConfigs), 0, "At least one perFSConfig must be specified")
})

var _ = Describe("E2E Tests", func() {
	perFSConfigsBytes, _ := ioutil.ReadFile(perFSConfigsPath)
	_ = yaml.UnmarshalStrict(perFSConfigsBytes, &perFSConfigs)

	staticDriver := driver.InitBeegfsStaticDriver(perFSConfigs)
	Context(testsuites.GetDriverNameWithFeatureTags(staticDriver), func() {
		testsuites.DefineTestSuite(staticDriver, csiTestSuites)
	})

	dynamicDriver := driver.InitBeegfsDynamicDriver(perFSConfigs)
	Context(testsuites.GetDriverNameWithFeatureTags(dynamicDriver), func() {
		testsuites.DefineTestSuite(dynamicDriver, csiTestSuites)
	})

	driver := driver.InitBeegfsDriver(perFSConfigs)
	Context(testsuites.GetDriverNameWithFeatureTags(driver), func() {
		testsuites.DefineTestSuite(driver, []func() testsuites.TestSuite{beegfssuites.InitBeegfsTestSuite})
	})
})

func Test(t *testing.T) {
	// Much of the code in this function is copied directly from the RunE2ETests function in
	// the k8s.io/kubernetes/test/e2e package.

	RegisterFailHandler(Fail)
	// Run tests through the Ginkgo runner with output to console + JUnit for Jenkins
	var r []Reporter
	if framework.TestContext.ReportDir != "" {
		if err := os.MkdirAll(framework.TestContext.ReportDir, 0755); err != nil {
			log.Fatalf("Failed creating report directory: %v", err)
		} else {
			r = append(r, reporters.NewJUnitReporter(path.Join(framework.TestContext.ReportDir, fmt.Sprintf("junit_%v%02d.xml", framework.TestContext.ReportPrefix, config.GinkgoConfig.ParallelNode))))
		}
	}
	log.Printf("Starting e2e run %q on Ginkgo node %d", framework.RunID, config.GinkgoConfig.ParallelNode)
	RunSpecsWithDefaultAndCustomReporters(t, "E2E Tests", r)
}
