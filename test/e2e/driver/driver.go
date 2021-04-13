package driver

import (
	"path"

	"github.com/netapp/beegfs-csi-driver/pkg/beegfs"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubernetes/test/e2e/framework"
	e2eskipper "k8s.io/kubernetes/test/e2e/framework/skipper"
	"k8s.io/kubernetes/test/e2e/storage/testpatterns"
	"k8s.io/kubernetes/test/e2e/storage/testsuites"
)

// Verify expected interfaces are properly implemented at compile time.
var _ testsuites.TestDriver = &BeegfsDriver{}
var _ testsuites.TestDriver = &BeegfsDynamicDriver{}
var _ testsuites.DynamicPVTestDriver = &BeegfsDriver{}
var _ testsuites.DynamicPVTestDriver = &BeegfsDynamicDriver{}
var _ testsuites.PreprovisionedVolumeTestDriver = &BeegfsDriver{}

// baseBeegfsDriver is unexported and cannot be directly accessed or instantiated. All exported drivers use it as
// their underlying data structure and can call its internal methods.
type baseBeegfsDriver struct {
	driverInfo                      testsuites.DriverInfo
	perFSConfigs                    []beegfs.FileSystemSpecificConfig
	fsIndex                         int
	extraSCParams                   map[string]string
	dynamicVolDirBasePathBeegfsRoot string // Set once on initialization (e.g. e.g. /e2e-test/dynamic).
	staticVolDirBasePathBeegfsRoot  string // Set once on initialization (e.g. /e2e-test/static).
	staticDirName                   string // Optionally set by a test (e.g.static2).
	staticDirNameOriginal           string // Set once on initialization (e.g. static1).
}

// BeegfsDriver is an exported driver that implements the testsuites.TestDriver, testsuites.DynamicPVTestDriver,
// testsuites.PreprovisionedVolumeTestDriver, and testsuites.PreprovisionedPVTestDriver interfaces. It is intended to
// be used in all beegfs-csi-driver specific tests.
type BeegfsDriver struct {
	*baseBeegfsDriver
}

// BeegfsDynamicDriver is an exported driver that implements the testsuites.TestDriver and
// testsuites.DynamicPVTestDriver interfaces. It intentionally does not implement the
// testsuites.PreprovisionedVolumeTestDriver and testsuites.PreprovisionedPVTestDriver interfaces. It is intended to be
// used for K8s built-in tests, which may use the pre-provisioned interface in unanticipated ways if allowed.
type BeegfsDynamicDriver struct {
	*baseBeegfsDriver
}

// baseBeegfsDriver implements the testsuites.TestDriver interface.
func (d *baseBeegfsDriver) GetDriverInfo() *testsuites.DriverInfo {
	return &d.driverInfo
}

// baseBeegfsDriver implements the testsuites.TestDriver interface.
func (d *baseBeegfsDriver) SkipUnsupportedTest(pattern testpatterns.TestPattern) {
	switch pattern.BindingMode {
	// Late binding ephemeral tests fail unless skipped, but they probably shouldn't.
	// TODO: Figure out why.
	case storagev1.VolumeBindingWaitForFirstConsumer:
		e2eskipper.Skipf("Driver %s does not support binding mode %s", d.driverInfo.Name, pattern.BindingMode)
	}
}

// baseBeegfsDriver implements the testsuites.TestDriver interface.
func (d *baseBeegfsDriver) PrepareTest(f *framework.Framework) (*testsuites.PerTestConfig, func()) {
	config := &testsuites.PerTestConfig{
		Driver:    d,
		Prefix:    "beegfs",
		Framework: f,
	}
	return config, func() {}
}

