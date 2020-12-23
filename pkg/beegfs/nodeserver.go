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
	"os"
	"path"

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
	nodeID            string
	ephemeral         bool
	maxVolumesPerNode int64
	pluginConfig      pluginConfig
	confTemplatePath  string
	mounter           mount.Interface
}

func NewNodeServer(nodeId string, ephemeral bool, maxVolumesPerNode int64, pluginConfig pluginConfig, confTemplatePath string) *nodeServer {
	return &nodeServer{
		nodeID:            nodeId,
		ephemeral:         ephemeral,
		maxVolumesPerNode: maxVolumesPerNode,
		pluginConfig:      pluginConfig,
		confTemplatePath:  confTemplatePath,
		mounter:           mount.New(""),
	}
}

func (ns *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	var (
		// Effort has gone into maintaining consistent terminology for these various paths. Check other RPCs and
		// functions for consistency before refactoring.
		volDirPathBeegfsRoot string // absolute path to BeeGFS directory rooted at BeeGFS root (e.g. /parent/volume)
		volDirPath           string // absolute path to BeeGFS directory (e.g. .../mount/parent/volume)
		mountPath            string // absolute path to mount point
		remountPath          string // absolute path to bind mount point
		err                  error
	)

	// Check arguments
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	_, volDirPathBeegfsRoot, err = parseBeegfsUrl(req.GetVolumeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Could not parse VolumeID %s", req.GetVolumeId())
	}

	// File system is staged at StagingTargetPath/mount
	mountPath = path.Join(req.GetStagingTargetPath(), "mount")
	// Volume to be mounted into pod is staged at StagingTargetPath/beegfs/some/relative/path
	volDirPath = path.Clean(path.Join(mountPath, volDirPathBeegfsRoot))
	remountPath = req.GetTargetPath() // We bind mount wherever the CO tells us to.

	// Check to make sure file system is not already bind mounted
	// Use mount.IsNotMountPoint because mounter.IsLikelyNotMountPoint can't detect bind mounts
	var notMnt bool
	notMnt, err = mount.IsNotMountPoint(ns.mounter, remountPath)
	if err != nil {
		if os.IsNotExist(err) {
			// The file system can't be mounted because the mount point hasn't been created
			if err = fs.MkdirAll(remountPath, 0750); err != nil {
				return nil, status.Error(codes.Internal, err.Error())
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

	// Bind mount volDirPath onto TargetPath
	opts := []string{"bind"}
	err = ns.mounter.Mount(volDirPath, remountPath, "beegfs", opts)
	if err != nil {
		glog.Error(err.Error())
		return nil, status.Errorf(codes.Internal, "failed to mount %s onto %s: %v", req.GetStagingTargetPath(),
			req.GetTargetPath(), err)
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	var (
		// Effort has gone into maintaining consistent terminology for these various paths. Check other RPCs and
		// functions for consistency before refactoring.
		remountPath string // absolute path to bind mount point
		err         error
	)

	// Check arguments
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Missing volume ID")
	}
	if len(req.GetTargetPath()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Missing target path")
	}

	remountPath = req.GetTargetPath()

	err = mount.CleanupMountPoint(remountPath, ns.mounter, true)
	if err != nil {
		glog.Error(err.Error())
		return nil, status.Errorf(codes.Internal, "failed to unmount %s", req.GetTargetPath())
	}
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	var (
		// Effort has gone into maintaining consistent terminology for these various paths. Check other RPCs and
		// functions for consistency before refactoring.
		sysMgmtdHost string // IP address or hostname of BeeGFS mgmtd service
		mountDirPath string // absolute path to directory containing configuration files and mount point
		err          error
	)

	// Check arguments
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	sysMgmtdHost, _, err = parseBeegfsUrl(req.GetVolumeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Failed to parse VolumeID %s", req.GetVolumeId())
	}
	if len(req.GetStagingTargetPath()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}
	if req.GetVolumeCapability() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume Capability missing in request")
	}

	mountDirPath = req.GetStagingTargetPath() // We mount wherever the CO tells us to.

	// TODO(jmccormi): Check and return all possible NodeStageVolume errors.
	// https://github.com/container-storage-interface/spec/blob/master/spec.md#nodestagevolume-errors

	// Ensure CO has already created StagingTargetPath.
	_, err = fs.Stat(req.GetStagingTargetPath())
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to stat the staging target directory for %s: %v",
			err, req.GetStagingTargetPath())
	}

	// Write configuration files and mount BeeGFS.
	// TODO(webere): Consider creating a single abstracting function.
	if err := fs.MkdirAll(mountDirPath, 0750); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	beegfsConfig := squashConfigForSysMgmtdHost(sysMgmtdHost, ns.pluginConfig)
	_, _, err = writeClientFiles(sysMgmtdHost, mountDirPath, ns.confTemplatePath, beegfsConfig)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "%v", err)
	}
	err = mountIfNecessary(mountDirPath, ns.mounter)
	if err != nil {
		glog.Error(err)
		return nil, status.Errorf(codes.Internal, "Failed to mount filesystem %s to %s", sysMgmtdHost, mountDirPath)
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	var (
		// Effort has gone into maintaining consistent terminology for these various paths. Check other RPCs and
		// functions for consistency before refactoring.
		mountDirPath string // absolute path to directory containing configuration files and mount point
		err          error
	)

	// Check arguments
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if len(req.GetStagingTargetPath()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	mountDirPath = req.GetStagingTargetPath() // We are mounted wherever the CO told us to.

	err = unmountAndCleanUpIfNecessary(mountDirPath, false, ns.mounter) // The CO will clean up mountDirPath.
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to clean up %s: %v", mountDirPath, err)
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
