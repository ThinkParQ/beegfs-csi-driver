/*
Copyright 2015, 2018 The Kubernetes Authors.

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

package e2e

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	beegfsv1 "github.com/netapp/beegfs-csi-driver/operator/api/v1"
	"github.com/netapp/beegfs-csi-driver/test/e2e/driver"
	beegfssuites "github.com/netapp/beegfs-csi-driver/test/e2e/testsuites"
	"github.com/netapp/beegfs-csi-driver/test/e2e/utils"
	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
	e2eframework "k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/testfiles"
	storageframework "k8s.io/kubernetes/test/e2e/storage/framework"
	storagesuites "k8s.io/kubernetes/test/e2e/storage/testsuites"
	e2etestingmanifests "k8s.io/kubernetes/test/e2e/testing-manifests"
	"sigs.k8s.io/yaml"
)

// A pointer to a BeegfsDriver is kept here as a global variable so its perFSConfigs can be set with
// beegfsDriver.SetPerFSConfigs in BeforeSuite. Another option would be to SetPerFSConfigs in beegfsDriver.PrepareTest,
// but this would cause us to query the K8s API server for every test. There are likely additional strategies for
// setting the BeegfsDriver's perFSConfigs, but Ginkgo's "order of execution" makes things difficult.
var beegfsDriver *driver.BeegfsDriver
var beegfsDynamicDriver *driver.BeegfsDynamicDriver

// Variables to be set by flags.
var dynamicVolDirBasePathBeegfsRoot, staticVolDirBasePathBeegfsRoot, staticVolDirName string

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	e2eframework.RegisterCommonFlags(flag.CommandLine)
	e2eframework.RegisterClusterFlags(flag.CommandLine)
	e2eframework.AfterReadingAllFlags(&e2eframework.TestContext)
	flag.StringVar(&dynamicVolDirBasePathBeegfsRoot, "dynamic-vol-dir-base-path", "/e2e-test/dynamic", "Path (from BeeGFS root) to the pre-existing base directory for dynamic provisioning tests. Defaults to /e2e/dynamic.")
	flag.StringVar(&staticVolDirBasePathBeegfsRoot, "static-vol-dir-base-path", "/e2e-test/static", "Path (from BeeGFS root) to the pre-existing base directory for static provisioning tests. Defaults to /e2e/static.")
	flag.StringVar(&staticVolDirName, "static-vol-dir-name", "", "Name of the pre-existing directory under static-vol-dir-base-path to be used as a volume for static provisioning tests. Static provisioning tests are skipped if left unset.")
}

var beegfsSuitesToRun = []func() storageframework.TestSuite{
	beegfssuites.InitBeegfsTestSuite,
}

// This method of preparing Kubernetes tests to run is heavily adapted from the
// github.com/kubernetes-sigs/aws-ebs-csi-driver/tests/e2e-kubernetes package
// (https://github.com/kubernetes-sigs/aws-ebs-csi-driver/blob/v0.10.1/tests/e2e-kubernetes/e2e_test.go#L81-L91).
// The general structure of this file is loosely adapted from the same package.
var k8sSuitesToRun = []func() storageframework.TestSuite{
	storagesuites.InitDisruptiveTestSuite,
	storagesuites.InitEphemeralTestSuite,           // No specs run without the GenericEphemeralVolumes feature gate.
	storagesuites.InitFsGroupChangePolicyTestSuite, // No specs run because Capabilities[CapFsGroup] = false.
	storagesuites.InitMultiVolumeTestSuite,
	// The provisioning suite tests provisioning storage with various options like NTFS and cloning.
	// The driver currently doesn't support any of these options so it is expected that all provisioning tests will skip.
	storagesuites.InitProvisioningTestSuite,  // No specs run.
	storagesuites.InitSnapshottableTestSuite, // No specs run.
	// One subpath test (should be able to unmount after the subpath directory is deleted [LinuxOnly]) fails
	// consistently and must be skipped in the "go test" or "ginkgo" command.
	// TODO(webere, A200): Fix broken subpath functionality.
	storagesuites.InitSubPathTestSuite,
	storagesuites.InitTopologyTestSuite,     // No specs run.
	storagesuites.InitVolumeExpandTestSuite, // No specs run.
	storagesuites.InitVolumeIOTestSuite,
	storagesuites.InitVolumeStressTestSuite,
	storagesuites.InitVolumeLimitsTestSuite, // No specs run.
	storagesuites.InitVolumeModeTestSuite,
}

var _ = ginkgo.SynchronizedBeforeSuite(func() []byte {
	cs, err := e2eframework.LoadClientset()
	e2eframework.ExpectNoError(err, "expected to load a client set")

	// Check for orphaned mounts on all nodes. This MUST be done in SynchronizedBeforeSuite (instead of BeforeSuite)
	// in case one process is fast and starts creating BeeGFS volumes before another is done checking. If the check
	// fails here, it is likely that a different test run (or a developer acting outside of the test infrastructure)
	// caused mounts to be orphaned.
	utils.VerifyNoOrphanedMounts(cs)

	// Get the driver's PluginConfig from the deployed ConfigMap and pass it as a byte slice (required by
	// SynchronizedBeforeSuite) to all nodes.
	driverCM := utils.GetConfigMapInUse(cs)
	driverConfigString, ok := driverCM.Data["csi-beegfs-config.yaml"]
	e2eframework.ExpectEqual(ok, true, "expected a csi-beegfs-config.yaml in ConfigMap")
	return []byte(driverConfigString)

}, func(driverConfigBytes []byte) {
	// Unmarshal the ConfigMap and use it to populate the global BeegfsDriver's perFSConfigs.
	var pluginConfig beegfsv1.PluginConfig
	err := yaml.UnmarshalStrict(driverConfigBytes, &pluginConfig)
	e2eframework.ExpectNoError(err, "expected to successfully unmarshal PluginConfig")
	e2eframework.ExpectNotEqual(len(pluginConfig.FileSystemSpecificConfigs), 0,
		"expected PluginConfig to specifically reference at least one file system")
	beegfsDriver.SetPerFSConfigs(pluginConfig.FileSystemSpecificConfigs)
	beegfsDynamicDriver.SetPerFSConfigs(pluginConfig.FileSystemSpecificConfigs)
})

var _ = ginkgo.SynchronizedAfterSuite(func() {}, func() {
	cs, err := e2eframework.LoadClientset()
	e2eframework.ExpectNoError(err, "expected to load a client set") // All remaining work requires a clientset.
	// Archive logs from node and controller service pods. Do this BEFORE VerifyNoOrphanedMounts because
	// VerifyNoOrphanedMounts does not generate service logs and because VerifyNoOrphanedMounts can fail.
	utils.ArchiveServiceLogs(cs, e2eframework.TestContext.ReportDir)
	// Check for orphaned mounts on all nodes. This MUST be done in SynchronizedAfterSuite (instead of AfterSuite)
	// because some processes will be done running tests and check while others are still creating BeeGFS volumes. If
	// the check fails here, it is likely that code changes introduced for this test run caused mounts to be orphaned.
	// Do this check AFTER ArchiveServiceLogs to ensure logs are captured and allow failure to ensure visibility of
	// orphaned mounts.
	utils.VerifyNoOrphanedMounts(cs)
})

var _ = ginkgo.Describe("E2E Tests", func() {
	beegfsDriver = driver.InitBeegfsDriver(dynamicVolDirBasePathBeegfsRoot, staticVolDirBasePathBeegfsRoot,
		staticVolDirName)
	ginkgo.Context(storageframework.GetDriverNameWithFeatureTags(beegfsDriver), func() {
		storageframework.DefineTestSuites(beegfsDriver, beegfsSuitesToRun)
	})

	beegfsDynamicDriver = driver.InitBeegfsDynamicDriver(dynamicVolDirBasePathBeegfsRoot)
	ginkgo.Context(storageframework.GetDriverNameWithFeatureTags(beegfsDynamicDriver), func() {
		storageframework.DefineTestSuites(beegfsDynamicDriver, k8sSuitesToRun)
	})
})

func Test(t *testing.T) {
	// Much of the code in this function is copied directly from the RunE2ETests function in
	// the k8s.io/kubernetes/test/e2e package
	// (https://github.com/kubernetes/kubernetes/blob/v1.19.0/test/e2e/e2e.go#L92-L131).

	config.DefaultReporterConfig.NoColor = true
	gomega.RegisterFailHandler(ginkgo.Fail)
	testfiles.AddFileSource(e2etestingmanifests.GetE2ETestingManifestsFS())
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
