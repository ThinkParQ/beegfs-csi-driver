/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package beegfs

import (
	"path"
	"reflect"
	"regexp"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	beegfsv1 "github.com/netapp/beegfs-csi-driver/operator/api/v1"
	"github.com/spf13/afero"
	"golang.org/x/net/context"
)

// This is included here as a constant for formatting reasons (literal looks better with no indentation involved).
const TestWriteClientFilesTemplate = `# A minimal configuration file that allows connInterfaces, connNetFilter, 
# connTcpOnlyFilter, and one arbitrary override.
sysMgmtdHost          =
connClientPortUDP     =
# One arbitrary key that can be overridden.
connMgmtdPortTCP      = 8008
connInterfacesFile    =
connNetFilterFile     =
connTcpOnlyFilterFile =
connAuthFile          =
connRDMAInterfacesFile =
`

// We cannot predict what connClientUDPPort will be chosen, so tests shouldn't actually check that line.
const TestWriteClientFilesBeegfsClientConf = `# A minimal configuration file that allows connInterfaces, connNetFilter,
# connTcpOnlyFilter, and one arbitrary override.
sysMgmtdHost           = 127.0.0.1
connClientPortUDP      = 49152
# One arbitrary key that can be overridden.
connMgmtdPortTCP       = 8000
connInterfacesFile     = /testvol/connInterfacesFile
connNetFilterFile      = /testvol/connNetFilterFile
connTcpOnlyFilterFile  = /testvol/connTcpOnlyFilterFile
connAuthFile           = /testvol/connAuthFile
connRDMAInterfacesFile = /testvol/connRDMAInterfacesFile
`

func TestNewBeegfsUrl(t *testing.T) {
	tests := map[string]struct {
		host, path string
		want       string
	}{
		"basic ip example": {
			host: "127.0.0.1",
			path: "/path/to/volume",
			want: "beegfs://127.0.0.1/path/to/volume",
		},
		"basic FQDN example": {
			host: "some.domain.com",
			path: "/path/to/volume",
			want: "beegfs://some.domain.com/path/to/volume",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := NewBeegfsURL(tc.host, tc.path)
			if tc.want != got {
				t.Fatalf("expected: %s, got: %s", tc.want, got)
			}
		})
	}
}

func TestParseBeegfsUrl(t *testing.T) {
	tests := map[string]struct {
		rawURL             string
		wantHost, wantPath string
		wantErr            bool
	}{
		"basic ip example": {
			rawURL:   "beegfs://127.0.0.1/path/to/volume",
			wantHost: "127.0.0.1",
			wantPath: "/path/to/volume",
			wantErr:  false,
		},
		"basic FQDN example": {
			rawURL:   "beegfs://some.domain.com/path/to/volume",
			wantHost: "some.domain.com",
			wantPath: "/path/to/volume",
			wantErr:  false,
		},
		"invalid URL example": {
			rawURL:   "beegfs:// some.domain.com/ path/to/volume",
			wantHost: "",
			wantPath: "",
			wantErr:  true,
		},
		"invalid https example": {
			rawURL:   "https://some.domain.com/path/to/volume",
			wantHost: "",
			wantPath: "",
			wantErr:  true,
		},
		"invalid empty string example": {
			rawURL:   "",
			wantHost: "",
			wantPath: "",
			wantErr:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotHost, gotPath, err := parseBeegfsURL(tc.rawURL)
			if tc.wantHost != gotHost {
				t.Fatalf("expected host: %s, got host: %s", tc.wantHost, gotHost)
			}

			if tc.wantPath != gotPath {
				t.Fatalf("expected path: %s, got path: %s", tc.wantPath, gotPath)
			}

			if tc.wantErr == true && err == nil {
				t.Fatalf("expected an error to occur for invalid URL: %s", tc.rawURL)
			}

			if tc.wantErr == false && err != nil {
				t.Fatalf("expected no error to occur: %v", err)
			}
		})
	}
}

