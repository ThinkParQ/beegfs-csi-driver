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

	// fmt.Printf("Context: %#v\n", ctx)
	// fmt.Printf("Request: %#v\n", req)

	//Ensure this node has a BeeGFS client configuration file for the requested BeeGFS instance:
	requestedConfPath, _, err := generateBeeGFSClientConf(req.VolumeContext, dataRoot, true)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "error %s occured generating a BeeGFS client conf file for %s at %s", err, req.VolumeContext, dataRoot)
	}

	// Ensure this BeeGFS instance is mounted:
	requestedMountPath, changed, err := mountBeegfs(dataRoot, requestedConfPath)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "error '%s' occured while attempting to ensure a mount point existed for %s under %s", err, req.GetVolumeId(), dataRoot)
	}

	glog.Infof("NodeStageVolume: BeeGFS is mounted at %v (change required: %v).", requestedMountPath, changed)

	// TODO (jmccormi): Return an appropriate response, probably the path where BeeGFS is mounted.
	return &csi.NodeStageVolumeResponse{}, nil
	//return nil, status.Error(codes.Unimplemented, "")
}

func (ns *nodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (ns *nodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (ns *nodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (ns *nodeServer) NodeGetVolumeStats(ctx context.Context, in *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (ns *nodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}
