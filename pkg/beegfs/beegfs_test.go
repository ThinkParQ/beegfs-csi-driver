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
	"github.com/spf13/afero"
)

// TestNewBeegfsDriver only tests various expected failure conditions. Some string parameters are not allowed to be
// empty. Many of these parameters are file paths. It is not possible to validate file paths because Linux allows
// paths with virtually all characters. However, we can ensure that a file exists at a path and can be read if it is
// supposed to.
func TestNewBeegfsDriver(t *testing.T) {
	fs = afero.NewMemMapFs()
	fsutil = afero.Afero{Fs: fs}

	const goodClientConfTemplatePath = "/goodClientConfTemplatePath"
	const badPermissionsClientConfTemplatePath = "/badPermissionsClientConfTemplatePath"

	type testCase struct {
		connAuthPath           string
		configPath             string
		csDataDir              string
		driverName             string
		endpoint               string
		nodeID                 string
		clientConfTemplatePath string
		version                string
		nodeUnstageTimeout     uint64
	}
	defaultTestCase := testCase{
		connAuthPath:           "", // Failure behavior tested in TestParseConnAuthFromFile.
		configPath:             "", // Failure behavior tested in TestParseConfigFromFile.
		csDataDir:              "/csDataDir",
		driverName:             "beegfs.csi.netapp.com",
		endpoint:               "/someEndpoint",
		nodeID:                 "node1",
		clientConfTemplatePath: goodClientConfTemplatePath,
	}

	_ = fsutil.WriteFile(goodClientConfTemplatePath, []byte{}, 0644)
	_ = fsutil.WriteFile(badPermissionsClientConfTemplatePath, []byte{}, 0100)

	tests := map[string]func() testCase{
		"no csDataDir": func() testCase {
			tc := defaultTestCase
			tc.csDataDir = ""
			return tc
		},
		"no endpoint": func() testCase {
			tc := defaultTestCase
			tc.endpoint = ""
			return tc
		},
		"no nodeID": func() testCase {
			tc := defaultTestCase
			tc.nodeID = ""
			return tc
		},
		"bad clientConfTemplatePath": func() testCase {
			tc := defaultTestCase
			tc.clientConfTemplatePath = "/badClientConfTemplatePath"
			return tc
		},
		"no clientConfTemplatePathAndNoDefault": func() testCase {
			tc := defaultTestCase
			tc.clientConfTemplatePath = ""
			return tc
		},
		// TODO(webere, A336): Add this test case when https://github.com/spf13/afero/issues/150 is resolved.
		// "bad permissions on clientConfTemplatePath": func() testCase {
		//	 tc := defaultTestCase
		//	 tc.clientConfTemplatePath = badPermissionsClientConfTemplatePath
		//	 t.Log("uid", os.Getuid())
		//	 return tc
		// },
	}

	for name, tcFunc := range tests {
		t.Run(name, func(t *testing.T) {
			tc := tcFunc()
			_, err := NewBeegfsDriver(tc.connAuthPath, tc.configPath, tc.csDataDir, tc.driverName, tc.endpoint,
				tc.nodeID, tc.clientConfTemplatePath, tc.version, tc.nodeUnstageTimeout)
			if err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

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

// Use the same wantBeegfsVolume for multiple NewBeegfsVolume tests.
var wantBeegfsVolume = beegfsVolume{
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

func TestNewBeegfsVolume(t *testing.T) {
	// Inputs are based on comments in the example preceding the beegfsVolume struct in beegfs.go.
	want := wantBeegfsVolume
	got := newBeegfsVolume(want.mountDirPath, want.sysMgmtdHost, want.volDirPathBeegfsRoot, beegfsv1.PluginConfig{})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("\nexpected: \n%+v, \ngot: \n%+v", want, got)
	}
}

func TestNewBeegfsVolumeFromID(t *testing.T) {
	want := wantBeegfsVolume
	got, err := newBeegfsVolumeFromID(want.mountDirPath, want.volumeID, beegfsv1.PluginConfig{})
	if err != nil {
		t.Fatalf("expected no error; got %v", err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("\nexpected: \n%+v, \ngot: \n%+v", want, got)
	}
}

func TestGetDefaultClientConfTemplatePath(t *testing.T) {
	fs = afero.NewMemMapFs()
	fsutil = afero.Afero{Fs: fs}

	// Should return an empty string if a file doesn't exist at any default path.
	if path := getDefaultClientConfTemplatePath(); path != "" {
		t.Fatalf("expected no valid default path; got %s", path)
	}

	// Should return a default path a file exists at one.
	const defaultPath = "/host/var/lib/kubelet/plugins/beegfs.csi.netapp.com/client/beegfs-client.conf"
	_ = fsutil.WriteFile(defaultPath, []byte{}, 0644)
	if path := getDefaultClientConfTemplatePath(); path != defaultPath {
		t.Fatalf("expected default path %s; got %s", defaultPath, path)
	}
}
