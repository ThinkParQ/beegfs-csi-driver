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

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/mount"
)

var (
	nodeCaps = []csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
	}
)

type nodeServer struct {
	nodeID                 string
	pluginConfig           pluginConfig
	clientConfTemplatePath string
	mounter                mount.Interface
}

func NewNodeServer(nodeId string, pluginConfig pluginConfig, clientConfTemplatePath string) *nodeServer {
	return &nodeServer{
		nodeID:                 nodeId,
		pluginConfig:           pluginConfig,
		clientConfTemplatePath: clientConfTemplatePath,
		mounter:                nil,
	}
}

func (ns *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	// Check arguments.
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Volume ID not provided")
	}
	stagingTargetPath := req.GetStagingTargetPath()
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Staging target path not provided")
	}
	targetPath := req.GetTargetPath()
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path not provided")
	}
	volCap := req.GetVolumeCapability()
	if volCap == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not provided")
	}
	if valid, reason := isValidVolumeCapability(volCap); !valid {
		return nil, status.Errorf(codes.InvalidArgument, "Volume capability not supported: %s", reason)
	}
	readOnly := req.GetReadonly()

	vol, err := newBeegfsVolumeFromID(stagingTargetPath, volumeID, ns.pluginConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Check to make sure file system is not already bind mounted
	// Use mount.IsNotMountPoint because mounter.IsLikelyNotMountPoint can't detect bind mounts
	var notMnt bool
	notMnt, err = mount.IsNotMountPoint(ns.mounter, targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			// The file system can't be mounted because the mount point hasn't been created
			glog.V(LogDebug).Infof("Making directories for mount point %s", targetPath)
			if err = fs.MkdirAll(targetPath, 0750); err != nil {
				return nil, status.Errorf(codes.Internal, "failed making directories for mount point %s: %s", targetPath, err.Error())
			}
			notMnt = true
		} else {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	if !notMnt {
		// The filesystem is already mounted. There is nothing to do.
		return &csi.NodePublishVolumeResponse{}, nil
	}

	// Bind mount volDirPath onto TargetPath.
	opts := []string{"bind"}
	if readOnly {
		// TODO(webere, A143): Get read-only mounts propagating outside of the plugin container.
		// When the driver runs in a container (as is standard in a K8s deployment), the bind mount appears read-only
		// within that container, but not outside the container (on the host or in another pod). K8s read-only mounts
		// still work as expected (due to a "last-mile" read-only bind mount into the K8s pod), but read-only may not
		// work as expected for other COs.
		opts = append(opts, "ro")
	}
	glog.V(LogDebug).Infof("Mounting %s to %s with options %s", vol.volDirPath, targetPath, opts)
	err = ns.mounter.Mount(vol.volDirPath, targetPath, "beegfs", opts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to mount %s onto %s: %v", req.GetStagingTargetPath(),
			req.GetTargetPath(), err)
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	// Check arguments.
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Volume ID not provided")
	}
	targetPath := req.GetTargetPath()
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path not provided")
	}

	glog.V(LogDebug).Infof("Cleaning up mount point %s", targetPath)
	if err := mount.CleanupMountPoint(targetPath, ns.mounter, true); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	// Check arguments.
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Volume ID not provided")
	}
	stagingTargetPath := req.GetStagingTargetPath()
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Staging target path not provided")
	}
	volCap := req.GetVolumeCapability()
	if volCap == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not provided")
	}
	if valid, reason := isValidVolumeCapability(volCap); !valid {
		return nil, status.Errorf(codes.InvalidArgument, "Volume capability not supported: %s", reason)
	}

	vol, err := newBeegfsVolumeFromID(stagingTargetPath, volumeID, ns.pluginConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Ensure mountDirPath already exists (CO should have created req.StagingTargetPath).
	_, err = fs.Stat(vol.mountDirPath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Write configuration files and mount BeeGFS.
	if err := writeClientFiles(vol, ns.clientConfTemplatePath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if err := mountIfNecessary(vol, ns.mounter); err != nil {
		// TODO(webere, A144): Return the appropriate codes.NOT_FOUND if the problem is that we can't find the volume.
		// https://github.com/container-storage-interface/spec/blob/master/spec.md#nodestagevolume-errors
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	// Check arguments.
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Volume ID not provided")
	}
	stagingTargetPath := req.GetStagingTargetPath()
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Staging target path not provided")
	}

	vol, err := newBeegfsVolumeFromID(stagingTargetPath, volumeID, ns.pluginConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = unmountAndCleanUpIfNecessary(vol, false, ns.mounter) // The CO will clean up mountDirPath.
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
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
