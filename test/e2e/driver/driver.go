/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package driver

import (
	"path"

	"github.com/netapp/beegfs-csi-driver/pkg/beegfs"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	e2eframework "k8s.io/kubernetes/test/e2e/framework"
	e2eskipper "k8s.io/kubernetes/test/e2e/framework/skipper"
	storageframework "k8s.io/kubernetes/test/e2e/storage/framework"
)

// Verify expected interfaces are properly implemented at compile time.
var _ storageframework.TestDriver = &BeegfsDriver{}
var _ storageframework.TestDriver = &BeegfsDynamicDriver{}
var _ storageframework.DynamicPVTestDriver = &BeegfsDriver{}
var _ storageframework.DynamicPVTestDriver = &BeegfsDynamicDriver{}
var _ storageframework.PreprovisionedVolumeTestDriver = &BeegfsDriver{}

// baseBeegfsDriver is unexported and cannot be directly accessed or instantiated. All exported drivers use it as
// their underlying data structure and can call its internal methods.
type baseBeegfsDriver struct {
	driverInfo                      storageframework.DriverInfo
	perFSConfigs                    []beegfs.FileSystemSpecificConfig
	fsIndex                         int
	extraSCParams                   map[string]string
	dynamicVolDirBasePathBeegfsRoot string // Set once on initialization (e.g. /e2e-test/dynamic).
	staticVolDirPathBeegfsRoot      string // Set once on initialization (e.g. /e2e-test/static/static1).
}

// BeegfsDriver is an exported driver that implements the storageframework.TestDriver,
// storageframework.DynamicPVTestDriver, storageframework.PreprovisionedVolumeTestDriver, and
// storageframework.PreprovisionedPVTestDriver interfaces. It is intended to be used in all beegfs-csi-driver specific
//tests.
type BeegfsDriver struct {
	*baseBeegfsDriver
}

// BeegfsDynamicDriver is an exported driver that implements the storageframework.TestDriver and
// storageframework.DynamicPVTestDriver interfaces. It intentionally does not implement the
// storageframework.PreprovisionedVolumeTestDriver and storageframework.PreprovisionedPVTestDriver interfaces. It is
// intended to be used for K8s built-in tests, which may use the pre-provisioned interface in unanticipated ways if
// allowed.
type BeegfsDynamicDriver struct {
	*baseBeegfsDriver
}

// baseBeegfsDriver implements the storageframework.TestDriver interface.
func (d *baseBeegfsDriver) GetDriverInfo() *storageframework.DriverInfo {
	return &d.driverInfo
}

// baseBeegfsDriver implements the storageframework.TestDriver interface.
func (d *baseBeegfsDriver) SkipUnsupportedTest(pattern storageframework.TestPattern) {
	if pattern.VolType == storageframework.PreprovisionedPV && d.staticVolDirPathBeegfsRoot == "" {
		e2eskipper.Skipf("Set staticVolDirPathBeegfsRoot to enable pre-provisioned tests -- skipping")
	}
}

// baseBeegfsDriver implements the storageframework.TestDriver interface.
func (d *baseBeegfsDriver) PrepareTest(f *e2eframework.Framework) (*storageframework.PerTestConfig, func()) {
	config := &storageframework.PerTestConfig{
		Driver:    d,
		Prefix:    "beegfs",
		Framework: f,
	}
	return config, func() {}
}

// initBaseBeegfsDriver handles basic initialization shared across all exported drivers.
func initBaseBeegfsDriver(dynamicVolDirBasePathBeegfsRoot, staticVolDirPathBeegfsRoot string) *baseBeegfsDriver {
	return &baseBeegfsDriver{
		driverInfo: storageframework.DriverInfo{
			Name: "beegfs",
			// FeatureTag:
			// MaxFileSize:
			// SupportedSizeRange:
			SupportedFsType: sets.NewString(""),
			// Map of string for supported mount option
			// TODO(jbostian, A141): Determine how to implement mount options.
			// SupportedMountOption:
			// RequiredMountOption:
			Capabilities: map[storageframework.Capability]bool{
				storageframework.CapPersistence:         true,
				storageframework.CapBlock:               false,
				storageframework.CapFsGroup:             false,
				storageframework.CapExec:                true,
				storageframework.CapSnapshotDataSource:  false,
				storageframework.CapPVCDataSource:       false,
				storageframework.CapMultiPODs:           true,
				storageframework.CapRWX:                 true,
				storageframework.CapControllerExpansion: false,
				storageframework.CapNodeExpansion:       false,
				storageframework.CapVolumeLimits:        false,
				// This setting is only used in two places, both in the multivolume test suite. Setting this to true
				// signals that we *only* support single-node volumes and should skip tests that require a volume to
				// be shared between two different nodes.
				storageframework.CapSingleNodeVolume: false,
				storageframework.CapTopology:         false,
			},
			// RequiredAccessModes:
			// TopologyKeys:
			// NumAllowedTopologies:
			StressTestOptions: &storageframework.StressTestOptions{
				NumPods:     10,
				NumRestarts: 3,
			},
			// VolumeSnapshotStressTestOptions:
		},
		perFSConfigs:                    make([]beegfs.FileSystemSpecificConfig, 0),
		fsIndex:                         0,
		dynamicVolDirBasePathBeegfsRoot: dynamicVolDirBasePathBeegfsRoot,
		staticVolDirPathBeegfsRoot:      staticVolDirPathBeegfsRoot,
	}
}