func TestWriteClientFiles(t *testing.T) {
	fs = afero.NewMemMapFs() // test sets up its own, new, memory-mapped file system
	fsutil = afero.Afero{Fs: fs}
	confTemplateDirPath := "/etc/beegfs"
	confTemplatePath := path.Join(confTemplateDirPath, "beegfs-client.conf")
	mountDirPath := "/testvol"
	sysMgmtdHost := "127.0.0.1"
	testConfig := beegfsv1.PluginConfig{
		DefaultConfig: beegfsv1.BeegfsConfig{
			ConnInterfaces:     []string{"ib0"},
			ConnNetFilter:      []string{"127.0.0.0/24"},
			ConnTcpOnlyFilter:  []string{"127.0.0.0"},
			ConnRDMAInterfaces: []string{"ib0"},
			BeegfsClientConf: map[string]string{
				"connMgmtdPortTCP": "8000",
			},
		},
		FileSystemSpecificConfigs: []beegfsv1.FileSystemSpecificConfig{
			{
				SysMgmtdHost: sysMgmtdHost,
				Config: beegfsv1.BeegfsConfig{
					ConnAuth: "secret1",
				},
			},
		},
	}
	wantConnAuthFile := "secret1"              // desired connAuthFile contents
	wantConnInterfacesFile := "ib0\n"          // desired connInterfacesFile contents
	wantConnNetFilterFile := "127.0.0.0/24\n"  // desired connNetFilterFile contents
	wantConnTcpOnlyFilterFile := "127.0.0.0\n" // desired connTcpOnlyFilterFile contents
	wantConnRDMAInterfacesFile := "ib0\n"      // desired connRDMAInterfacesFile contents

	// set up template directory in memory-mapped filesystem
	if err := fs.MkdirAll(confTemplateDirPath, 0755); err != nil {
		t.Fatalf("failed to set up configuration template directory: %v", err)
	}
	if err := fsutil.WriteFile(confTemplatePath, []byte(TestWriteClientFilesTemplate), 0644); err != nil {
		t.Fatalf("failed to write template beegfs-client.conf: %v", err)
	}

	// set up conf directory in memory-mapped filesystem
	if err := fs.Mkdir(mountDirPath, 0755); err != nil {
		t.Fatalf("failed to set up new configuration directory: %v", err)
	}

	vol := newBeegfsVolume(mountDirPath, sysMgmtdHost, "test", testConfig)
	if err := writeClientFiles(context.Background(), vol, confTemplatePath); err != nil {
		t.Fatalf("expected no error to occur: %v", err)
	}

	// check written beegfs-client.conf
	got, err := fsutil.ReadFile(vol.clientConfPath)
	if err != nil {
		t.Errorf("could not read output beegfs-client.conf")
	}
	// We cannot predict what connClientUDPPort will be chosen, so we don't check that line.
	udpExpression := regexp.MustCompile(`connClientPortUDP\s*=\s\d*\n`)
	wantString := udpExpression.ReplaceAllString(TestWriteClientFilesBeegfsClientConf, "")
	gotString := udpExpression.ReplaceAllString(string(got), "")
	if wantString != gotString {
		t.Errorf("beegfs-client.conf does not match; expected:\n%vgot:\n%v",
			wantString, gotString)
	}

	// check written connAuthFile
	got, err = fsutil.ReadFile(path.Join(vol.mountDirPath, "connAuthFile"))
	if err != nil {
		t.Errorf("could not read output connAuthFile")
	}
	if wantConnAuthFile != string(got) {
		t.Errorf("connAuthFile does not match; expected:\n%vgot:\n%v",
			wantConnAuthFile, string(got))
	}

	// check written connInterfacesFile
	got, err = fsutil.ReadFile(path.Join(vol.mountDirPath, "connInterfacesFile"))
	if err != nil {
		t.Errorf("could not read output connInterfacesFile")
	}
	if wantConnInterfacesFile != string(got) {
		t.Errorf("connInterfacesFile does not match; expected:\n%vgot:\n%v",
			wantConnInterfacesFile, string(got))
	}

	// check written connNetFilterFile
	got, err = fsutil.ReadFile(path.Join(vol.mountDirPath, "connNetFilterFile"))
	if err != nil {
		t.Errorf("could not read output connNetFilterFile")
	}
	if wantConnNetFilterFile != string(got) {
		t.Errorf("connNetFilterFile does not match; expected:\n%vgot:\n%v",
			wantConnNetFilterFile, string(got))
	}

	// check written connRDMAInterfacesFile
	got, err = fsutil.ReadFile(path.Join(vol.mountDirPath, "connRDMAInterfacesFile"))
	if err != nil {
		t.Errorf("could not read output connRDMAInterfacesFile")
	}
	if wantConnRDMAInterfacesFile != string(got) {
		t.Errorf("connRDMAInterfacesFile does not match; exected\n%vgot:\n%v",
			wantConnRDMAInterfacesFile, string(got))
	}

	// check written connTcpOnlyFilterFile
	got, err = fsutil.ReadFile(path.Join(vol.mountDirPath, "connTcpOnlyFilterFile"))
	if err != nil {
		t.Errorf("could not read output connInterfacesFile")
	}
	if wantConnTcpOnlyFilterFile != string(got) {
		t.Errorf("connTcpOnlyFilterFile does not match; expected:\n%vgot:\n%v",
			wantConnTcpOnlyFilterFile, string(got))
	}
}

