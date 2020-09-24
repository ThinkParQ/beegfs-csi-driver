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

	//	"io"
	//	"io/ioutil"
	"os"
	"path/filepath"

	//	"strings"

	"github.com/golang/glog"
	//	"google.golang.org/grpc/codes"
	//	"google.golang.org/grpc/status"
	//	"k8s.io/kubernetes/pkg/volume/util/volumepathhandler"
	//	utilexec "k8s.io/utils/exec"

	//	timestamp "github.com/golang/protobuf/ptypes/timestamp"

	// todo(eastburj): change module name to the correct github repository and consider supporting multiple go.mod, remove hostpath from the project, or update all corresponding imports in hostpath
	common "github.com/kubernetes-csi/csi-driver-host-path/pkg/common"
)

type beegfs struct {
	name              string
	nodeID            string
	version           string
	endpoint          string
	ephemeral         bool
	maxVolumesPerNode int64

	ids *identityServer
	ns  *nodeServer
	cs  *controllerServer
}

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

const (
	// Directory where data for volumes and snapshots are persisted.
	// This can be ephemeral within the container or persisted if
	// backed by a Pod volume.
	dataRoot = "/csi-data-dir"
)

func init() {
	beegfsVolumes = map[string]beegfsVolume{}
	// todo(eastburj): load beegfsVolumes from a persistent location (in case the process restarts)
}

func NewBeegfsDriver(driverName, nodeID, endpoint string, ephemeral bool, maxVolumesPerNode int64, version string) (*beegfs, error) {
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

	if err := os.MkdirAll(dataRoot, 0750); err != nil {
		return nil, fmt.Errorf("failed to create dataRoot: %v", err)
	}

	glog.Infof("Driver: %v ", driverName)
	glog.Infof("Version: %s", vendorVersion)

	return &beegfs{
		name:              driverName,
		version:           vendorVersion,
		nodeID:            nodeID,
		endpoint:          endpoint,
		ephemeral:         ephemeral,
		maxVolumesPerNode: maxVolumesPerNode,
	}, nil
}

func (b *beegfs) Run() {
	// Create GRPC servers
	b.ids = NewIdentityServer(b.name, b.version)
	b.ns = NewNodeServer(b.nodeID, b.ephemeral, b.maxVolumesPerNode)
	b.cs = NewControllerServer(b.ephemeral, b.nodeID)

	s := common.NewNonBlockingGRPCServer()
	s.Start(b.endpoint, b.ids, b.cs, b.ns)
	s.Wait()
}

func getVolumeByID(volumeID string) (beegfsVolume, error) {
	if beegfsVol, ok := beegfsVolumes[volumeID]; ok {
		return beegfsVol, nil
	}
	return beegfsVolume{}, fmt.Errorf("volume id %s does not exist in the volumes list", volumeID)
}

func getVolumeByName(volName string) (beegfsVolume, error) {
	for _, beegfsVol := range beegfsVolumes {
		if beegfsVol.VolName == volName {
			return beegfsVol, nil
		}
	}
	return beegfsVolume{}, fmt.Errorf("volume name %s does not exist in the volumes list", volName)
}

// getVolumePath returns the canonical path for beegfs volume
func getVolumePath(volID string) string {
	return filepath.Join(dataRoot, volID)
}

// updateVolume updates the existing beegfs volume.
func updateBeegfsVolume(volID string, volume beegfsVolume) error {
	glog.V(4).Infof("updating beegfs volume: %s", volID)

	if _, err := getVolumeByID(volID); err != nil {
		return err
	}

	// todo(eastburj): persist volume updates (in case the process restarts)
	beegfsVolumes[volID] = volume
	return nil
}