// InitBeegfsDriver returns a pointer to a BeegfsDriver.
func InitBeegfsDriver(dynamicVolDirBasePathBeegfsRoot, staticVolDirBasePathBeegfsRoot,
	staticVolDirName string) *BeegfsDriver {
	staticVolDirPathBeegfsRoot := ""
	if staticVolDirBasePathBeegfsRoot != "" && staticVolDirName != "" {
		staticVolDirPathBeegfsRoot = path.Join(staticVolDirBasePathBeegfsRoot, staticVolDirName)
	}
	return &BeegfsDriver{baseBeegfsDriver: initBaseBeegfsDriver(dynamicVolDirBasePathBeegfsRoot,
		staticVolDirPathBeegfsRoot)}
}

// InitBeegfsDynamicDriver returns a pointer to a BeegfsDynamicDriver.
func InitBeegfsDynamicDriver(dynamicVolDirBasePathBeegfsRoot string) *BeegfsDynamicDriver {
	return &BeegfsDynamicDriver{baseBeegfsDriver: initBaseBeegfsDriver(dynamicVolDirBasePathBeegfsRoot, "")}
}

// baseBeegfsDriver directly implements the storageframework.DynamicPVTestDriver interface.
func (d *baseBeegfsDriver) GetDynamicProvisionStorageClass(config *storageframework.PerTestConfig,
	fsType string) *storagev1.StorageClass {
	params := map[string]string{
		"sysMgmtdHost":   d.perFSConfigs[d.fsIndex].SysMgmtdHost,
		"volDirBasePath": d.dynamicVolDirBasePathBeegfsRoot,
	}
	if d.extraSCParams != nil {
		for k, v := range d.extraSCParams {
			params[k] = v
		}
	}
	// Do not explicitly set volumeBindingMode. By default, this results in VolumeBindingImmediate behavior. Framework
	// functions overwrite volumeBindingMode from storageframework.TestPattern as needed.
	return storageframework.GetStorageClass("beegfs.csi.netapp.com", params, nil, config.Framework.Namespace.Name)
}

// BeegfsDriver implements the storageframework.PreprovisionedVolumeTestDriver interface.
// CreateVolume returns a storageframework.TestVolume that appropriately references a pre-created directory on a
// BeeGFS file system known to the driver. Tests can use SetFSIndex and SetStaticDirName to modify its behavior.
func (d *BeegfsDriver) CreateVolume(config *storageframework.PerTestConfig, volumeType storageframework.TestVolType) storageframework.TestVolume {
	fsConfig := d.perFSConfigs[d.fsIndex]
	return beegfsVolume{
		volumeID: beegfs.NewBeegfsUrl(fsConfig.SysMgmtdHost, d.staticVolDirPathBeegfsRoot),
	}
}

// BeegfsDriver implements the storageframework.PreprovisionedPVTestDriver interface.
// GetPersistentVolumeSource returns a PersistentVolumeSource that appropriately references a pre-created directory
// on a BeeGFS file system known to the driver.
func (d *BeegfsDriver) GetPersistentVolumeSource(readOnly bool, fsType string,
	testVolume storageframework.TestVolume) (*corev1.PersistentVolumeSource, *corev1.VolumeNodeAffinity) {
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

// beegfsVolume implements the storageframework.TestVolume interface.
// The end-to-end Kubernetes tests and various framework functions expect to handle a storageframework.TestVolume that
// knows how to delete itself (out-of-band of a running CSI driver).
type beegfsVolume struct {
	volumeID string // pkg/beegfs.beegfsVolume.volumeID
}

// beegfsVolume implements the storageframework.TestVolume interface.
// We don't actually do anything when DeleteVolume() is called.
func (v beegfsVolume) DeleteVolume() {
	// Intentionally empty.
	// Our pre-provisioned volumes are not created on demand and are not deleted at the end of a test.
}
