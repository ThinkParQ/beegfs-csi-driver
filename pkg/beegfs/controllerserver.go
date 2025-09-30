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
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	beegfsv1 "github.com/netapp/beegfs-csi-driver/operator/api/v1"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/mount-utils"
)

var (
	// controllerCaps represents the capabilities of the controller service
	controllerCaps = []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
	}
)

type controllerServer struct {
	ctlExec                beegfsCtlExecutorInterface
	nodeID                 string
	pluginConfig           beegfsv1.PluginConfig
	clientConfTemplatePath string
	mounter                mount.Interface
	csDataDir              string
	volumeIDsInFlight      *threadSafeStringLock
	volumeStatusMap        *threadSafeStatusMap
	nodeUnstageTimeout     uint64
	csi.UnimplementedControllerServer
}

func newControllerServer(nodeID string, pluginConfig beegfsv1.PluginConfig, clientConfTemplatePath, csDataDir string,
	nodeUnstageTimeout uint64) (*controllerServer, error) {
	executor, err := newBeeGFSCtlExecutor()
	if err != nil {
		return nil, err
	}
	return &controllerServer{
		ctlExec:                executor,
		nodeID:                 nodeID,
		pluginConfig:           pluginConfig,
		clientConfTemplatePath: clientConfTemplatePath,
		csDataDir:              csDataDir,
		mounter:                mount.New(""),
		volumeIDsInFlight:      newThreadSafeStringLock(),
		volumeStatusMap:        newThreadSafeStatusMap(),
		nodeUnstageTimeout:     nodeUnstageTimeout,
	}, err
}

func newControllerServerSanity(nodeID string, pluginConfig beegfsv1.PluginConfig, clientConfTemplatePath, csDataDir string,
	nodeUnstageTimeout uint64) *controllerServer {
	return &controllerServer{
		ctlExec:                &fakeBeegfsCtlExecutor{},
		nodeID:                 nodeID,
		pluginConfig:           pluginConfig,
		clientConfTemplatePath: clientConfTemplatePath,
		csDataDir:              csDataDir,
		mounter:                mount.NewFakeMounter([]mount.MountPoint{}),
		volumeIDsInFlight:      newThreadSafeStringLock(),
		volumeStatusMap:        newThreadSafeStatusMap(),
		nodeUnstageTimeout:     nodeUnstageTimeout,
	}
}

