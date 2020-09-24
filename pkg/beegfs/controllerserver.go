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
	"strings"

	//	"fmt"
	//	"math"
	//	"os"
	//	"path/filepath"
	//	"sort"
	//	"strconv"
	//
	//	"github.com/golang/protobuf/ptypes"
	//
	"github.com/golang/glog"
	//	"github.com/pborman/uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/container-storage-interface/spec/lib/go/csi"
	//	utilexec "k8s.io/utils/exec"
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

	// Get or generate necessary parameters to generate URL
	reqParams := req.GetParameters()
	sysMgmtdHost, ok := getBeegfsConfValueFromParams(sysMgmtdHostKey, reqParams)
	if !ok {
		return nil, fmt.Errorf("%s%s not in CreateVolumeRequest.Parameters", beegfsConfPrefix, sysMgmtdHostKey)
	}
	parentDirPath, ok := reqParams[volDirBasePathKey]
	if !ok {
		return nil, fmt.Errorf("%s not in CreateVolumeRequest parameters", volDirBasePathKey)
	}
	dirPath := path.Join(parentDirPath, req.GetName())
	cfgFilePath := path.Join(dataRoot, strings.Replace(sysMgmtdHost, ".", "_", 3)+"_beegfs-client.conf")

	// Check if volume already exists
	_, err := beegfsCtlExec(cfgFilePath, []string{"--unmounted", "--getentryinfo", dirPath})
	if err != nil {
		// TODO(webere) More in-depth error check
		// We couldn't find the volume so we need to create one
		_, err := beegfsCtlExec(cfgFilePath, []string{"--unmounted", "--createdir", dirPath})
		if err != nil {
			// We couldn't create the volume
			return nil, fmt.Errorf("could not create volume with path %s on filesystem %s", dirPath, sysMgmtdHost)
		}
	}

	volumeID := newBeegfsUrl(sysMgmtdHost, dirPath)
        // TODO(jparnell) handle volume map

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId: volumeID,
		},
	}, nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
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

	if _, err := getVolumeByID(volumeID)
		if err == nil {
			reurn nil, status.Error(codes.NotFound, "Volume not found with ID %q", volumeID)
		}
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
