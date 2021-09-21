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
	"context"
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
	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	e2eframework "k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	storageframework "k8s.io/kubernetes/test/e2e/storage/framework"
	storagesuites "k8s.io/kubernetes/test/e2e/storage/testsuites"
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

var _ = ginkgo.BeforeSuite(func() {
	cs, err := e2eframework.LoadClientset()
	e2eframework.ExpectNoError(err, "expected to load a client set")

	// Get the controller Pod (usually csi-beegfs-controller-0 in default or kube-system namespace). Wait for it to be
	// running so we don't read the stale ConfigMap from a terminated deployment.
	controllerPods, err := e2epod.WaitForPodsWithLabelRunningReady(cs, "",
		labels.SelectorFromSet(map[string]string{"app": "csi-beegfs-controller"}), 1, e2eframework.PodStartTimeout)
	e2eframework.ExpectNoError(err, "expected to find exactly one controller pod")

	// Get the name of the ConfigMap from the controller Pod.
	var driverCMName string
	controllerNS := controllerPods.Items[0].ObjectMeta.Namespace
	for _, volume := range controllerPods.Items[0].Spec.Volumes {
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
	var pluginConfig beegfsv1.PluginConfig
	err = yaml.UnmarshalStrict([]byte(driverConfigString), &pluginConfig)
	e2eframework.ExpectNoError(err, "expected to successfully unmarshal ConfigMap")
	e2eframework.ExpectNotEqual(len(pluginConfig.FileSystemSpecificConfigs), 0,
		"expected csi-beegfs-config.yaml to include at least one config")
	beegfsDriver.SetPerFSConfigs(pluginConfig.FileSystemSpecificConfigs)
	beegfsDynamicDriver.SetPerFSConfigs(pluginConfig.FileSystemSpecificConfigs)
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
