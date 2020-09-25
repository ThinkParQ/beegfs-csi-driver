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
	"strings"

	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

const (
	volDirBasePathKey = "volDirBasePath"
	beegfsConfPrefix  = "beegfsConf/"
	sysMgmtdHostKey   = "sysMgmtdHost"
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
	caps   []*csi.ControllerServiceCapability
	nodeID string
}

func NewControllerServer(ephemeral bool, nodeID string) *controllerServer {
	if ephemeral {
		return &controllerServer{caps: getControllerServiceCapabilities(nil), nodeID: nodeID}
	}
	return &controllerServer{
		caps: getControllerServiceCapabilities(
			[]csi.ControllerServiceCapability_RPC_Type{
				csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
			}),
		nodeID: nodeID,
	}
}

func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, fmt.Errorf("Volume capabilities not provided")
	}

	if !cs.isValidVolumeCapabilities(volCaps) {
		return nil, fmt.Errorf("Volume capabilities not supported")
	}

	// TODO(webere): This function returns quite a few errors with no valid GRPC error codes
	// Get or generate necessary parameters to generate URL
	reqParams := req.GetParameters()
	sysMgmtdHost, ok := getBeegfsConfValueFromParams(sysMgmtdHostKey, reqParams)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument,"%s%s not in CreateVolumeRequest.Parameters", beegfsConfPrefix, sysMgmtdHostKey)
	}
	volDirBasePath, ok := reqParams[volDirBasePathKey]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "%s not in CreateVolumeRequest parameters", volDirBasePathKey)
	}
	volDirBasePath = strings.TrimLeft(volDirBasePath, "/")  // Trim leading slash if included

	dirPath := path.Join(volDirBasePath, req.GetName())

	// Generate a beegfs-client.conf file under dataRoot if necessary
	cfgFilePath, _, err := generateBeeGFSClientConf(reqParams, dataRoot, true)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Check if volume already exists
	_, err = beegfsCtlExec(cfgFilePath, []string{"--unmounted", "--getentryinfo", dirPath})
	if err != nil {
		// TODO(webere) More in-depth error check
		// We can't find the volume so we need to create one
		glog.Infof("Volume %s does not exist under directory %s on BeeGFS instance %s", req.GetName(), volDirBasePath,
			sysMgmtdHost)

		// Create parent directories if necessary
		// Create a slice of paths where the first path is the most general and each subsequent path is less general
		dirsToMake := []string{dirPath}
		for dir := path.Dir(dirPath); dir != "."; {  // path.Dir() returns "." if there is no parent
			dirsToMake = append([]string{dir}, dirsToMake...)  // Prepend so the more general path comes first
			dir = path.Dir(dir)
		}
		// Starting with the most general path, create all directories required to eventually create dirPath
		for _, dir := range dirsToMake {
			_, err := beegfsCtlExec(cfgFilePath, []string{"--unmounted", "--createdir", dir})
			if err != nil && strings.Contains(err.Error(), "Entry exists already"){
				// We can't create the volume
				return nil, status.Errorf(codes.Internal, "cannot create directory with path %s on filesystem %s", dir, sysMgmtdHost)
			}
		}
	} else {
		glog.Infof("Volume %s already exists under directory %s on BeeGFS instance %s", req.GetName(), volDirBasePath,
			sysMgmtdHost)
	}

	volumeID := newBeegfsUrl(sysMgmtdHost, dirPath)
        // TODO(jparnell) handle volume map
	glog.Infof("Generated ID %s for volume %s", volumeID, req.GetName())

	// TODO(webere): Clean up beegfs-client.conf file if we know we no longer need it
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volumeID,
			VolumeContext: reqParams, // These params will be needed again by the node service
		},
	}, nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	// Get and parse volumeID
	volumeId := req.GetVolumeId()
	sysMgmtdHost, dirPath, err := parseBeegfsUrl(volumeId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", err)
	}

	// Generate a beegfs-client.conf file under dataRoot if necessary
	// We don't have params, so don't attempt to overwrite the file if it exists
	simpleParams := map[string]string{path.Join(beegfsConfPrefix, "sysMgmtdHost"): sysMgmtdHost}
	cfgFilePath, _, err := generateBeeGFSClientConf(simpleParams, dataRoot, false)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err)
	}

	// Delete volume
	// TODO(webere): This function CAN'T work as anticipated, because there is no deletedir or related beegfs-ctl command
	if _, err := beegfsCtlExec(cfgFilePath, []string{"--unmounted", "--deletedir"}); err != nil {
		return nil, status.Errorf(codes.Unavailable, "Cannot delete volume with path %s on filesystem %s", dirPath, sysMgmtdHost)
	}

	// TODO(webere): Clean up beegfs-client.conf file if we know we no longer need it
	return nil, nil
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

func (cs *controllerServer) validateControllerServiceRequest(c csi.ControllerServiceCapability_RPC_Type) error {
	return status.Error(codes.Unimplemented, "")
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