func TestSquashConfigForSysMgmtdHost(t *testing.T) {
	defaultConfig := *beegfsv1.NewBeegfsConfig()
	defaultConfig.ConnInterfaces = []string{"ib0"}
	fileSystemSpecificBeegfsConfig := *beegfsv1.NewBeegfsConfig()
	fileSystemSpecificBeegfsConfig.ConnInterfaces = []string{"ib1"}
	testConfig := beegfsv1.PluginConfig{
		DefaultConfig: defaultConfig,
		FileSystemSpecificConfigs: []beegfsv1.FileSystemSpecificConfig{
			{
				SysMgmtdHost: "127.0.0.1",
				Config:       fileSystemSpecificBeegfsConfig,
			},
		},
	}

	tests := map[string]struct {
		sysMgmtdHost string
		want         beegfsv1.BeegfsConfig
	}{
		"not matching sysMgmtdHost": {
			sysMgmtdHost: "127.0.0.0",
			want:         defaultConfig,
		},
		"matching sysMgmtdHost": {
			sysMgmtdHost: "127.0.0.1",
			want:         fileSystemSpecificBeegfsConfig,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := squashConfigForSysMgmtdHost(tc.sysMgmtdHost, testConfig)
			if !reflect.DeepEqual(tc.want, got) {
				t.Fatalf("expected BeegfsConfig: %v, got BeegfsConfig: %v", tc.want, got)
			}
		})
	}
}

func TestGetEphemeralPortUDP(t *testing.T) {
	_, err := getEphemeralPortUDP()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSanitizeVolumeID(t *testing.T) {
	tests := map[string]struct {
		provided string
		want     string
	}{
		"basic ip example": {
			provided: "beegfs://127.0.0.1/path/to/volume",
			want:     "127.0.0.1_path_to_volume",
		},
		"basic FQDN example": {
			provided: "beegfs://some.domain.com/path/to/volume",
			want:     "some.domain.com_path_to_volume",
		},
		"example with underscores": {
			provided: "beegfs://some.domain.com/path_with_underscores/to/volume",
			want:     "some.domain.com_path__with__underscores_to_volume",
		},
		"example with too many characters": {
			provided: "beegfs://some.domain.com/lots/of/characters/lots/of/characters/lots/of/characters/" +
				"lots/of/characters/lots/of/characters/lots/of/characters/lots/of/characters/lots/of/characters/" +
				"lots/of/characters/lots/of/characters/lots/of/characters/lots/of/characters/lots/of/characters",
			want: "20d02d3ce23bd842f5a9334f478c87c3f131e51e",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := sanitizeVolumeID(tc.provided)
			if tc.want != got {
				t.Fatalf("expected: %s, got: %s", tc.want, got)
			}
		})
	}
}