// initBaseBeegfsDriver handles basic initialization shared across all exported drivers.
func initBaseBeegfsDriver() *baseBeegfsDriver {
	return &baseBeegfsDriver{
		driverInfo: testsuites.DriverInfo{
			Name: "beegfs",
			// FeatureTag:
			// MaxFileSize:
			// SupportedSizeRange:
			SupportedFsType: sets.NewString(""),
			// Map of string for supported mount option
			// SupportedMountOption:
			// RequiredMountOption:
			Capabilities: map[testsuites.Capability]bool{
				testsuites.CapPersistence:         true,
				testsuites.CapBlock:               false,
				testsuites.CapFsGroup:             false,
				testsuites.CapExec:                true,
				testsuites.CapSnapshotDataSource:  false,
				testsuites.CapPVCDataSource:       false,
				testsuites.CapMultiPODs:           true,
				testsuites.CapRWX:                 true,
				testsuites.CapControllerExpansion: false,
				testsuites.CapNodeExpansion:       false,
				testsuites.CapVolumeLimits:        false,
				testsuites.CapSingleNodeVolume:    false, // TODO: Experiment more. Setting this to true seems to skip some multi-node tests.
				testsuites.CapTopology:            false,
			},
			// RequiredAccessModes:
			// TopologyKeys:
			// NumAllowedTopologies:
			StressTestOptions: &testsuites.StressTestOptions{
				NumPods:     10,
				NumRestarts: 3,
			},
			// VolumeSnapshotStressTestOptions:
		},
		perFSConfigs:                    make([]beegfs.FileSystemSpecificConfig, 0),
		fsIndex:                         0,
		dynamicVolDirBasePathBeegfsRoot: path.Join("e2e-test", "dynamic"),
		staticVolDirBasePathBeegfsRoot:  path.Join("e2e-test", "static"),
		staticDirName:                   "static1",
		staticDirNameOriginal:           "static1",
	}
}

// InitBeegfsDriver returns a pointer to a BeegfsDriver.
func InitBeegfsDriver() *BeegfsDriver {
	return &BeegfsDriver{baseBeegfsDriver: initBaseBeegfsDriver()}
}

// InitBeegfsDynamicDriver returns a pointer to a BeegfsDynamicDriver.
func InitBeegfsDynamicDriver() *BeegfsDynamicDriver {
	return &BeegfsDynamicDriver{baseBeegfsDriver: initBaseBeegfsDriver()}
}

// baseBeegfsDriver directly implements the testsuites.DynamicPVTestDriver interface.
func (d *baseBeegfsDriver) GetDynamicProvisionStorageClass(config *testsuites.PerTestConfig,
	fsType string) *storagev1.StorageClass {
	bindingMode := storagev1.VolumeBindingImmediate
	params := map[string]string{
		"sysMgmtdHost":   d.perFSConfigs[d.fsIndex].SysMgmtdHost,
		"volDirBasePath": d.dynamicVolDirBasePathBeegfsRoot,
	}
	if d.extraSCParams != nil {
		for k, v := range d.extraSCParams {
			params[k] = v
		}
	}
	return testsuites.GetStorageClass("beegfs.csi.netapp.com", params, &bindingMode,
		config.Framework.Namespace.Name, "e2e-sc-")
}

// BeegfsDriver implements the testsuites.PreprovisionedVolumeTestDriver interface.
// CreateVolume returns a testsuites.TestVolume that appropriately references a pre-created directory on a BeeGFS file
// system known to the driver. Tests can use SetFSIndex and SetStaticDirName to modify its behavior.
func (d *BeegfsDriver) CreateVolume(config *testsuites.PerTestConfig, volumeType testpatterns.TestVolType) testsuites.TestVolume {
	fsConfig := d.perFSConfigs[d.fsIndex]
	volDirPathBeegfsRoot := path.Join(d.staticVolDirBasePathBeegfsRoot, d.staticDirName)
	return beegfsVolume{
		volumeID: beegfs.NewBeegfsUrl(fsConfig.SysMgmtdHost, volDirPathBeegfsRoot),
	}
}

