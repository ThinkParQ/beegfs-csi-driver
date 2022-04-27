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

	"github.com/container-storage-interface/spec/lib/go/csi"
	beegfsv1 "github.com/netapp/beegfs-csi-driver/operator/api/v1"
	"github.com/pkg/errors"
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
	ctlExec                beegfsCtlExecutorInterface
	nodeID                 string
	pluginConfig           beegfsv1.PluginConfig
	clientConfTemplatePath string
	mounter                mount.Interface
}

func newNodeServer(nodeID string, pluginConfig beegfsv1.PluginConfig, clientConfTemplatePath string) (*nodeServer, error) {
	executor, err := newBeeGFSCtlExecutor()
	if err != nil {
		return nil, err
	}
	return &nodeServer{
		ctlExec:                executor,
		nodeID:                 nodeID,
		pluginConfig:           pluginConfig,
		clientConfTemplatePath: clientConfTemplatePath,
		mounter:                mount.New(""),
	}, nil
}

func newNodeServerSanity(nodeID string, pluginConfig beegfsv1.PluginConfig, clientConfTemplatePath string) *nodeServer {
	return &nodeServer{
		ctlExec:                &fakeBeegfsCtlExecutor{},
		nodeID:                 nodeID,
		pluginConfig:           pluginConfig,
		clientConfTemplatePath: clientConfTemplatePath,
		mounter:                mount.NewFakeMounter([]mount.MountPoint{}),
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
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}

	// Only continue if our target directory exists. Check using beegfs-ctl instead of something more straightforward
	// (e.g. fs.Stat(vol.volDirPath)) because beegfs-ctl is easier to mock for sanity tests. Client files are already
	// written. If they weren't, the volume couldn't have been staged.
	if _, err := ns.ctlExec.statDirectoryForVolume(ctx, vol, vol.volDirPathBeegfsRoot); err != nil {
		if errors.As(err, &ctlNotExistError{}) {
			return nil, newGrpcErrorFromCause(codes.NotFound, err)
		}
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}

	// Check to make sure file system is not already bind mounted
	// Use mount.IsNotMountPoint because mounter.IsLikelyNotMountPoint can't detect bind mounts
	var notMnt bool
	notMnt, err = mount.IsNotMountPoint(ns.mounter, targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			// The file system can't be mounted because the mount point hasn't been created
			if err = fs.MkdirAll(targetPath, 0750); err != nil {
				err = errors.WithStack(err)
				return nil, newGrpcErrorFromCause(codes.Internal, err)
			}
			notMnt = true
		} else {
			err = errors.WithStack(err)
			return nil, newGrpcErrorFromCause(codes.Internal, err)
		}
	}
	if !notMnt {
		// The filesystem is already mounted. There is nothing to do.
		LogDebug(ctx, "Volume is already mounted to path", "volumeID", vol.volumeID, "path", vol.mountPath)
		return &csi.NodePublishVolumeResponse{}, nil
	}

	opts := volCap.GetMount().MountFlags
	// Bind mount volDirPath onto TargetPath.
	opts = append(opts, "bind")
	if readOnly {
		// TODO(webere, A143): Get read-only mounts propagating outside of the plugin container.
		// When the driver runs in a container (as is standard in a K8s deployment), the bind mount appears read-only
		// within that container, but not outside the container (on the host or in another pod). K8s read-only mounts
		// still work as expected (due to a "last-mile" read-only bind mount into the K8s pod), but read-only may not
		// work as expected for other COs.
		opts = append(opts, "ro")
	}
	opts = removeInvalidMountOptions(ctx, opts)
	LogDebug(ctx, "Mounting volume", "volDirPath", vol.volDirPath, "targetPath", targetPath, "options", opts)
	err = ns.mounter.Mount(vol.volDirPath, targetPath, "beegfs", opts)
	if err != nil {
		err = errors.WithStack(err)
		return nil, newGrpcErrorFromCause(codes.Internal, err)
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

	LogDebug(ctx, "Unmounting volume", "volumeID", volumeID, "mountPath", targetPath)
	if err := mount.CleanupMountPoint(targetPath, ns.mounter, true); err != nil {
		err = errors.WithStack(err)
		return nil, newGrpcErrorFromCause(codes.Internal, err)
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
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}

	// Ensure mountDirPath already exists (CO should have created req.StagingTargetPath).
	_, err = fs.Stat(vol.mountDirPath)
	if err != nil {
		err = errors.WithStack(err)
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}

	// Write configuration files.
	if err := writeClientFiles(ctx, vol, ns.clientConfTemplatePath); err != nil {
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}

	mountOptions := volCap.GetMount().MountFlags
	// Only mount BeeGFS if beegfs-ctl reports our target directory exists.
	if _, err := ns.ctlExec.statDirectoryForVolume(ctx, vol, vol.volDirPathBeegfsRoot); err != nil {
		if errors.As(err, &ctlNotExistError{}) {
			return nil, newGrpcErrorFromCause(codes.NotFound, err)
		}
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}
	if err := mountIfNecessary(ctx, vol, mountOptions, ns.mounter); err != nil {
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}

	// At this point, the volume should be mounted. Add our nodeID to the appropriate tracking directory. This is best
	// effort. Log if something goes wrong, but don't fail the operation.
	nodesPath := path.Join(vol.csiDirPath, "nodes")
	nodePath := path.Join(nodesPath, ns.nodeID)
	dirExists, err := fsutil.DirExists(nodesPath)
	if err != nil {
		LogError(ctx, err, "Failed attempting to stat node tracking directory", "path", nodesPath, "volumeID", vol.volumeID)
	} else if !dirExists {
		LogVerbose(ctx, "Node tracking directory doesn't exist for volume", "path", nodesPath, "volumeID", vol.volumeID)
	} else {
		LogDebug(ctx, "Creating file in node tracking directory", "path", nodePath, "volumeID", vol.volumeID)
		// Use WriteFile instead of Create to avoid the need for Close.
		if err = fsutil.WriteFile(nodePath, []byte{}, 0640); err != nil {
			LogError(ctx, err, "Failed to create file in node tracking directory", "path", nodePath, "volumeID", vol.volumeID)
		}
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
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}

	// While the volume is still mounted, delete our nodeID from the appropriate tracking directory. This is best
	// effort. Log if something goes wrong, but don't fail the operation.
	nodesPath := path.Join(vol.csiDirPath, "nodes")
	nodePath := path.Join(nodesPath, ns.nodeID)
	fileExists, err := fsutil.Exists(nodePath)
	if err != nil {
		LogError(ctx, err, "Failed attempting to stat node tracking file", "path", nodePath, "volumeID", vol.volumeID)
	} else if !fileExists {
		LogVerbose(ctx, "Node tracking file doesn't exist for node", "path", nodePath, "volumeID", vol.volumeID)
	} else {
		LogDebug(ctx, "Deleting node tracking file", "path", nodePath, "volumeID", vol.volumeID)
		if err = fs.Remove(nodePath); err != nil {
			LogError(ctx, err, "Failed to delete node tracking file", "path", nodePath, "volumeID", vol.volumeID)
		}
	}

	err = unmountAndCleanUpIfNecessary(ctx, vol, false, ns.mounter) // The CO will clean up mountDirPath.
	if err != nil {
		return nil, newGrpcErrorFromCause(codes.Internal, err)
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
