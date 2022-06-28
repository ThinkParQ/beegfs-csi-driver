/*
Copyright 2017 The Kubernetes Authors.

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

package beegfs

import (
	"os"
	"path"

	beegfsv1 "github.com/netapp/beegfs-csi-driver/operator/api/v1"
	"github.com/pkg/errors"
)

const (
	volDirBasePathKey             = "volDirBasePath"
	sysMgmtdHostKey               = "sysMgmtdHost"
	stripePatternStoragePoolIDKey = "stripePattern/storagePoolID"
	stripePatternChunkSizeKey     = "stripePattern/chunkSize"
	stripePatternNumTargetsKey    = "stripePattern/numTargets"
	permissionsUIDKey             = "permissions/uid"
	permissionsGIDKey             = "permissions/gid"
	permissionsModeKey            = "permissions/mode"
	defaultPermissionsMode        = 0o0777

	logLevelDebug   = 3 // This log level is used for most informational logs in RPCs and GRPC calls
	logLevelVerbose = 5 // This log level is used for only very repetitive logs such as the Probe GRPC call
)

type beegfs struct {
	driverName             string
	nodeID                 string
	version                string
	endpoint               string
	pluginConfig           beegfsv1.PluginConfig
	clientConfTemplatePath string
	csDataDir              string // directory controller service uses to create BeeGFS config files and mount file systems

	ids *identityServer
	ns  *nodeServer
	cs  *controllerServer
}

// beegfsVolume contains any distinguishing information about a BeeGFS "volume" (directory) and its parent BeeGFS file
// system that may be required by an RPC call. Not all RPC calls require all parameters, but beegfsVolumes should be
// constructed with all parameters to eliminate the need to check whether a parameter has been set. All paths are
// absolute but are rooted from either the host or BeeGFS. Path variables rooted from the host have the suffix Path.
// Path variables rooted from BeeGFS have the suffix PathBeegfsRoot.
//
// From the host's perspective (file or directory names in "") (all variable names represent absolute paths):
//    /
//    |-- ...
//        |-- mountDirPath
//            |-- "beegfs-client.conf" (clientConfPath)
//            |-- "connInterfacesFile"
//            |-- "connNetFilterFile"
//            |-- "connRDMAInterfacesFile"
//            |-- "connTcpOnlyFilterFile"
//            |-- "mount" (mountPath)
//                |-- ...
//                    |-- ".csi"
//                        |-- volumes
//                            |-- csiDirPath
//                    |-- volDirBasePath
//                        |-- volDirPath (same as volDirPathBeegfsRoot)
//
// From the perspective of the BeeGFS file system (all variable names represent absolute paths):
//    /
//    |-- ...
//        |-- volDirBasePathBeegfsRoot (same as volDirBasePath)
//            |-- ".csi"
//                |-- volumes
//                    |-- csiDirPathBeegfsRoot (same as csiDirPath)
//            |-- volDirPathBeegfsRoot (same as volDirPath)
type beegfsVolume struct {
	config                   beegfsv1.BeegfsConfig
	clientConfPath           string // absolute path to beegfs-client.conf from host root (e.g. /.../mountDirPath/beegfs-client.conf)
	csiDirPath               string // absolute path to CSI metadata directory from host root (e.g. /.../mountDirPath/mount/.../parent/.csi/volumes/volume)
	csiDirPathBeegfsRoot     string // absolute path to CSI metadata directory from BeeGFS root (e.g. /.../parent/.csi/volumes/volume)
	mountDirPath             string // absolute path to directory containing configuration files and mount point from node root (e.g. /.../mountDirPath)
	mountPath                string // absolute path to mount point from host root (e.g. /.../mountDirPath/mount)
	sysMgmtdHost             string // IP address or hostname of BeeGFS mgmtd service
	volDirBasePathBeegfsRoot string // absolute path to BeeGFS parent directory from BeeGFS root (e.g. /.../parent)
	volDirBasePath           string // absolute path to BeeGFS parent directory from host root (e.g. /.../mountDirPath/mount/.../parent)
	volDirPathBeegfsRoot     string // absolute path to BeeGFS directory from BeeGFS root (e.g. /.../parent/volume)
	volDirPath               string // absolute path to BeeGFS directory from host root (e.g. /.../mountDirPath/mount/.../parent/volume)
	volumeID                 string // like beegfs://sysMgmtdHost/volDirPathBeegfsRoot
}

// stripePatternConfig contains our internal representation of all CreateVolume parameters (StorageClass parameters in
// K8s) that should be prefaced with stripePattern/.
type stripePatternConfig struct {
	storagePoolID           string
	stripePatternChunkSize  string
	stripePatternNumTargets string
}

// permissionsConfig contains our internal representation of all CreateVolume parameters (StorageClass parameters in
// K8s) that should be prefaced with permissions/. We expect to receive mode as a three or four digit octal literal in
// typical Unix fashion and store it as a uint16 for easy output in this same format.
type permissionsConfig struct {
	uid  uint32 // The majority of UNIX-like systems support 32-bit UIDs.
	gid  uint32 // The majority of UNIX-like systems support 32-bit GIDs.
	mode uint16 // A full access mode consists of four base-8 digits (12 bits).
}

// reqParameters contains all possible parameters from CreateVolumeRequest.parameters or ValidateVolumeCapabilitiesRequest.parameters.
// We utilize the reqParameters struct with ValidateReqParameters to validate all parameters passed, ensuring correct parameters
// when called by by CreateVolume and ValidateVolumeCapabilities.
type reqParameters struct {
	sysMgmtdHost             string
	volDirBasePathBeegfsRoot string
	volStripePatternConfig   stripePatternConfig
	volPermissionsConfig     permissionsConfig
}

// hasNonDefaultOwnerOrGroup returns true if either uid or gid are not 0 and false otherwise.
func (cfg permissionsConfig) hasNonDefaultOwnerOrGroup() bool { return cfg.uid > 0 || cfg.gid > 0 }

// hasSpecialPermissions returns true if the sticky bit, setgid bit, or setuid bit are set (i.e. if the integer value
// of mode is greater than 0o0777 or 511).
func (cfg permissionsConfig) hasSpecialPermissions() bool {
	// A non-zero first octal digit represents special permissions.
	return cfg.mode > 0o0777
}

// We store mode as a uint16, but the os package requires an os.FileMode for some functions. goFileMode returns a
// correct os.FileMode representation of the stored mode.
func (cfg permissionsConfig) goFileMode() os.FileMode {
	// os.FileMode doesn't represent special permissions using the same bits as Unix.
	// Extract the normal permissions and add in the special permissions as a separate sequence of steps.
	goMode := os.FileMode(cfg.mode & 0o777)
	stickyBit := (cfg.mode & 0o1000) > 0 // The Unix sticky bit is the 10th most significant bit.
	setgidBit := (cfg.mode & 0o2000) > 0 // The Unix setgid bit is the 11th most significant bit.
	setuidBit := (cfg.mode & 0o4000) > 0 // The Unix setuid bit is the 12th most significant bit.
	if stickyBit {
		goMode = goMode | os.ModeSticky
	}
	if setgidBit {
		goMode = goMode | os.ModeSetgid
	}
	if setuidBit {
		goMode = goMode | os.ModeSetuid
	}
	return goMode
}

var (
	vendorVersion = "dev"
)

// NewBeegfsDriver initializes a working BeegfsDriver.
func NewBeegfsDriver(connAuthPath, configPath, csDataDir, driverName, endpoint, nodeID, clientConfTemplatePath,
	version string, nodeUnstageTimeout uint64) (*beegfs, error) {

	if err := verifyBeegfsClientModuleIsAvailable(); err != nil {
		return nil, err
	}

	driver, err := newBeegfsDriver(connAuthPath, configPath, csDataDir, driverName, endpoint, nodeID,
		clientConfTemplatePath, version, nodeUnstageTimeout)
	if err != nil {
		return nil, err
	}

	// Create complex GRPC servers.
	if driver.ns, err = newNodeServer(driver.nodeID, driver.pluginConfig, driver.clientConfTemplatePath); err != nil {
		return nil, err
	}
	if driver.cs, err = newControllerServer(driver.nodeID, driver.pluginConfig, driver.clientConfTemplatePath,
		driver.csDataDir, nodeUnstageTimeout); err != nil {
		return nil, err
	}

	return driver, nil
}

// NewBeegfsDriverSanity initializes a BeegfsDriver that doesn't have a working mounter or beegfs-ctl execution
// capabilities. This BeegfsDriver can be used for sanity testing on any machine.
func NewBeegfsDriverSanity(connAuthPath, configPath, csDataDir, driverName, endpoint, nodeID, clientConfTemplatePath,
	version string, nodeUnstageTimeout uint64) (*beegfs, error) {
	driver, err := newBeegfsDriver(connAuthPath, configPath, csDataDir, driverName, endpoint, nodeID,
		clientConfTemplatePath, version, nodeUnstageTimeout)
	if err != nil {
		return nil, err
	}

	// Create complex GRPC servers.
	driver.ns = newNodeServerSanity(driver.nodeID, driver.pluginConfig, driver.clientConfTemplatePath)
	driver.cs = newControllerServerSanity(driver.nodeID, driver.pluginConfig, driver.clientConfTemplatePath,
		driver.csDataDir, nodeUnstageTimeout)

	return driver, nil
}

// newBeegfsDriver is used by both NewBeegfsDriver and NewBeegfsDriverSanity for common initialization.
func newBeegfsDriver(connAuthPath, configPath, csDataDir, driverName, endpoint, nodeID, clientConfTemplatePath,
	version string, nodeUnstageTimeout uint64) (*beegfs, error) {
	if driverName == "" {
		return nil, errors.New("no driver name provided")
	}

	if nodeID == "" {
		return nil, errors.New("no node id provided")
	}

	if endpoint == "" {
		return nil, errors.New("no driver endpoint provided")
	}

	if version != "" {
		vendorVersion = version
	}

	var err error
	if clientConfTemplatePath != "" {
		if _, err := fsutil.ReadFile(clientConfTemplatePath); err != nil {
			return nil, errors.WithMessage(err, "failed to read client configuration template file")
		}
	} else if clientConfTemplatePath = getDefaultClientConfTemplatePath(); clientConfTemplatePath == "" {
		return nil, errors.New("failed to get valid default client configuration template file")
	}

	var pluginConfig beegfsv1.PluginConfig
	if configPath != "" {
		if pluginConfig, err = parseConfigFromFile(configPath, nodeID); err != nil {
			return nil, errors.WithMessage(err, "failed to handle configuration file")
		}
	}

	if connAuthPath != "" {
		if err = parseConnAuthFromFile(connAuthPath, &pluginConfig); err != nil {
			return nil, errors.WithMessage(err, "failed to handle connAuth file")
		}
	}

	if csDataDir == "" {
		return nil, errors.New("no controller service data directory path provided")
	} else if err := fs.MkdirAll(csDataDir, 0750); err != nil {
		return nil, errors.Wrap(err, "failed to create csDataDir")
	}

	logger(nil).Info("Driver initializing", "driverName", driverName, "version", vendorVersion)

	driver := beegfs{
		driverName:             driverName,
		version:                vendorVersion,
		nodeID:                 nodeID,
		endpoint:               endpoint,
		pluginConfig:           pluginConfig,
		clientConfTemplatePath: clientConfTemplatePath,
		csDataDir:              csDataDir,
	}

	// Create simple gRPC identity server.
	driver.ids = newIdentityServer(driver.driverName, driver.version)

	return &driver, nil
}

func (b *beegfs) Run() {
	s := newNonBlockingGRPCServer()
	s.Start(b.endpoint, b.ids, b.cs, b.ns)
	s.Wait()
}

// newBeeGFSVolume creates a beegfsVolume from parameters.
func newBeegfsVolume(mountDirPath, sysMgmtdHost, volDirPathBeegfsRoot string, pluginConfig beegfsv1.PluginConfig) beegfsVolume {
	// These parameters must be constructed outside of the struct literal.
	mountPath := path.Join(mountDirPath, "mount")
	volDirPath := path.Join(mountPath, volDirPathBeegfsRoot)
	volDirBasePath := path.Dir(volDirPath)
	volDirBasePathBeegfsRoot := path.Dir(volDirPathBeegfsRoot)
	volName := path.Base(volDirPathBeegfsRoot) // volName is always the last element of volDirPathBeegfsRoot.

	return beegfsVolume{
		config:                   squashConfigForSysMgmtdHost(sysMgmtdHost, pluginConfig),
		clientConfPath:           path.Join(mountDirPath, "beegfs-client.conf"),
		csiDirPath:               path.Join(volDirBasePath, ".csi", "volumes", volName),
		csiDirPathBeegfsRoot:     path.Join(volDirBasePathBeegfsRoot, ".csi", "volumes", volName),
		mountDirPath:             mountDirPath,
		mountPath:                mountPath,
		sysMgmtdHost:             sysMgmtdHost,
		volDirBasePathBeegfsRoot: volDirBasePathBeegfsRoot,
		volDirBasePath:           volDirBasePath,
		volDirPathBeegfsRoot:     volDirPathBeegfsRoot,
		volDirPath:               volDirPath,
		volumeID:                 NewBeegfsURL(sysMgmtdHost, volDirPathBeegfsRoot),
	}
}

// newBeeGFSVolume creates a beegfsVolume from a volumeID.
func newBeegfsVolumeFromID(mountDirPath, volumeID string, pluginConfig beegfsv1.PluginConfig) (beegfsVolume, error) {
	sysMgmtdHost, volDirPathBeegfsRoot, err := parseBeegfsURL(volumeID)
	if err != nil {
		return beegfsVolume{}, err
	}
	return newBeegfsVolume(mountDirPath, sysMgmtdHost, volDirPathBeegfsRoot, pluginConfig), nil
}

// getDefaultClientConfTemplatePath looks for a beegfs-client.conf file in an ordered slice of default paths. It
// returns the first valid path it finds or an empty string if none of the default paths are valid.
func getDefaultClientConfTemplatePath() string {
	defaultPaths := []string{
		// Default beegfs-client.conf install location.
		"/etc/beegfs/beegfs-client.conf",
		// Default beegfs-client.conf install location inside CSI container.
		"/host/etc/beegfs/beegfs-client.conf",
		// Client files subdirectory of plugin data directory.
		"/var/lib/kubelet/plugins/beegfs.csi.netapp.com/client/beegfs-client.conf",
		// Client files subdirectory of plugin data directory inside CSI container.
		"/host/var/lib/kubelet/plugins/beegfs.csi.netapp.com/client/beegfs-client.conf",
	}
	for _, path := range defaultPaths {
		if _, err := fsutil.ReadFile(path); err == nil {
			return path
		}
	}
	// None of the default paths pointed to a readable file.
	return ""
}
