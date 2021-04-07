package driver

import (
	"os"
	"path"

	"github.com/netapp/beegfs-csi-driver/pkg/beegfs"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/storage/names"
	"k8s.io/kubernetes/test/e2e/framework"
	e2eskipper "k8s.io/kubernetes/test/e2e/framework/skipper"
	"k8s.io/kubernetes/test/e2e/storage/testpatterns"
	"k8s.io/kubernetes/test/e2e/storage/testsuites"
)

// Verify expected interfaces are properly implemented at compile time.
var _ testsuites.TestDriver = &BeegfsDriver{}
var _ testsuites.DynamicPVTestDriver = &BeegfsDriver{}
var _ testsuites.DynamicPVTestDriver = &BeegfsDynamicDriver{}
var _ testsuites.PreprovisionedVolumeTestDriver = &BeegfsDriver{}
var _ testsuites.PreprovisionedVolumeTestDriver = &BeegfsStaticDriver{}
var _ testsuites.PreprovisionedPVTestDriver = &BeegfsDriver{}
var _ testsuites.PreprovisionedPVTestDriver = &BeegfsStaticDriver{}
var _ testsuites.TestVolume = beegfsVolume{}

// baseBeegfsDriver is unexported and cannot be directly accessed or instantiated. All exported drivers use it as
// their underlying data structure and can call its internal methods.
type baseBeegfsDriver struct {
	driverInfo    testsuites.DriverInfo
	perFSConfigs  []PerFSConfig
	fsIndex       int
	extraSCParams map[string]string
}

// BeegfsDriver is an exported driver that implements the testsuites.TestDriver, testsuites.DynamicPVTestDriver,
// testsuites.PreprovisionedVolumeTestDriver, and testsuites.PreprovisionedPVTestDriver interfaces. It is intended to
// be used in all beegfs-csi-driver specific tests.
type BeegfsDriver struct {
	*baseBeegfsDriver
}

// BeegfsStaticDriver is an exported driver that implements the testsuites.TestDriver,
// testsuites.PreprovisionedVolumeTestDriver, and testsuites.PreprovisionedPVTestDriver interfaces. It intentionally
// does not implement the testsuites.DynamicPVTestDriver interface because Kubernetes end-to-end tests skip
// pre-provisioned test patterns for drivers that implement that interface.
type BeegfsStaticDriver struct {
	*baseBeegfsDriver
}

// BeegfsDynamicDriver is an exported driver that implements the testsuites.TestDriver and
// testsuites.DynamicPVTestDriver interfaces. It intentionally does not implement the
// testsuites.PreprovisionedVolumeTestDriver and testsuites.PreprovisionedPVTestDriver interfaces because all
// pre-provisioned test patterns are handled by BeegfsStaticDriver.
type BeegfsDynamicDriver struct {
	*baseBeegfsDriver
}

// PerFSConfig contains all of the information the a driver based on baseBeegfsDriver needs to output PersistentVolume
// or StorageClass source or create PersistentVolumes out-of-band. Additionally, tests may need to access perFSConfigs
// to correctly format beegfs-ctl commands, etc. A list of perFSConfigs must be loaded into the driver for proper
// operation.
type PerFSConfig struct {
	SysMgmtdHost             string `yaml:"sysMgmtdHost"`
	MountPath                string `yaml:"mountPath"`
	VolDirBasePathBeegfsRoot string `yaml:"volDirBasePathBeegfsRoot"`
	RDMACapable              bool   `yaml:"rdmaCapable"`
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
func initBaseBeegfsDriver(perFSConfigs []PerFSConfig) *baseBeegfsDriver {
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
		perFSConfigs: perFSConfigs,
		fsIndex:      0,
	}
}

// InitBeegfsDriver returns a pointer to a BeegfsDriver.
func InitBeegfsDriver(perFSConfigs []PerFSConfig) *BeegfsDriver {
	baseDriver := initBaseBeegfsDriver(perFSConfigs)
	baseDriver.driverInfo.FeatureTag = "Dual"
	return &BeegfsDriver{baseDriver}
}

