/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package beegfs

import (
	"context"
	"path"
	"reflect"
	"testing"
	"time"

	v1 "github.com/netapp/beegfs-csi-driver/operator/api/v1"
	"github.com/spf13/afero"
)

func TestGetStripePatternConfigFromParams(t *testing.T) {
	tests := map[string]struct {
		reqParams map[string]string
		want      stripePatternConfig
		wantErr   bool
	}{
		"nothing example": {
			reqParams: map[string]string{},
			want: stripePatternConfig{
				storagePoolID:           "",
				stripePatternChunkSize:  "",
				stripePatternNumTargets: "",
			},
			wantErr: false,
		},
		"everything example": {
			reqParams: map[string]string{
				stripePatternStoragePoolIDKey: "2",
				stripePatternChunkSizeKey:     "2m",
				stripePatternNumTargetsKey:    "4",
			},
			want: stripePatternConfig{
				storagePoolID:           "2",
				stripePatternChunkSize:  "2m",
				stripePatternNumTargets: "4",
			},
			wantErr: false,
		},
		"stripePatternStoragePoolIDKey example": {
			reqParams: map[string]string{
				stripePatternStoragePoolIDKey: "2",
				stripePatternChunkSizeKey:     "",
				stripePatternNumTargetsKey:    "",
			},
			want: stripePatternConfig{
				storagePoolID:           "2",
				stripePatternChunkSize:  "",
				stripePatternNumTargets: "",
			},
			wantErr: false,
		},
		"stripePatternNumTargetsKey example": {
			reqParams: map[string]string{
				stripePatternStoragePoolIDKey: "",
				stripePatternChunkSizeKey:     "",
				stripePatternNumTargetsKey:    "4",
			},
			want: stripePatternConfig{
				storagePoolID:           "",
				stripePatternChunkSize:  "",
				stripePatternNumTargets: "4",
			},
			wantErr: false,
		},
		"stripePatternChunkSizeKey example": {
			reqParams: map[string]string{
				stripePatternStoragePoolIDKey: "",
				stripePatternChunkSizeKey:     "2m",
				stripePatternNumTargetsKey:    "",
			},
			want: stripePatternConfig{
				storagePoolID:           "",
				stripePatternChunkSize:  "2m",
				stripePatternNumTargets: "",
			},
			wantErr: false,
		},
		"wrong example": {
			reqParams: map[string]string{
				"stripePattern/storagepoolid": "2",
				"stripePattern/chunksize":     "2m",
				"stripePattern/numtargets":    "4",
			},
			want: stripePatternConfig{
				storagePoolID:           "",
				stripePatternChunkSize:  "",
				stripePatternNumTargets: "",
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := getStripePatternConfigFromParams(tc.reqParams)
			if !reflect.DeepEqual(tc.want, got) {
				t.Fatalf("expected: %s, got: %s", tc.want, got)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if tc.wantErr && err == nil {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}

func TestGetPermissionsConfigFromParams(t *testing.T) {
	tests := map[string]struct {
		reqParams map[string]string
		want      permissionsConfig
		wantErr   bool
	}{
		"no permissions/ parameters": {
			reqParams: map[string]string{},
			want:      permissionsConfig{mode: defaultPermissionsMode},
			wantErr:   false,
		},
		"acceptable uid": {
			reqParams: map[string]string{permissionsUIDKey: "1500"},
			want:      permissionsConfig{uid: 1500, mode: defaultPermissionsMode},
			wantErr:   false,
		},
		"too large uid": {
			reqParams: map[string]string{permissionsUIDKey: "4294967296"},
			wantErr:   true,
		},
		"unparseable uid": {
			reqParams: map[string]string{permissionsUIDKey: "strange_value"},
			wantErr:   true,
		},
		"acceptable gid": {
			reqParams: map[string]string{permissionsGIDKey: "1500"},
			want:      permissionsConfig{gid: 1500, mode: defaultPermissionsMode},
			wantErr:   false,
		},
		"too large gid": {
			reqParams: map[string]string{permissionsGIDKey: "4294967296"},
			wantErr:   true,
		},
		"unparseable gid": {
			reqParams: map[string]string{permissionsGIDKey: "strange_value"},
			wantErr:   true,
		},
		"smallest valid three digit octal mode": {
			reqParams: map[string]string{permissionsModeKey: "000"},
			want:      permissionsConfig{mode: 0o0000},
			wantErr:   false,
		},
		"smallest valid four digit octal mode": {
			reqParams: map[string]string{permissionsModeKey: "0000"},
			want:      permissionsConfig{mode: 0o0000},
			wantErr:   false,
		},
		"largest valid three digit octal mode": {
			reqParams: map[string]string{permissionsModeKey: "777"},
			want:      permissionsConfig{mode: 0o0777},
			wantErr:   false,
		},
		"largest valid four digit octal mode": {
			reqParams: map[string]string{permissionsModeKey: "7777"},
			want:      permissionsConfig{mode: 0o7777},
			wantErr:   false,
		},
		"arbitrary three digit octal mode": {
			reqParams: map[string]string{permissionsModeKey: "755"},
			want:      permissionsConfig{mode: 0o0755},
			wantErr:   false,
		},
		"arbitrary four digit octal mode": {
			reqParams: map[string]string{permissionsModeKey: "2755"},
			want:      permissionsConfig{mode: 0o2755},
			wantErr:   false,
		},
		"extra leading zeroes in octal mode": {
			reqParams: map[string]string{permissionsModeKey: "000000777"},
			want:      permissionsConfig{mode: 0o0777},
			wantErr:   false,
		},
		"non-octal numerical mode": {
			reqParams: map[string]string{permissionsModeKey: "888"},
			wantErr:   true,
		},
		"non-numerical mode": {
			reqParams: map[string]string{permissionsModeKey: "strange_Value"},
			wantErr:   true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := getPermissionsConfigFromParams(tc.reqParams)
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error occurred: %s", err)
			}
			if tc.wantErr && err == nil {
				t.Fatalf("expected error did not occur")
			}
			if !tc.wantErr && !reflect.DeepEqual(tc.want, got) {
				t.Fatalf("expected: %v, got: %v", tc.want, got)
			}
		})
	}
}

func TestDeleteVolumeUntilWaitEmptyNodesDir(t *testing.T) {
	fs = afero.NewMemMapFs() // Set up a new memory-mapped file system.
	fsutil = afero.Afero{Fs: fs}
	vol := newBeegfsVolume("mountDirPath", "sysMgmtdHost", "volDirPathBeegfsRoot", v1.PluginConfig{})
	nodesPath := path.Join(vol.csiDirPath, "nodes")
	if err := fs.MkdirAll(nodesPath, 0750); err != nil {
		t.Fatal("error in setup")
	}
	if err := fs.MkdirAll(vol.mountDirPath, 0777); err != nil {
		t.Fatal("error in setup")
	}

	if err := deleteVolumeUntilWait(context.TODO(), vol, 0); err != nil {
		t.Fatal("expected no error deleting volume")
	}
	if _, err := fs.Stat(vol.csiDirPath); err == nil {
		t.Fatalf("expected %s to be deleted", vol.csiDirPath)
	}
	if _, err := fs.Stat(vol.volDirPath); err == nil {
		t.Fatalf("expected %s to be deleted", vol.volDirPath)
	}
}

func TestDeleteVolumeUntilWaitNoCSIDir(t *testing.T) {
	fs = afero.NewMemMapFs() // Set up a new memory-mapped file system.
	fsutil = afero.Afero{Fs: fs}
	vol := newBeegfsVolume("mountDirPath", "sysMgmtdHost", "volDirPathBeegfsRoot", v1.PluginConfig{})
	if err := fs.MkdirAll(vol.mountDirPath, 0777); err != nil {
		t.Fatal("error in setup")
	}

	if err := deleteVolumeUntilWait(context.TODO(), vol, 0); err != nil {
		t.Fatal("expected no error deleting volume")
	}
	if _, err := fs.Stat(vol.volDirPath); err == nil {
		t.Fatalf("expected %s to be deleted", vol.volDirPath)
	}
}

func TestDeleteVolumeUntilWaitNodesDirNeverEmpties(t *testing.T) {
	fs = afero.NewMemMapFs() // Set up a new memory-mapped file system.
	fsutil = afero.Afero{Fs: fs}
	vol := newBeegfsVolume("mountDirPath", "sysMgmtdHost", "volDirPathBeegfsRoot", v1.PluginConfig{})
	nodesPath := path.Join(vol.csiDirPath, "nodes")
	if err := fs.MkdirAll(nodesPath, 0750); err != nil {
		t.Fatal("error in setup")
	}
	if err := fs.MkdirAll(vol.mountDirPath, 0777); err != nil {
		t.Fatal("error in setup")
	}

	if err := deleteVolumeUntilWait(context.TODO(), vol, 0); err != nil {
		t.Fatal("expected no error deleting volume")
	}
	if _, err := fs.Stat(vol.csiDirPath); err == nil {
		t.Fatalf("expected %s to be deleted", vol.csiDirPath)
	}
	if _, err := fs.Stat(vol.volDirPath); err == nil {
		t.Fatalf("expected %s to be deleted", vol.volDirPath)
	}
}

func TestDeleteVolumeUntilWaitNodesDirEmptiesEventually(t *testing.T) {
	fs = afero.NewMemMapFs() // Set up a new memory-mapped file system.
	fsutil = afero.Afero{Fs: fs}
	vol := newBeegfsVolume("mountDirPath", "sysMgmtdHost", "volDirPathBeegfsRoot", v1.PluginConfig{})
	nodesPath := path.Join(vol.csiDirPath, "nodes")
	nodeFile := path.Join(nodesPath, "node")
	if err := fs.MkdirAll(nodesPath, 0750); err != nil {
		t.Fatal("error in setup")
	}
	if _, err := fs.Create(nodeFile); err != nil {
		t.Fatal("error in setup")
	}
	if err := fs.MkdirAll(vol.mountDirPath, 0777); err != nil {
		t.Fatal("error in setup")
	}

	// Empty the nodes directory after a couple of seconds.
	emptyTime := time.Duration(2) * time.Second
	go func() {
		time.Sleep(emptyTime)
		_ = fsutil.Remove(nodeFile)
	}()

	start := time.Now()
	const waitTime = 10
	if err := deleteVolumeUntilWait(context.TODO(), vol, uint64(waitTime)); err != nil {
		t.Fatal("expected no error deleting volume")
	}
	if _, err := fs.Stat(vol.csiDirPath); err == nil {
		t.Fatalf("expected %s to be deleted", vol.csiDirPath)
	}
	if _, err := fs.Stat(vol.volDirPath); err == nil {
		t.Fatalf("expected %s to be deleted", vol.volDirPath)
	}
	if time.Since(start) > 2*emptyTime || time.Since(start) > waitTime*time.Second {
		t.Fatalf("expected delete to take ~%f seconds but it took %f", emptyTime.Seconds(), time.Since(start).Seconds())
	}
}
