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
	"errors"
	"fmt"
	"github.com/golang/glog"
)

const (
	volDirBasePathKey = "volDirBasePath"
	sysMgmtdHostKey   = "sysMgmtdHost"
)

type beegfs struct {
	name              string
	nodeID            string
	version           string
	endpoint          string
	ephemeral         bool
	maxVolumesPerNode int64
	pluginConfig      pluginConfig
	confTemplatePath  string
	csDataDir         string // directory controller service uses to create BeeGFS config files and mount file systems

	ids *identityServer
	ns  *nodeServer
	cs  *controllerServer
}

// TODO(webere): Determine whether or not we can throw this away.
type beegfsVolume struct {
	VolName string `json:"volName"`
	VolID   string `json:"volID"`
	VolSize int64  `json:"volSize"`
	VolPath string `json:"volPath"`
	//	VolAccessType accessType `json:"volAccessType"`
	ParentVolID  string `json:"parentVolID,omitempty"`
	ParentSnapID string `json:"parentSnapID,omitempty"`
	Ephemeral    bool   `json:"ephemeral"`
}

var (
	vendorVersion = "dev"

	beegfsVolumes map[string]beegfsVolume
)

func init() {
	beegfsVolumes = map[string]beegfsVolume{}
	// todo(eastburj): load beegfsVolumes from a persistent location (in case the process restarts)
}

func NewBeegfsDriver(driverName, nodeID, endpoint, configFile, templateClientConfFile, csDataDir, version string,
	ephemeral bool, maxVolumesPerNode int64) (*beegfs, error) {
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
	if configFile != "" {
		var err error
		pluginConfig, err = parseConfigFromFile(configFile, nodeID)
		if err != nil {
			return nil, fmt.Errorf("failed to handle configuration file: %v", err)
		}
	}

	if err := fs.MkdirAll(csDataDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create csDataDir: %v", err)
	}

	glog.Infof("Driver: %v ", driverName)
	glog.Infof("Version: %s", vendorVersion)

	var driver beegfs
	driver = beegfs{
		name:              driverName,
		version:           vendorVersion,
		nodeID:            nodeID,
		endpoint:          endpoint,
		ephemeral:         ephemeral,
		maxVolumesPerNode: maxVolumesPerNode,
		pluginConfig:      pluginConfig,
		confTemplatePath:  templateClientConfFile,
		csDataDir:         csDataDir,
	}

	// Create GRPC servers
	driver.ids = NewIdentityServer(driver.name, driver.version)
	driver.ns = NewNodeServer(driver.nodeID, driver.ephemeral, driver.maxVolumesPerNode, driver.pluginConfig, driver.confTemplatePath)
	driver.cs = NewControllerServer(driver.ephemeral, driver.nodeID, driver.pluginConfig, driver.confTemplatePath, driver.csDataDir)

	return &driver, nil
}

func (b *beegfs) Run() {
	s := NewNonBlockingGRPCServer()
	s.Start(b.endpoint, b.ids, b.cs, b.ns)
	s.Wait()
}

func getVolumeByID(volumeID string) (beegfsVolume, error) {
	if beegfsVol, ok := beegfsVolumes[volumeID]; ok {
		return beegfsVol, nil
	}
	return beegfsVolume{}, fmt.Errorf("volume id %s does not exist in the volumes list", volumeID)
}