func TestIsValidVolumeCapabilities(t *testing.T) {
	tests := map[string]struct {
		caps      []*csi.VolumeCapability
		wantValid bool
	}{
		"single supported capability": {
			caps: []*csi.VolumeCapability{
				{
					AccessMode: &csi.VolumeCapability_AccessMode{ // all access modes are supported
						Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
					},
					AccessType: &csi.VolumeCapability_Mount{ // mount is supported
						Mount: &csi.VolumeCapability_MountVolume{},
					},
				},
			},
			wantValid: true,
		},
		"multiple supported capabilities": {
			caps: []*csi.VolumeCapability{
				{
					AccessMode: &csi.VolumeCapability_AccessMode{ // all access modes are supported
						Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
					},
					AccessType: &csi.VolumeCapability_Mount{ // mount is supported
						Mount: &csi.VolumeCapability_MountVolume{},
					},
				},
				{
					AccessMode: &csi.VolumeCapability_AccessMode{ // all access modes are supported
						Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY,
					},
					AccessType: &csi.VolumeCapability_Mount{ // mount is supported
						Mount: &csi.VolumeCapability_MountVolume{},
					},
				},
			},
			wantValid: true,
		},
		"unsupported capability": {
			caps: []*csi.VolumeCapability{
				{
					AccessMode: &csi.VolumeCapability_AccessMode{ // all access modes are supported
						Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
					},
					AccessType: &csi.VolumeCapability_Block{ // block is not supported
						Block: &csi.VolumeCapability_BlockVolume{},
					},
				},
			},
			wantValid: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotValid, reason := isValidVolumeCapabilities(tc.caps)
			if tc.wantValid != gotValid {
				t.Fatalf("expected: %t, got: %t, reason: %s", tc.wantValid, gotValid, reason)
			}
		})
	}
}

func TestAddContextToMountOptionsIfNecessary(t *testing.T) {
	tests := map[string]struct {
		input []string
		want  []string
	}{
		"input with context at the end": {
			input: []string{"some", "mount", "options", "context=some_user:some_role:some_type:some_level"},
			want:  []string{"some", "mount", "options", "context=some_user:some_role:some_type:some_level"},
		},
		"input with context at some arbitrary location": {
			input: []string{"some", "context=some_user:some_role:some_type:some_level", "mount", "options"},
			want:  []string{"some", "context=some_user:some_role:some_type:some_level", "mount", "options"},
		},
		"input without context": {
			input: []string{"some", "mount", "options"},
			want:  []string{"some", "mount", "options", "context=system_u:object_r:container_file_t:s0"},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := addContextToMountOptionsIfNecessary(tc.input)
			if !reflect.DeepEqual(tc.want, got) {
				t.Fatalf("expected: %v, got: %v", tc.want, got)
			}
		})
	}
}

func TestRemoveInvalidMountOptions(t *testing.T) {
	inputOpts := []string{"option1", "option1", "cfgFile", "option2"}
	expectedOutput := []string{"option1", "option2"}
	actualOutput := removeInvalidMountOptions(context.TODO(), inputOpts)
	if len(expectedOutput) != len(actualOutput) {
		t.Fatalf("removeInvalidMountOptions didn't produce expected output. Expected: %v\tActual: %v",
			expectedOutput, actualOutput)
	}
	for i := range expectedOutput {
		if expectedOutput[i] != actualOutput[i] {
			t.Fatalf("removeInvalidMountOptions didn't produce expected output. Expected: %v\tActual: %v",
				expectedOutput, actualOutput)
		}
	}
}