// InitBeegfsStaticDriver returns a pointer to a BeegfsStaticDriver.
func InitBeegfsStaticDriver(perFSConfigs []PerFSConfig) *BeegfsStaticDriver {
	baseDriver := initBaseBeegfsDriver(perFSConfigs)
	baseDriver.driverInfo.FeatureTag = "Static"
	return &BeegfsStaticDriver{baseDriver}
}

// InitBeegfsDynamicDriver returns a pointer to a BeegfsDynamicDriver.
func InitBeegfsDynamicDriver(perFSConfigs []PerFSConfig) *BeegfsDynamicDriver {
	baseDriver := initBaseBeegfsDriver(perFSConfigs)
	baseDriver.driverInfo.FeatureTag = "Dynamic"
	return &BeegfsDynamicDriver{baseDriver}
}

// ---------------------------------------------------------------------------------------------------------------------
// The following functions and methods relate to the pre-provisioned (static) workflow.

// BeegfsDriver implements the testsuites.PreprovisionedVolumeTestDriver interface.
func (d *BeegfsDriver) CreateVolume(config *testsuites.PerTestConfig,
	volumeType testpatterns.TestVolType) testsuites.TestVolume {
	return d.createVolume(config, volumeType)
}

// BeegfsStaticDriver implements the testsuites.PreprovisionedVolumeTestDriver interface.
func (d *BeegfsStaticDriver) CreateVolume(config *testsuites.PerTestConfig,
	volumeType testpatterns.TestVolType) testsuites.TestVolume {
	return d.createVolume(config, volumeType)
}

// createVolume uses OS tools to create a directory on a BeeGFS file system mounted to the testing node. It returns
// a beegfsVolume that can be later used to delete the directory. fsIndex controls which PerFSConfig is used for the
// volume. Kubernetes end-to-end tests do not manipulate fsIndex, so they only use the first PerFSConfig, but
// beegfs-csi-driver specific tests can exert more control.
// NOTE: We use fsIndex in this roundabout way instead of as a function parameter because Kubernetes tests and
// framework functions call CreateVolume with this exact signature. beegfs-csi-driver specific tests can use all of
// the framework functions and built-in tests by FIRST manipulating fsIndex of extraSCParams and SECOND calling the
// functions and tests as normal.
func (d *baseBeegfsDriver) createVolume(config *testsuites.PerTestConfig,
	volumeType testpatterns.TestVolType) beegfsVolume {
	fsConfig := d.GetFSConfig()
	dirName := names.SimpleNameGenerator.GenerateName(config.Framework.Namespace.Name + "-static-")
	volDirPathBeegfsRoot := path.Join(fsConfig.VolDirBasePathBeegfsRoot, dirName)
	volDirPath := path.Join(fsConfig.MountPath, volDirPathBeegfsRoot)
	err := os.Mkdir(volDirPath, 0o755)
	framework.ExpectNoError(err)
	return beegfsVolume{
		volumeID:   beegfs.NewBeegfsUrl(fsConfig.SysMgmtdHost, volDirPathBeegfsRoot),
		volDirPath: volDirPath,
	}
}

// BeegfsDriver implements the testsuites.PreprovisionedPVTestDriver interface.
func (d *BeegfsDriver) GetPersistentVolumeSource(readOnly bool, fsType string,
	testVolume testsuites.TestVolume) (*corev1.PersistentVolumeSource, *corev1.VolumeNodeAffinity) {
	return d.getPersistentVolumeSource(readOnly, fsType, testVolume)
}

// BeegfsStaticDriver implements the testsuites.PreprovisionedPVTestDriver interface.
func (d *BeegfsStaticDriver) GetPersistentVolumeSource(readOnly bool, fsType string,
	testVolume testsuites.TestVolume) (*corev1.PersistentVolumeSource, *corev1.VolumeNodeAffinity) {
	return d.getPersistentVolumeSource(readOnly, fsType, testVolume)
}

