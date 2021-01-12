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
	"fmt"
	"path"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/mount"
)

var (
	// controllerCaps represents the capability of controller service
	controllerCaps = []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	}

	// TODO(jparnell) consider reader options
	volumeCaps = []csi.VolumeCapability_AccessMode{
		{
			Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		},
		{
			Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
		},
	}
)

type controllerServer struct {
	ctlExec          beegfsCtlExecutorInterface
	caps             []*csi.ControllerServiceCapability
	nodeID           string
	pluginConfig     pluginConfig
	confTemplatePath string
	mounter          mount.Interface
	dataDir          string
}

func NewControllerServer(ephemeral bool, nodeID string, pluginConfig pluginConfig, confTemplatePath, dataDir string) *controllerServer {
	if ephemeral {
		return &controllerServer{caps: getControllerServiceCapabilities(nil), nodeID: nodeID}
	}
	return &controllerServer{
		ctlExec: &beegfsCtlExecutor{},
		caps: getControllerServiceCapabilities(
			[]csi.ControllerServiceCapability_RPC_Type{
				csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
			}),
		nodeID:           nodeID,
		pluginConfig:     pluginConfig,
		confTemplatePath: confTemplatePath,
		dataDir:          dataDir,
		mounter:          nil,
	}
}

// CreateVolume generates a new volumeID and uses beegfs-ctl to create an associated directory at the proper location
// on the referenced BeeGFS file system. CreateVolume uses beegfs-ctl instead of mounting the file system and using
// mkdir because it needs to be able to use beegfs-ctl to set stripe patterns, etc. anyway.
// TODO(webere): This function returns quite a few errors with no valid GRPC error codes
func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	// Check arguments.
	volName := req.GetName()
	if len(volName) == 0 {
		return nil, fmt.Errorf("Volume name not provided")
	}
	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, fmt.Errorf("Volume capabilities not provided")
	}
	if !cs.isValidVolumeCapabilities(volCaps) {
		return nil, fmt.Errorf("Volume capabilities not supported")
	}
	reqParams := req.GetParameters()
	if len(reqParams) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Request parameters not provided")
	}
	sysMgmtdHost, ok := reqParams[sysMgmtdHostKey]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "%s missing in request parameters", sysMgmtdHostKey)
	}
	volDirBasePathBeegfsRoot, ok := reqParams[volDirBasePathKey]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "%s missing in request parameters", volDirBasePathKey)
	}
	volDirBasePathBeegfsRoot = path.Clean(path.Join("/", volDirBasePathBeegfsRoot))

	vol := cs.newBeegfsVolume(sysMgmtdHost, volDirBasePathBeegfsRoot, volName)

	// Write configuration files but do not mount BeeGFS.
	if err := fs.MkdirAll(vol.mountDirPath, 0750); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if err := writeClientFiles(vol, cs.confTemplatePath); err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}

	if err := cs.ctlExec.createDirectoryForVolume(vol); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Clean up configuration files.
	if err := cleanUpIfNecessary(vol, true); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId: vol.volumeID,
		},
	}, nil
}

// DeleteVolume deletes the directory referenced in the volumeID from the BeeGFS file system referenced in the
// volumeID.
func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	// Check arguments.
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Volume ID not provided")
	}

	vol, err := cs.newBeegfsVolumeFromID(volumeID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Write configuration files and mount BeeGFS.
	if err := fs.MkdirAll(vol.mountDirPath, 0750); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if err := writeClientFiles(vol, cs.confTemplatePath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if err := mountIfNecessary(vol, cs.mounter); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Delete volume from mounted BeeGFS.
	glog.Infof("Removing %s from filesystem %s", vol.volDirPath, vol.sysMgmtdHost)
	if err = fs.RemoveAll(vol.volDirPath); err != nil {
		glog.Error(err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Unmount BeeGFS and clean up configuration files.
	if err = unmountAndCleanUpIfNecessary(vol, true, cs.mounter); err != nil {
		glog.Error(err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *controllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	var caps []*csi.ControllerServiceCapability
	for _, cap := range controllerCaps {
		c := &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: cap,
				},
			},
		}
		caps = append(caps, c)
	}
	return &csi.ControllerGetCapabilitiesResponse{Capabilities: caps}, nil
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not provided")
	}

	if _, err := getVolumeByID(volumeID); err != nil {
		return nil, status.Error(codes.NotFound, volumeID)
	}

	confirmed := cs.isValidVolumeCapabilities(volCaps)
	if confirmed {
		return &csi.ValidateVolumeCapabilitiesResponse{
			Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
				// TODO(jparnell) if volume context is provided, could validate it too
				// VolumeContext:      req.GetVolumeContext(),
				VolumeCapabilities: volCaps,
				// TODO(jparnell) if parameters are provided, could validate them too
				// Parameters:      req.GetParameters(),
			},
		}, nil
	} else {
		return &csi.ValidateVolumeCapabilitiesResponse{}, nil
	}
}

func (cs *controllerServer) isValidVolumeCapabilities(volCaps []*csi.VolumeCapability) bool {
	hasSupport := func(cap *csi.VolumeCapability) bool {
		for _, c := range volumeCaps {
			if c.GetMode() == cap.AccessMode.GetMode() {
				return true
			}
		}
		return false
	}

	foundAll := true
	for _, c := range volCaps {
		if !hasSupport(c) {
			foundAll = false
		}
	}
	return foundAll
}

func (cs *controllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) ControllerGetVolume(ctx context.Context, in *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func getControllerServiceCapabilities(cl []csi.ControllerServiceCapability_RPC_Type) []*csi.ControllerServiceCapability {
	var csc []*csi.ControllerServiceCapability

	for _, cap := range cl {
		glog.Infof("Enabling controller service capability: %v", cap.String())
		csc = append(csc, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: cap,
				},
			},
		})
	}

	return csc
}

// (*controllerServer) newBeegfsVolume is a wrapper around newBeegfsVolume that makes it easier to call in the context
// of the controller service. (*controllerServer) newBeegfsVolume selects the mountDirPath and passes the controller
//service's pluginConfig.
func (cs *controllerServer) newBeegfsVolume(sysMgmtdHost, volDirBasePathBeegfsRoot, volName string) beegfsVolume {
	volDirPathBeegfsRoot := path.Join(volDirBasePathBeegfsRoot, volName)
	// This volumeID construction duplicates the one further down in the stack. We do it anyway to generate an
	// appropriate mountDirPath.
	volumeID := newBeegfsUrl(sysMgmtdHost, volDirPathBeegfsRoot)
	mountDirPath := path.Join(cs.dataDir, sanitizeVolumeID(volumeID)) // e.g. /dataDir/127.0.0.1_scratch_pvc-12345678
	return newBeegfsVolume(mountDirPath, sysMgmtdHost, volDirPathBeegfsRoot, cs.pluginConfig)
}

// (*controllerServer) newBeegfsVolumeFromID is a wrapper around newBeegfsVolumeFromID that makes it easier to call in
// the context of the controller service. (*controllerServer) newBeegfsVolumeFromID selects the mountDirPath and passes
// the controller service's pluginConfig.
func (cs *controllerServer) newBeegfsVolumeFromID(volumeID string) (beegfsVolume, error) {
	return newBeegfsVolumeFromID(cs.dataDir, volumeID, cs.pluginConfig)
}