// CreateVolume generates a new volumeID and uses beegfs-ctl to create an associated directory at the proper location
// on the referenced BeeGFS file system. CreateVolume will not mount the filesystem unless the volume configuration
// requires the use of special permissions (sticky bit, setuid, setgid), in which case the filesystem will be mounted
// to properly set those permissions.
func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	// Check arguments.
	volName := req.GetName()
	if len(volName) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume name not provided")
	}
	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not provided")
	}
	if valid, reason := isValidVolumeCapabilities(volCaps); !valid {
		return nil, status.Errorf(codes.InvalidArgument, "Volume capabilities not supported: %s", reason)
	}

	if len(req.GetParameters()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Parameters not provided")
	}
	params, err := validateReqParams(req.GetParameters())
	if err != nil {
		return nil, newGrpcErrorFromCause(codes.InvalidArgument, err)
	}

	// Construct an internal representation of the volume.
	vol := cs.newBeegfsVolume(params.sysMgmtdHost, params.volDirBasePathBeegfsRoot, volName)

	// Return success if we don't need to do anything.
	if status, ok := cs.volumeStatusMap.readStatus(vol.volumeID); ok && status == statusCreated {
		return &csi.CreateVolumeResponse{
			Volume: &csi.Volume{
				VolumeId: vol.volumeID,
			},
		}, nil
	}

	// Obtain exclusive control over the volume.
	if !cs.volumeIDsInFlight.obtainLockOnString(vol.volumeID) {
		return nil, status.Errorf(codes.Aborted, "volumeID %s is in use by another request; check BeeGFS network "+
			"configuration if this problem persists", vol.volumeID)
	}
	defer cs.volumeIDsInFlight.releaseLockOnString(vol.volumeID)

	// Write configuration files but do not mount BeeGFS.
	defer func() {
		// Failure to clean up is an internal problem. The CO only cares whether or not we created the volume.
		// If CreateVolume exits with an error, the deferred cleanup will pass on the error regardless of the outcome
		// of the cleanup.
		// In theory an error unmounting the filesystem could result in an orphaned mount on the host. We don't think
		// returning an error is appropriate if the mount failed to clean if the volume was successfully created.
		// We may consider alternate behavior if we observe cases where orphaned mounts occur and are believed
		// to cause a problem. We could retry the unmount, but until then we will leave the behavior as is.
		if err := unmountAndCleanUpIfNecessary(ctx, vol, true, cs.mounter); err != nil {
			LogError(ctx, err, "Failed to clean up path for volume", "path", vol.mountDirPath, "volumeID", vol.volumeID)
		}
	}()
	if err := fs.MkdirAll(vol.mountDirPath, 0750); err != nil {
		err = errors.WithStack(err)
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}
	if err := writeClientFiles(ctx, vol, cs.clientConfTemplatePath); err != nil {
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}

	// Use beegfs-ctl to create the directory and stripe it appropriately.
	if err := cs.ctlExec.createDirectoryForVolume(ctx, vol, vol.volDirPathBeegfsRoot, params.volPermissionsConfig); err != nil {
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}
	if err := cs.ctlExec.setPatternForVolume(ctx, vol, params.volStripePatternConfig); err != nil {
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}

	// Mount BeeGFS and use OS tools to change the access mode only if beegfs-ctl could not handle the access mode
	// on its own. beegfs-ctl cannot handle access modes with special permissions (e.g. the set gid bit). These are
	// governed by the first three bits of a 12 bit access mode (i.e. the first digit in four digit octal notation).
	if params.volPermissionsConfig.hasSpecialPermissions() {
		if err := mountIfNecessary(ctx, vol, []string{}, cs.mounter); err != nil {
			return nil, newGrpcErrorFromCause(codes.Internal, err)
		}
		LogDebug(ctx, "Applying permissions", "permissions", fmt.Sprintf("%4o", params.volPermissionsConfig.mode),
			"volDirPath", vol.volDirPath, "volumeID", vol.volumeID)
		if err := os.Chmod(vol.volDirPath, params.volPermissionsConfig.goFileMode()); err != nil {
			return nil, newGrpcErrorFromCause(codes.Internal, err)
		}
	}

	// Use beegfs-ctl to create the directory we will use to track the nodes that mount this volume. Don't mount and
	// use mkdir in case there is no other need to mount.
	if cs.nodeUnstageTimeout > 0 {
		nodesDirPath := path.Join(vol.csiDirPathBeegfsRoot, "nodes")
		nodesDirPermissions := permissionsConfig{mode: 0750}
		if err = cs.ctlExec.createDirectoryForVolume(ctx, vol, nodesDirPath, nodesDirPermissions); err != nil {
			LogError(ctx, err,
				"Failed to create subdirectory for node tracking", "path", nodesDirPath, "volumeID", vol.volumeID)
		}
	} else {
		LogVerbose(ctx, "Node tracking not enabled", "volumeID", vol.volumeID)
	}

	// Update status and return.
	cs.volumeStatusMap.writeStatus(vol.volumeID, statusCreated)
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId: vol.volumeID,
		},
	}, nil
}

