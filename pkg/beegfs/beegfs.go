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

package beegfs

import (
	"path"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/utils/mount"
)

const (
	volDirBasePathKey          = "volDirBasePath"
	sysMgmtdHostKey            = "sysMgmtdHost"
	storagePoolIDKey           = "stripePattern/storagePoolID"
	stripePatternChunkSizeKey  = "stripePattern/chunkSize"
	stripePatternNumTargetsKey = "stripePattern/numTargets"

	LogDebug   = glog.Level(3) // This log level is used for most informational logs in RPCs and GRPC calls
	LogVerbose = glog.Level(5) // This log level is used for only very repetitive logs such as the Probe GRPC call
)

type beegfs struct {
	driverName             string
	nodeID                 string
	version                string
	endpoint               string
	pluginConfig           pluginConfig
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
//            |-- "connTcpOnlyFilterFile"
//            |-- "mount" (mountPath)
//                |-- ...
//                    |-- volDirBasePath
//                        |-- volDirPath (same as volDirPathBeegfsRoot)
//
// From the perspective of the BeeGFS file system (all variable names represent absolute paths):
//    /
//    |-- ...
//        |-- volDirBasePathBeegfsRoot
//            |-- volDirPathBeegfsRoot (same as volDirPath)
type beegfsVolume struct {
	config                   beegfsConfig
	clientConfPath           string // absolute path to beegfs-client.conf from host root (e.g. .../mountDirPath/beegfs-client.conf)
	mountDirPath             string // absolute path to directory containing configuration files and mount point from node root
	mountPath                string // absolute path to mount point from host root (e.g. .../mountDirPath/mount)
	sysMgmtdHost             string // IP address or hostname of BeeGFS mgmtd service
	volDirBasePathBeegfsRoot string // absolute path to BeeGFS parent directory from BeeGFS root (e.g. /parent)
	volDirBasePath           string // absolute path to BeeGFS parent directory from host root (e.g. ../mountDirPath/mount/parent)
	volDirPathBeegfsRoot     string // absolute path to BeeGFS directory from BeeGFS root (e.g. /parent/volume)
	volDirPath               string // absolute path to BeeGFS directory from host root (e.g. .../mountDirPath/mount/parent/volume)
	volumeID                 string // like beegfs://sysMgmtdHost/volDirPathBeegfsRoot
}

type stripePatternConfig struct {
	storagePoolID           string
	stripePatternChunkSize  string
	stripePatternNumTargets string
}

var (
	vendorVersion = "dev"
)

func NewBeegfsDriver(configPath, csDataDir, driverName, endpoint, nodeID, clientConfTemplatePath, version string) (*beegfs, error) {
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

	var pluginConfig pluginConfig
	if configPath != "" {
		var err error
		if pluginConfig, err = parseConfigFromFile(configPath, nodeID); err != nil {
			return nil, errors.WithMessage(err, "failed to handle configuration file")
		}
	}

	if err := fs.MkdirAll(csDataDir, 0750); err != nil {
		return nil, errors.Wrap(err, "failed to create csDataDir")
	}

	glog.Infof("Driver: %v ", driverName)
	glog.Infof("Version: %s", vendorVersion)

	var driver beegfs
	driver = beegfs{
		driverName:             driverName,
		version:                vendorVersion,
		nodeID:                 nodeID,
		endpoint:               endpoint,
		pluginConfig:           pluginConfig,
		clientConfTemplatePath: clientConfTemplatePath,
		csDataDir:              csDataDir,
	}

	// Create GRPC servers
	driver.ids = NewIdentityServer(driver.driverName, driver.version)
	driver.ns = NewNodeServer(driver.nodeID, driver.pluginConfig, driver.clientConfTemplatePath)
	driver.cs = NewControllerServer(driver.nodeID, driver.pluginConfig, driver.clientConfTemplatePath, driver.csDataDir)

	return &driver, nil
}

func (b *beegfs) Run() {
	if b.cs.mounter == nil {
		b.cs.mounter = mount.New("")
	}
	if b.ns.mounter == nil {
		b.ns.mounter = mount.New("")
	}

	s := NewNonBlockingGRPCServer()
	s.Start(b.endpoint, b.ids, b.cs, b.ns)
	s.Wait()
}

// newBeeGFSVolume creates a beegfsVolume from parameters.
func newBeegfsVolume(mountDirPath, sysMgmtdHost, volDirPathBeegfsRoot string, pluginConfig pluginConfig) beegfsVolume {
	// These parameters must be constructed outside of the struct literal.
	mountPath := path.Join(mountDirPath, "mount")
	volDirPath := path.Join(mountPath, volDirPathBeegfsRoot)

	return beegfsVolume{
		config:                   squashConfigForSysMgmtdHost(sysMgmtdHost, pluginConfig),
		clientConfPath:           path.Join(mountDirPath, "beegfs-client.conf"),
		mountDirPath:             mountDirPath,
		mountPath:                mountPath,
		sysMgmtdHost:             sysMgmtdHost,
		volDirBasePathBeegfsRoot: path.Dir(volDirPathBeegfsRoot),
		volDirBasePath:           path.Dir(volDirPath),
		volDirPathBeegfsRoot:     volDirPathBeegfsRoot,
		volDirPath:               volDirPath,
		volumeID:                 newBeegfsUrl(sysMgmtdHost, volDirPathBeegfsRoot),
	}
}

// newBeeGFSVolume creates a beegfsVolume from a volumeID.
func newBeegfsVolumeFromID(mountDirPath, volumeID string, pluginConfig pluginConfig) (beegfsVolume, error) {
	sysMgmtdHost, volDirPathBeegfsRoot, err := parseBeegfsUrl(volumeID)
	if err != nil {
		return beegfsVolume{}, err
	}
	return newBeegfsVolume(mountDirPath, sysMgmtdHost, volDirPathBeegfsRoot, pluginConfig), nil
}
