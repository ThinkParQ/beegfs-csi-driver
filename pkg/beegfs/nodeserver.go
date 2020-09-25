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
	//	"fmt"
	//	"os"
	//	"strings"
	//
	//	"github.com/golang/glog"

	"os"
	"path"
	"strings"
	"golang.org/x/net/context"

	//
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	//	"k8s.io/kubernetes/pkg/util/mount"
	//	"k8s.io/kubernetes/pkg/volume/util/volumepathhandler"
)

const TopologyKeyNode = "topology.hostpath.csi/node"

var (
	nodeCaps = []csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
	}
)

type nodeServer struct {
	nodeID            string
	ephemeral         bool
	maxVolumesPerNode int64
}

func NewNodeServer(nodeId string, ephemeral bool, maxVolumesPerNode int64) *nodeServer {
	return &nodeServer{
		nodeID:            nodeId,
		ephemeral:         ephemeral,
		maxVolumesPerNode: maxVolumesPerNode,
	}
}

func (ns *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (ns *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (ns *nodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	// Check arguments
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if len(req.GetStagingTargetPath()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}
	if req.GetVolumeCapability() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume Capability missing in request")
	}

	// (jmccormi) TODO: Check and return all possible NodeStageVolume errors:
	//	https://github.com/container-storage-interface/spec/blob/master/spec.md#nodestagevolume-errors

	/* (jmccormi) the full volumeStagingTargetPath within the local root filesystem for each BeeGFS volume is determined as follows:
		staging_target_path/
			sysMgmtdHost_beegfs_vols/ <-- Replacing . with _ if provided an IP address for sysMgmtdHost.
				volPath/			  <-- The full path to the requested directory within the BeeGFS instance.
					sysMgmtdHost_beegfs/ <-- The actual BeeGFS mount point for this volume will be created here. 
					sysMgmtdHost_beegfs-client.conf <-- A corresponding BeeGFS client config file will be created here.
		=== Example ===
		/mnt/
			10_113_72_217_beegfs_vols/
				jmccormi_scratch/jmccormi_test_1/
					10_113_72_217_beegfs/
					10_113_72_217_beegfs-client.conf
	*/

	sysMgmtdHost, volPath, err := parseBeegfsUrl(req.GetVolumeId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	volumeStagingTargetPath := path.Join(req.GetStagingTargetPath(), strings.Replace(sysMgmtdHost, ".", "_", 3)+"_beegfs_vols", volPath)

	err = os.MkdirAll(path.Join(volumeStagingTargetPath), 0755)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s\n occured creating volume staging target path %s for volume ID %s", err, volumeStagingTargetPath, req.GetVolumeId())
	}

	//Ensure a BeeGFS client configuration file exists for this volume:
	requestedConfPath, _, err := generateBeeGFSClientConf(req.VolumeContext, volumeStagingTargetPath, true)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "%s\noccured generating a BeeGFS client conf file for %s at %s", err, req.VolumeContext, volumeStagingTargetPath)
	}

	// Ensure there is a BeeGFS mount point for this volume: 
	requestedMountPath, changed, err := mountBeegfs(volumeStagingTargetPath, requestedConfPath)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "%s\noccured while attempting to ensure a mount point existed for %s under %s", err, req.GetVolumeId(), volumeStagingTargetPath)
	}

	glog.Infof("NodeStageVolume: BeeGFS is mounted at %v (change required: %v).", requestedMountPath, changed)
	return &csi.NodeStageVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (ns *nodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId: ns.nodeID,
	}, nil
}

func (ns *nodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	var caps []*csi.NodeServiceCapability
	for _, cap := range nodeCaps {
		c := &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: cap,
				},
			},
		}
		caps = append(caps, c)
	}
	return &csi.NodeGetCapabilitiesResponse{Capabilities: caps}, nil
}

func (ns *nodeServer) NodeGetVolumeStats(ctx context.Context, in *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (ns *nodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}