// DeleteVolume deletes the directory referenced in the volumeID from the BeeGFS file system referenced in the
// volumeID.
func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (resp *csi.DeleteVolumeResponse, err error) {
	// Check arguments.
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Volume ID not provided")
	}

	// Construct an internal representation of the volume.
	vol, err := cs.newBeegfsVolumeFromID(volumeID)
	if err != nil {
		LogError(ctx, err, "Beegfs volume not found for deletion", "volumeID", volumeID)
		return &csi.DeleteVolumeResponse{}, nil
	}

	// Return success if we don't need to do anything.
	if status, ok := cs.volumeStatusMap.readStatus(vol.volumeID); ok && status == statusDeleted {
		return &csi.DeleteVolumeResponse{}, nil
	}

	// Obtain exclusive control over the volume.
	if !cs.volumeIDsInFlight.obtainLockOnString(vol.volumeID) {
		return nil, status.Errorf(codes.Aborted, "volumeID %s is in use by another request; check BeeGFS network "+
			"configuration if this problem persists", vol.volumeID)
	}
	defer cs.volumeIDsInFlight.releaseLockOnString(vol.volumeID)

	// Prepare to clean up.
	defer func() {
		// Clean up no matter what and return an error if cleanup fails. Ignoring cleanup failure might lead to silent
		// orphaned mounts. One occasional cause of cleanup failure is a mount reporting "busy" on attempted unmount
		// immediately after a directory or file deletion. Another DeleteVolume call resolves the issue.
		if cleanupErr := unmountAndCleanUpIfNecessary(ctx, vol, true, cs.mounter); cleanupErr != nil {
			// err and resp are named values that were being returned by DeleteVolume. We can modify them here to
			// return something different.
			resp = nil
			if err != nil {
				// Instead of overwriting the returned GrpcError, let's just log a separate error here.
				LogError(ctx, err, "Failed to clean up path for volume", "path", vol.mountDirPath, "volumeID", vol.volumeID)
			} else {
				err = newGrpcErrorFromCause(codes.Internal, cleanupErr)
			}
		}
		if err == nil {
			// Only record success if both deletion and cleanup are successful. This allows a subsequent DeleteVolume
			// request to attempt a failed cleanup again.
			cs.volumeStatusMap.writeStatus(vol.volumeID, statusDeleted)
		}
	}()

	// Write configuration files and mount BeeGFS.
	if err = fs.MkdirAll(vol.mountDirPath, 0750); err != nil {
		err = errors.WithStack(err)
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}
	if err = writeClientFiles(ctx, vol, cs.clientConfTemplatePath); err != nil {
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}
	if err = mountIfNecessary(ctx, vol, []string{}, cs.mounter); err != nil {
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}

	// Delete volume from mounted BeeGFS.
	if err = deleteVolumeUntilWait(ctx, vol, cs.nodeUnstageTimeout); err != nil {
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *controllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return &csi.ControllerGetCapabilitiesResponse{Capabilities: getControllerServiceCapabilities(controllerCaps)}, nil
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	// Check arguments.
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}
	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not provided")
	}

	// Validate parameters if provided.
	params := req.GetParameters()
	if len(params) != 0 {
		_, err := validateReqParams(params)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "%s", err)
		}
	}

	// Construct an internal representation of the volume and ensure no other request is currently referencing it.
	vol, err := cs.newBeegfsVolumeFromID(volumeID)
	if err != nil {
		err = errors.WithMessage(err, "volume ID is invalid or the volume does not exist")
		return nil, newGrpcErrorFromCause(codes.NotFound, err)
	}
	if !cs.volumeIDsInFlight.obtainLockOnString(vol.volumeID) {
		return nil, status.Errorf(codes.Aborted, "volumeID %s is in use by another request; check BeeGFS network "+
			"configuration if this problem persists", vol.volumeID)
	}
	defer cs.volumeIDsInFlight.releaseLockOnString(vol.volumeID)

	// Write configuration files but do not mount BeeGFS.
	defer func() {
		// Failure to clean up is an internal problem. The CO only cares whether or not the volume exists.
		// Cleanup failures may leave directories and/or files but no orphaned mounts because no mounts are involved.
		if err := cleanUpIfNecessary(ctx, vol, true); err != nil {
			LogError(ctx, err, "Failed to clean up path for volume", "path", vol.mountDirPath, "volumeID", vol.volumeID)
		}
	}()
	if err := fs.MkdirAll(vol.mountDirPath, 0750); err != nil {
		err = errors.WithStack(err)
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}
	if err := writeClientFiles(ctx, vol, cs.clientConfTemplatePath); err != nil {
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}

	if _, err := cs.ctlExec.statDirectoryForVolume(ctx, vol, vol.volDirPathBeegfsRoot); err != nil {
		if errors.As(err, &ctlNotExistError{}) {
			return nil, newGrpcErrorFromCause(codes.NotFound, err)
		}
		return nil, newGrpcErrorFromCause(codes.Internal, err)
	}

	confirmed, reason := isValidVolumeCapabilities(volCaps)
	if confirmed {
		return &csi.ValidateVolumeCapabilitiesResponse{
			Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
				// VolumeContext: req.GetVolumeContext(),  // Our volumes do not include a context.
				VolumeCapabilities: volCaps,
				// We rely on validateReqParams to ensure all parameters are valid. If we make it to this point
				// parameters are valid.
				Parameters: params,
			},
		}, nil
	}
	return &csi.ValidateVolumeCapabilitiesResponse{
		Message: reason,
	}, nil
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
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}
	// While currently volume "capacity" has no meaning as far as the driver is concerned, some
	// applications rely on the capacity of the PV/PVC in the K8s API to make certain decisions.
	// For these applications it is helpful to support volume resizing.
	return &csi.ControllerExpandVolumeResponse{
		CapacityBytes:         req.CapacityRange.RequiredBytes,
		NodeExpansionRequired: false,
	}, nil
}