// getPersistentVolumeSource returns PersistentVolume source that appropriately references a pre-created directory
// on a BeeGFS file system mounted to the testing node.
func (d *baseBeegfsDriver) getPersistentVolumeSource(readOnly bool, fsType string,
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

// ---------------------------------------------------------------------------------------------------------------------
// The following functions and methods relate to the dynamic workflow.

// BeegfsDriver directly implements the testsuites.DynamicPVTestDriver interface.
func (d *BeegfsDriver) GetDynamicProvisionStorageClass(config *testsuites.PerTestConfig,
	fsType string) *storagev1.StorageClass {
	return d.getDynamicProvisionStorageClass(config, fsType)
}

// BeegfsDynamicDriver implements the testsuites.DynamicPVTestDriver interface.
func (d *BeegfsDynamicDriver) GetDynamicProvisionStorageClass(config *testsuites.PerTestConfig,
	fsType string) *storagev1.StorageClass {
	return d.getDynamicProvisionStorageClass(config, fsType)
}

// getDynamicProvisionStorageClass returns StorageClass source that appropriately references a file system described
// by one of the driver's perFSConfigs. fsIndex controls which PerFSConfig is used for the
// StorageClass and extraSCParams injects additional StorageClass parameters. Kubernetes end-to-end tests do not
// manipulate fsIndex or extraSCParams, so they only use the first PerFSConfig and generate a basic StorageClass, but
// beegfs-csi-driver specific tests can exert more control.
// NOTE: We use fsIndex and extraSCParameters in this roundabout way instead of as a function parameter because
// Kubernetes tests and framework functions call getDynamicProvisionStorageClass with this exact signature.
// beegfs-csi-driver specific tests can use all of the framework functions and built-in tests by FIRST manipulating
// fsIndex of extraSCParams and SECOND calling the functions and tests as normal.
func (d *baseBeegfsDriver) getDynamicProvisionStorageClass(config *testsuites.PerTestConfig,
	fsType string) *storagev1.StorageClass {
	bindingMode := storagev1.VolumeBindingImmediate
	params := map[string]string{
		"sysMgmtdHost":   d.perFSConfigs[d.fsIndex].SysMgmtdHost,
		"volDirBasePath": d.perFSConfigs[d.fsIndex].VolDirBasePathBeegfsRoot,
	}
	if d.extraSCParams != nil {
		for k, v := range d.extraSCParams {
			params[k] = v
		}
	}
	return testsuites.GetStorageClass("beegfs.csi.netapp.com", params, &bindingMode,
		config.Framework.Namespace.Name, "e2e-sc-")
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

// ---------------------------------------------------------------------------------------------------------------------
// The following types, functions, and methods relate to the testsuites.TestVolume interface.

// beegfsVolume implements the testsuites.TestVolume interface.
// The end-to-end Kubernetes tests and various framework functions expect to handle a testsuites.TestVolume that knows
// how to delete itself (out-of-band of a running CSI driver).
type beegfsVolume struct {
	volumeID   string // pkg/beegfs.beegfsVolume.volumeID
	volDirPath string // path to mounted location (for easy deletion)
}

// beegfsVolume implements the testsuites.TestVolume interface.
// When properly set, volDirPath is the path from the root of the testing node to a directory on a mounted BeeGFS file
// system.
func (v beegfsVolume) DeleteVolume() {
	_ = os.RemoveAll(v.volDirPath)
}

// ---------------------------------------------------------------------------------------------------------------------
// The following functions and methods relate allow for the manipulation of a BeegfsDriver in preparation for (or as
// part of the cleanup after) beegfs-csi-driver specific tests.

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

// For now, GetFSConfig returns the PerFSConfig associated with the current fsIndex. External methods may use this
// information to, for example, format beegfs-ctl commands. It would be preferable to factor out any need to return
// a PerFSConfig as beegfs-csi-driver specific tests are developed.
func (d *baseBeegfsDriver) GetFSConfig() PerFSConfig {
	return d.perFSConfigs[d.fsIndex]
}

// SetFSIndexForRDMA looks for an RDMA capable file system and sets fsIndex to refer to the first one it finds. It
// returns false if there are no RDMA capable file systems.
func (d *baseBeegfsDriver) SetFSIndexForRDMA() bool {
	for i, cfg := range d.perFSConfigs {
		if cfg.RDMACapable {
			d.SetFSIndex(i)
			return true
		}
	}
	return false // There are no RDMA capable file systems.
}