// BeegfsDriver implements the testsuites.PreprovisionedPVTestDriver interface.
// GetPersistentVolumeSource returns PersistentVolumeSource that appropriately references a pre-created directory
// on a BeeGFS file system known to the driver.
func (d *BeegfsDriver) GetPersistentVolumeSource(readOnly bool, fsType string,
	testVolume testsuites.TestVolume) (*corev1.PersistentVolumeSource, *corev1.VolumeNodeAffinity) {
	beegfsVol := testVolume.(beegfsVolume) // Assert that we have a beegfsVolume.
	csiSource := corev1.CSIPersistentVolumeSource{
		Driver:       "beegfs.csi.netapp.com",
		VolumeHandle: beegfsVol.volumeID,
		ReadOnly:     readOnly,
		FSType:       fsType,
	}
	volumeSource := corev1.PersistentVolumeSource{
		CSI: &csiSource,
	}
	return &volumeSource, nil
}

// SetStorageClassParams injects additional parameters into the driver. These parameters will appear in all
// generated StorageClasses until UnsetStorageClassParams() is called.
func (d *baseBeegfsDriver) SetStorageClassParams(extraSCParams map[string]string) {
	d.extraSCParams = extraSCParams
}

// UnsetStorageClassParams() reverses SetStorageClassParams.
func (d *baseBeegfsDriver) UnsetStorageClassParams() {
	d.extraSCParams = nil
}

// SetFSIndex determines which PerFSConfig will be used for various volume provisioning related tasks. It intentionally
// has no internal error correction. Use GetNumFS to determine the maximum fsIndex to set. If you set fsIndex above the
// maximum, tests will fail.
func (d *baseBeegfsDriver) SetFSIndex(fsIndex int) {
	d.fsIndex = fsIndex
}

// GetNumFS returns the maximum fsIndex that should be used with setFSIndex. It may also be useful in skipping certain
// beegfs-csi-driver specific tests (e.g. a test that requires two different file systems should be skipped if
// GetNumFS returns 1.
func (d *baseBeegfsDriver) GetNumFS() int {
	return len(d.perFSConfigs)
}

// SetFSIndexForRDMA looks for an RDMA capable file system and sets fsIndex to refer to the first one it finds. It
// returns false if there are no RDMA capable file systems.
func (d *baseBeegfsDriver) SetFSIndexForRDMA() bool {
	for i, cfg := range d.perFSConfigs {
		if boolString, ok := cfg.Config.BeegfsClientConf["connUseRDMA"]; ok {
			if boolString == "true" {
				d.SetFSIndex(i)
				return true
			}
		}
	}
	return false // There are no RDMA capable file systems.
}

// SetPerFSConfigs sets perFSConfigs from a slice of beegfs.FileSystemSpecificConfigs.
func (d *baseBeegfsDriver) SetPerFSConfigs(perFSConfigs []beegfs.FileSystemSpecificConfig) {
	d.perFSConfigs = perFSConfigs
}

// SetStaticDirName controls the volDirPathBeegfsRoot used by CreateVolume and (by extension)
// getPersistentVolumeSource. Set it to refer to an existing directory under staticVolDirBasePathBeegfsRoot on a BeeGFS
// file system known to the driver.
func (d *baseBeegfsDriver) SetStaticDirName(staticDirName string) {
	d.staticDirName = staticDirName
}

// UnsetStaticDirName reverses SetStaticDirName.
func (d *baseBeegfsDriver) UnsetStaticDirName() {
	d.staticDirName = d.staticDirNameOriginal
}

// beegfsVolume implements the testsuites.TestVolume interface.
// The end-to-end Kubernetes tests and various framework functions expect to handle a testsuites.TestVolume that knows
// how to delete itself (out-of-band of a running CSI driver).
type beegfsVolume struct {
	volumeID string // pkg/beegfs.beegfsVolume.volumeID
}

// beegfsVolume implements the testsuites.TestVolume interface.
// When properly set, volDirPath is the path from the root of the testing node to a directory on a mounted BeeGFS file
// system.
func (v beegfsVolume) DeleteVolume() {
	// Intentionally empty.
	// Our pre-provisioned volumes are not created on demand and are not deleted at the end of a test.
}