func (cs *controllerServer) ControllerGetVolume(ctx context.Context, in *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) ControllerModifyVolume(ctx context.Context, in *csi.ControllerModifyVolumeRequest) (*csi.ControllerModifyVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// getControllerServiceCapabilities will convert a slice of ControllerServiceCapability_RPC_Type entries to a slice
// of ControllerServiceCapability structs. This makes it easier to define a set of ControllerServiceCapabilities by Type
// and then pass them on with the more complicated ControllerServiceCapability structure.
func getControllerServiceCapabilities(cl []csi.ControllerServiceCapability_RPC_Type) []*csi.ControllerServiceCapability {
	var csc []*csi.ControllerServiceCapability
	for _, cap := range cl {
		LogDebug(context.TODO(), "Adding controller service capability", "capability", cap.String())
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

// getStripePatternConfigFromParams parses a map of parameters and sets stripePatternConfig variables, if provided.
// Once a parameter is found it is verified, set, and then deleted from the original map. This is to help validateReqParams narrow
// down if extra parameters exist. getStripePatternConfigFromParams then returns a stripePatternConfig object and the
// original map excluding stripePattern parameters.
func getStripePatternConfigFromParams(reqParams map[string]string) (stripePatternConfig, map[string]string, error) {
	cfg := stripePatternConfig{}
	for param := range reqParams {
		if strings.Contains(param, "stripePattern/") {
			switch param {
			case stripePatternStoragePoolIDKey:
				cfg.storagePoolID = reqParams[stripePatternStoragePoolIDKey]
				// Validate storagePoolID is an integer only.
				if cfg.storagePoolID != "" {
					_, err := strconv.ParseUint(cfg.storagePoolID, 10, 16)
					if err != nil {
						return cfg, nil, errors.Wrap(err, "could not parse provided StoragePoolID")
					}
				}
				delete(reqParams, stripePatternStoragePoolIDKey)
			case stripePatternChunkSizeKey:
				cfg.stripePatternChunkSize = reqParams[stripePatternChunkSizeKey]
				// Validate StripePatternChunkSize has only digits followed by a single upper or lowercase letter.
				if cfg.stripePatternChunkSize != "" {
					r := regexp.MustCompile("(^[0-9]+[a-zA-Z]$)")
					matched := r.MatchString(cfg.stripePatternChunkSize)
					if !matched {
						return cfg, nil, errors.New("could not parse provided chunkSize")
					}
				}
				delete(reqParams, stripePatternChunkSizeKey)
			case stripePatternNumTargetsKey:
				cfg.stripePatternNumTargets = reqParams[stripePatternNumTargetsKey]
				// Validate stripePatternNumTargets value is an integer.
				if cfg.stripePatternNumTargets != "" {
					_, err := strconv.ParseUint(cfg.stripePatternNumTargets, 10, 16)
					if err != nil {
						return cfg, nil, errors.Wrap(err, "could not parse provided numTargets")
					}
				}
				delete(reqParams, stripePatternNumTargetsKey)
			default:
				return cfg, nil, errors.Errorf("CreateVolume parameter invalid: %s", param)
			}
		}
	}
	return cfg, reqParams, nil
}

// getPermissionsConfigFromParams parses a map of parameters and sets permissionsConfig variables, if provided.
// Once a parameters is found and set, it is deleted from the original map. This is to help validateReqParams narrow
// down if extra parameters exist. getPermissionsConfigFromParams then returns a permissionsConfig object and the
// original map excluding permissionsConfig parameters.
func getPermissionsConfigFromParams(reqParams map[string]string) (permissionsConfig, map[string]string, error) {
	cfg := permissionsConfig{mode: defaultPermissionsMode}
	for param := range reqParams {
		if strings.HasPrefix(param, "permissions/") {
			switch param {
			case permissionsUIDKey:
				val := reqParams[permissionsUIDKey]
				uid, err := strconv.ParseUint(val, 10, 32) // UIDs are <= 32 bits.
				if err != nil {
					return cfg, nil, errors.Wrap(err, "could not parse provided UID")
				}
				cfg.uid = uint32(uid)
				delete(reqParams, permissionsUIDKey)
			case permissionsGIDKey:
				val := reqParams[permissionsGIDKey]
				gid, err := strconv.ParseUint(val, 10, 32) // GIDs are <= 32 bits.
				if err != nil {
					return cfg, nil, errors.Wrap(err, "could not parse provided GID")
				}
				cfg.gid = uint32(gid)
				delete(reqParams, permissionsGIDKey)
			case permissionsModeKey:
				val := reqParams[permissionsModeKey]
				mode, err := strconv.ParseUint(val, 8, 12) // Full modes are 12 bits.
				if err != nil {
					return cfg, nil, errors.Wrap(err, "could not parse provided mode")
				}
				cfg.mode = uint16(mode)
				delete(reqParams, permissionsModeKey)
			default:
				return cfg, nil, errors.Errorf("CreateVolume parameter invalid: %s", param)
			}
		}
	}
	return cfg, reqParams, nil
}

// (*controllerServer) newBeegfsVolume is a wrapper around newBeegfsVolume that makes it easier to call in the context
// of the controller service. (*controllerServer) newBeegfsVolume selects the mountDirPath and passes the controller
// service's PluginConfig.
func (cs *controllerServer) newBeegfsVolume(sysMgmtdHost, volDirBasePathBeegfsRoot, volName string) beegfsVolume {
	volDirPathBeegfsRoot := path.Join(volDirBasePathBeegfsRoot, volName)
	// This volumeID construction duplicates the one further down in the stack. We do it anyway to generate an
	// appropriate mountDirPath.
	volumeID := NewBeegfsURL(sysMgmtdHost, volDirPathBeegfsRoot)
	mountDirPath := path.Join(cs.csDataDir, sanitizeVolumeID(volumeID)) // e.g. /csDataDir/127.0.0.1_scratch_pvc-12345678
	return newBeegfsVolume(mountDirPath, sysMgmtdHost, volDirPathBeegfsRoot, cs.pluginConfig)
}

// (*controllerServer) newBeegfsVolumeFromID is a wrapper around newBeegfsVolumeFromID that makes it easier to call in
// the context of the controller service. (*controllerServer) newBeegfsVolumeFromID selects the mountDirPath and passes
// the controller service's PluginConfig.
func (cs *controllerServer) newBeegfsVolumeFromID(volumeID string) (beegfsVolume, error) {
	mountDirPath := path.Join(cs.csDataDir, sanitizeVolumeID(volumeID)) // e.g. /csDataDir/127.0.0.1_scratch_pvc-12345678
	return newBeegfsVolumeFromID(mountDirPath, volumeID, cs.pluginConfig)
}

func deleteVolumeUntilWait(ctx context.Context, vol beegfsVolume, waitTime uint64) error {
	start := time.Now()
	nodesPath := path.Join(vol.csiDirPath, "nodes")
	for {
		dirExists, err := fsutil.DirExists(nodesPath)
		if err != nil {
			// For some unknown reason, we couldn't check for the existence of the .csi/volumes/volume/nodes directory.
			return errors.WithStack(err)
		} else if dirExists {
			isEmpty, err := fsutil.IsEmpty(nodesPath)
			if err != nil {
				// For some unknown reason, we couldn't attempt to read from the .csi/volumes/volume/nodes directory.
				return errors.WithStack(err)
			} else if !isEmpty {
				if time.Since(start) < time.Duration(waitTime)*time.Second {
					// We found the .csi/volumes/volume/nodes/ directory, but it isn't yet empty and we're willing to wait.
					secondsLeft := int64((time.Duration(waitTime)*time.Second - time.Since(start)).Seconds())
					LogVerbose(ctx, "Waiting for volume to unstage from all nodes",
						"secondsLeft", secondsLeft, "volumeID", vol.volumeID)
					time.Sleep(time.Duration(2) * time.Second)
					continue // Wait for the next loop to do anything else.
				} else {
					// The .csi/volumes/volume/nodes directory is not empty, but we're no longer willing to wait.
					// If an error occurs reading the directory entries, just log an empty list of remaining nodes.
					remainingNodeInfos, _ := fsutil.ReadDir(nodesPath)
					var remainingNodeNames []string
					for _, fileInfo := range remainingNodeInfos {
						remainingNodeNames = append(remainingNodeNames, fileInfo.Name())
					}
					LogDebug(ctx, "Volume did not unstage on all nodes; orphan mounts may remain",
						"remainingNodes", remainingNodeNames, "volumeID", vol.volumeID)
				}
			}
			// Whether the .csi/volumes/volume/nodes/ directory is empty or we're done waiting, we should delete it.
			LogDebug(ctx, "Deleting BeeGFS directory", "path", vol.csiDirPathBeegfsRoot, "volumeID", vol.volumeID)
			if err = fsutil.RemoveAll(vol.csiDirPath); err != nil {
				return errors.WithStack(err)
			}
			break // Go on to delete the volume.
		} else {
			// It's fine if the .csi/volumes/volume/nodes directory does not exist. It was likely never created in the
			// first place. We'll just fall back to naive deletion behavior.
			LogVerbose(ctx, "No node tracking information found", "path", vol.csiDirPathBeegfsRoot, "volumeID", vol.volumeID)
			break // Go on to delete the volume.
		}
	}
	// Now it's time to delete the volume itself.
	LogDebug(ctx, "Deleting BeeGFS directory", "path", vol.volDirPathBeegfsRoot, "volumeID", vol.volumeID)
	if err := fs.RemoveAll(vol.volDirPath); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// validateReqParams validates plugin specific parameters if provided. If we find an expected parameter, we initialize
// the reqParameters struct with its corresponding value. We also verify that all parameters being passed
// are valid, with no extra parameters. Each time we set a desired value to reqParameters we delete it from the
// params map narrowing down any extra parameters.
func validateReqParams(params map[string]string) (reqParams reqParameters, err error) {

	sysMgmtdHost, ok := params[sysMgmtdHostKey]
	if !ok || sysMgmtdHost == "" {
		return reqParameters{}, errors.New("sysMgmtdHost not provided")
	}
	reqParams.sysMgmtdHost = sysMgmtdHost
	delete(params, sysMgmtdHostKey)

	volDirBasePathBeegfsRoot, ok := params[volDirBasePathKey]
	if !ok {
		return reqParameters{}, errors.New("volDirBasePath not provided")
	}
	reqParams.volDirBasePathBeegfsRoot = volDirBasePathBeegfsRoot
	reqParams.volDirBasePathBeegfsRoot = path.Clean(path.Join("/", volDirBasePathBeegfsRoot))
	delete(params, volDirBasePathKey)

	stripePatternConfig, params, err := getStripePatternConfigFromParams(params)
	if err != nil {
		return reqParameters{}, err
	}
	reqParams.volStripePatternConfig = stripePatternConfig

	volPermissionsConfig, params, err := getPermissionsConfigFromParams(params)
	if err != nil {
		return reqParameters{}, err
	}
	reqParams.volPermissionsConfig = volPermissionsConfig

	// If extra parameters remain in params, return error and the parameters that remain.
	if len(params) != 0 {
		return reqParameters{}, errors.Errorf("CreateVolume parameter invalid: %s", params)
	}
	return reqParams, nil
}
