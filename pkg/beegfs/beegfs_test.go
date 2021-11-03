/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package beegfs

import (
	"path"
	"reflect"
	"testing"

	beegfsv1 "github.com/netapp/beegfs-csi-driver/operator/api/v1"
)

func TestHasNonDefaultOwnerOrGroup(t *testing.T) {
	tests := map[string]struct {
		cfg  permissionsConfig
		want bool
	}{
		"default": {
			cfg:  permissionsConfig{},
			want: false,
		},
		"non-default UID": {
			cfg:  permissionsConfig{uid: 1000},
			want: true,
		},
		"non-default GID": {
			cfg:  permissionsConfig{gid: 1000},
			want: true,
		},
		"non-default both": {
			cfg:  permissionsConfig{uid: 1000, gid: 1000},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.cfg.hasNonDefaultOwnerOrGroup()
			if !tc.want == got {
				t.Fatalf("expected: %t, got: %t", tc.want, got)
			}
		})
	}
}

func TestHasSpecialPermissions(t *testing.T) {
	tests := map[string]struct {
		cfg  permissionsConfig
		want bool
	}{
		"no special permissions": {
			cfg:  permissionsConfig{mode: 0777},
			want: false,
		},
		"special permissions": {
			cfg:  permissionsConfig{mode: 2777},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.cfg.hasSpecialPermissions()
			if !tc.want == got {
				t.Fatalf("expected: %t, got: %t", tc.want, got)
			}
		})
	}
}

func TestGoFileMode(t *testing.T) {
	tests := map[string]struct {
		config       permissionsConfig
		goModeString string
	}{
		"no bits": {
			config:       permissionsConfig{mode: 0o0777},
			goModeString: "-rwxrwxrwx",
		},
		"sticky bit": {
			config:       permissionsConfig{mode: 0o1777},
			goModeString: "trwxrwxrwx",
		},
		"setgid bit": {
			config:       permissionsConfig{mode: 0o2777},
			goModeString: "grwxrwxrwx",
		},
		"setuid bit": {
			config:       permissionsConfig{mode: 0o4777},
			goModeString: "urwxrwxrwx",
		},
		"all bits": {
			config:       permissionsConfig{mode: 0o7777},
			goModeString: "ugtrwxrwxrwx",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.config.goFileMode()
			if tc.goModeString != got.String() {
				t.Fatalf("expected: %s, got: %s", tc.goModeString, got.String())
			}
		})
	}
}

func TestNewBeegfsVolume(t *testing.T) {
	// Inputs are based on comments in the example preceding the beegfsVolume struct in beegfs.go.
	want := beegfsVolume{
		config:                   *beegfsv1.NewBeegfsConfig(),
		clientConfPath:           path.Join("/", "...", "mountDirPath", "beegfs-client.conf"),
		csiDirPath:               path.Join("/", "...", "mountDirPath", "mount", "...", "parent", ".csi", "volumes", "volume"),
		csiDirPathBeegfsRoot:     path.Join("/", "...", "parent", ".csi", "volumes", "volume"),
		mountDirPath:             path.Join("/", "...", "mountDirPath"),
		mountPath:                path.Join("/", "...", "mountDirPath", "mount"),
		sysMgmtdHost:             "sysMgmtdHost",
		volDirBasePathBeegfsRoot: path.Join("/", "...", "parent"),
		volDirBasePath:           path.Join("/", "...", "/", "mountDirPath", "mount", "...", "parent"),
		volDirPath:               path.Join("/", "...", "mountDirPath", "mount", "...", "parent", "volume"),
		volDirPathBeegfsRoot:     path.Join("/", "...", "parent", "volume"),
		volumeID:                 NewBeegfsUrl("sysMgmtdHost", path.Join("/", "...", "parent", "volume")),
	}
	got := newBeegfsVolume(path.Join("/", "...", "mountDirPath"),
		"sysMgmtdHost",
		path.Join("/", "...", "parent", "volume"),
		beegfsv1.PluginConfig{})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("\nexpected: \n%+v, \ngot: \n%+v", want, got)
	}
}
