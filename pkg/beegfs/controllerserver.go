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
	// Get or generate necessary parameters to generate URL
	reqParams := req.GetParameters()
	sysMgmtdHost, ok := getBeegfsConfValueFromParams(sysMgmtdHostKey, reqParams)
	if !ok {
		return nil, fmt.Errorf("%s%s not in CreateVolumeRequest.Parameters", beegfsConfPrefix, sysMgmtdHostKey)
	}
	volDirBasePath, ok := reqParams[volDirBasePathKey]
	if !ok {
		return nil, fmt.Errorf("%s not in CreateVolumeRequest parameters", volDirBasePathKey)
	}
	dirPath := path.Join(volDirBasePath, req.GetName())

	// Generate a beegfs-client.conf file under dataRoot if necessary
	cfgFilePath, _, err := generateBeeGFSClientConf(reqParams, dataRoot)
	if err != nil {
		return nil, err
	}

	// Check if volume already exists
	_, err = beegfsCtlExec(cfgFilePath, []string{"--unmounted", "--getentryinfo", dirPath})
	if err != nil {
		// TODO(webere) More in-depth error check
		// We can't find the volume so we need to create one
		glog.Infof("Volume %s does not exist under directory %s on BeeGFS instance %s", req.GetName(), volDirBasePath,
			sysMgmtdHost)
		_, err := beegfsCtlExec(cfgFilePath, []string{"--unmounted", "--createdir", dirPath})
		if err != nil {
			// We can't create the volume
			return nil, fmt.Errorf("cannot create volume with path %s on filesystem %s", dirPath, sysMgmtdHost)
		}
	} else {
		glog.Infof("Volume %s already exists under directory %s on BeeGFS instance %s", req.GetName(), volDirBasePath,
			sysMgmtdHost)
	}

	volumeID := newBeegfsUrl(sysMgmtdHost, dirPath)
	glog.Infof("Generated ID %s for volume %s", volumeID, req.GetName())

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volumeID,
			VolumeContext: reqParams, // These params will be needed again by the node service
		},
	}, nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
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
