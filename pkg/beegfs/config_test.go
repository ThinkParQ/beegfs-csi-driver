/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package beegfs

import (
	"reflect"
	"testing"

	beegfsv1 "github.com/netapp/beegfs-csi-driver/operator/api/v1"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// TestParseConfig makes use of .yaml files in the testdata directory. Numerical values in these files attempt to
// follow a convention in which 0 represents a value that is unmodified (e.g. 8000 or 127.0.0.0), 1 represents a value
// that has been modified once (e.g. 8001 or 127.0.0.1), etc.
func TestParseConfigFromFile(t *testing.T) {
	fs = afero.NewOsFs()
	fsutil = afero.Afero{Fs: fs}
	tests := map[string]struct {
		configFile string
		nodeID     string
		want       beegfsv1.PluginConfig
	}{
		"basic all fields correct": {
			configFile: "testdata/basic.yaml",
			nodeID:     "testnode",
			want: beegfsv1.PluginConfig{
				DefaultConfig: beegfsv1.BeegfsConfig{
					ConnInterfaces:    []string{"ib0"},
					ConnNetFilter:     []string{"127.0.0.0/24"},
					ConnTcpOnlyFilter: []string{"127.0.0.0"},
					BeegfsClientConf:  map[string]string{"connMgmtdPort": "8000", "connUseRDMA": "true"},
				},
				FileSystemSpecificConfigs: []beegfsv1.FileSystemSpecificConfig{
					{
						SysMgmtdHost: "127.0.0.0",
						Config: beegfsv1.BeegfsConfig{
							ConnInterfaces:    []string{"ib0"},
							ConnNetFilter:     []string{"127.0.0.0/24"},
							ConnTcpOnlyFilter: []string{"127.0.0.0"},
							BeegfsClientConf:  map[string]string{"connMgmtdPort": "8000", "connUseRDMA": "true"},
						},
					},
				},
			},
		},
		"node default override (matching nodeid) all fields correct": {
			// because "testnode" is in nodeList, default values should be overridden
			configFile: "testdata/node-default-override.yaml",
			nodeID:     "testnode",
			want: beegfsv1.PluginConfig{
				DefaultConfig: beegfsv1.BeegfsConfig{
					ConnInterfaces:    []string{"ib1"},
					ConnNetFilter:     []string{"127.0.0.1/24"},
					ConnTcpOnlyFilter: []string{"127.0.0.1"},
					BeegfsClientConf:  map[string]string{"connMgmtdPort": "8001"},
				},
			},
		},
		"node default override double (matching nodeid) all fields correct": {
			// because "testnode" is in nodeList, default values should be overridden, then overridden again
			configFile: "testdata/node-default-override-double.yaml",
			nodeID:     "testnode",
			want: beegfsv1.PluginConfig{
				DefaultConfig: beegfsv1.BeegfsConfig{
					ConnInterfaces:    []string{"ib2"},
					ConnNetFilter:     []string{"127.0.0.2/24"},
					ConnTcpOnlyFilter: []string{"127.0.0.2"},
					BeegfsClientConf:  map[string]string{"connMgmtdPort": "8002"},
				},
			},
		},
		"node default override (not matching nodeid) all fields correct": {
			// because "testnode" is NOT in nodeList, default values should NOT be overridden
			configFile: "testdata/node-default-override.yaml",
			nodeID:     "nottestnode",
			want: beegfsv1.PluginConfig{
				DefaultConfig: beegfsv1.BeegfsConfig{
					ConnInterfaces:    []string{"ib0"},
					ConnNetFilter:     []string{"127.0.0.0/24"},
					ConnTcpOnlyFilter: []string{"127.0.0.0"},
					BeegfsClientConf:  map[string]string{"connMgmtdPort": "8000"},
				},
			},
		},
		"node specific filesystem specific override": {
			// because "testnode" is in nodeList, file system specific values should be overridden
			configFile: "testdata/node-filesystem-override.yaml",
			nodeID:     "testnode",
			want: beegfsv1.PluginConfig{
				DefaultConfig: beegfsv1.BeegfsConfig{
					ConnInterfaces:    []string{"ib0"},
					ConnNetFilter:     []string{"127.0.0.0/24"},
					ConnTcpOnlyFilter: []string{"127.0.0.0"},
					BeegfsClientConf:  map[string]string{"connMgmtdPort": "8000"},
				},
				FileSystemSpecificConfigs: []beegfsv1.FileSystemSpecificConfig{
					{
						SysMgmtdHost: "127.0.0.1",
						Config: beegfsv1.BeegfsConfig{
							ConnInterfaces:    []string{"ib1"},
							ConnNetFilter:     []string{"127.0.0.1/24"},
							ConnTcpOnlyFilter: []string{"127.0.0.1"},
							BeegfsClientConf:  map[string]string{"connMgmtdPort": "8001"},
						},
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := parseConfigFromFile(tc.configFile, tc.nodeID)
			if err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(tc.want, got) {
				t.Fatalf("expected: %v, got: %v", tc.want, got)
			}
		})
	}
}

func TestConnAuthNotParsedFromConfig(t *testing.T) {
	_, err := parseConfigFromFile("testdata/basic-with-connauth.yaml", "testnode")
	if err == nil {
		t.Fatal("should fail to parse configuration file with connAuth information")
	}
}

func TestParseConnAuthFromFile(t *testing.T) {
	fs = afero.NewOsFs()
	fsutil = afero.Afero{Fs: fs}
	tests := map[string]struct {
		path        string
		startConfig beegfsv1.PluginConfig
		want        beegfsv1.PluginConfig
	}{
		"non-matching file system specific config": {
			path: "testdata/connauthfile.yaml",
			startConfig: beegfsv1.PluginConfig{
				FileSystemSpecificConfigs: []beegfsv1.FileSystemSpecificConfig{
					{
						SysMgmtdHost: "127.0.0.1",
						Config: beegfsv1.BeegfsConfig{
							BeegfsClientConf: map[string]string{"testkey": "testvalue"},
						},
					},
				},
			},
			want: beegfsv1.PluginConfig{
				FileSystemSpecificConfigs: []beegfsv1.FileSystemSpecificConfig{
					{
						SysMgmtdHost: "127.0.0.1",
						Config: beegfsv1.BeegfsConfig{
							BeegfsClientConf: map[string]string{"testkey": "testvalue"},
						},
					},
					{
						SysMgmtdHost: "127.0.0.0",
						Config: beegfsv1.BeegfsConfig{
							ConnAuth: "secret1",
						},
					},
				},
			},
		},
		"matching file system specific config and no default config": {
			path: "testdata/connauthfile.yaml",
			startConfig: beegfsv1.PluginConfig{
				FileSystemSpecificConfigs: []beegfsv1.FileSystemSpecificConfig{
					{
						SysMgmtdHost: "127.0.0.0",
						Config: beegfsv1.BeegfsConfig{
							BeegfsClientConf: map[string]string{"testkey": "testvalue"},
						},
					},
				},
			},
			want: beegfsv1.PluginConfig{
				FileSystemSpecificConfigs: []beegfsv1.FileSystemSpecificConfig{
					{
						SysMgmtdHost: "127.0.0.0",
						Config: beegfsv1.BeegfsConfig{
							BeegfsClientConf: map[string]string{"testkey": "testvalue"},
							ConnAuth:         "secret1",
						},
					},
				},
			},
		},
		"matching filesystem specific config and default config": {
			path: "testdata/connauthfile.yaml",
			startConfig: beegfsv1.PluginConfig{
				DefaultConfig: beegfsv1.BeegfsConfig{
					BeegfsClientConf: map[string]string{"testkey": "testvalue"},
				},
				FileSystemSpecificConfigs: []beegfsv1.FileSystemSpecificConfig{
					{
						SysMgmtdHost: "127.0.0.0",
						Config: beegfsv1.BeegfsConfig{
							BeegfsClientConf: map[string]string{"testkey": "testvalue"},
						},
					},
				},
			},
			want: beegfsv1.PluginConfig{
				DefaultConfig: beegfsv1.BeegfsConfig{BeegfsClientConf: map[string]string{"testkey": "testvalue"}},
				FileSystemSpecificConfigs: []beegfsv1.FileSystemSpecificConfig{
					{
						SysMgmtdHost: "127.0.0.0",
						Config: beegfsv1.BeegfsConfig{
							BeegfsClientConf: map[string]string{"testkey": "testvalue"},
							ConnAuth:         "secret1",
						},
					},
				},
			},
		},
		"nil pluginConfig": {
			path: "testdata/connauthfile.yaml",
			want: beegfsv1.PluginConfig{
				FileSystemSpecificConfigs: []beegfsv1.FileSystemSpecificConfig{
					{
						SysMgmtdHost: "127.0.0.0",
						Config: beegfsv1.BeegfsConfig{
							ConnAuth: "secret1",
						},
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := parseConnAuthFromFile(tc.path, &tc.startConfig)
			if err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(tc.want, tc.startConfig) {
				t.Fatalf("expected: %v, got: %v", tc.want, tc.startConfig)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	basicConfig, err := parseConfigFromFile("testdata/basic.yaml", "testnode")
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]struct {
		expectedError error
		config        beegfsv1.PluginConfig
	}{
		"basic config passes validation": {
			nil,
			basicConfig,
		},
		"sysMgmtdHost with domain name": {
			nil,
			beegfsv1.PluginConfig{
				FileSystemSpecificConfigs: []beegfsv1.FileSystemSpecificConfig{
					{
						SysMgmtdHost: "subdomain.somewebsite.com",
					},
				},
			},
		},
		"invalid sysMgmtdHost": {
			errors.New("invalid SysMgmtdHost testinvalid"),
			beegfsv1.PluginConfig{
				FileSystemSpecificConfigs: []beegfsv1.FileSystemSpecificConfig{
					{
						SysMgmtdHost: "testinvalid",
					},
				},
			},
		},
		"invalid connNetFilter": {
			errors.New("invalid ConnNetFilter testinvalid"),
			beegfsv1.PluginConfig{
				FileSystemSpecificConfigs: []beegfsv1.FileSystemSpecificConfig{
					{
						SysMgmtdHost: "127.0.0.0",
						Config: beegfsv1.BeegfsConfig{
							ConnNetFilter: []string{"testinvalid"},
						},
					},
				},
			},
		},
		"invalid ConnTCPOnlyFilter": {
			errors.New("invalid ConnTCPOnlyFilter testinvalid"),
			beegfsv1.PluginConfig{
				DefaultConfig: beegfsv1.BeegfsConfig{
					ConnTcpOnlyFilter: []string{"127.0.0.0", "testinvalid"},
				},
				FileSystemSpecificConfigs: []beegfsv1.FileSystemSpecificConfig{
					{
						SysMgmtdHost: "127.0.0.0",
					},
				},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateConfig(&tc.config)
			if (err != nil && tc.expectedError == nil) || (err == nil && tc.expectedError != nil) ||
				(err != nil && tc.expectedError != nil && err.Error() != tc.expectedError.Error()) {
				t.Fatalf("expected error: %v, got: %v", tc.expectedError, err)
			}
		})
	}
}

// Verifies that stripping a config removes any options marked as "no effect"
func TestStripNoEffectConfig(t *testing.T) {
	originalConfig, err := parseConfigFromFile("testdata/basic.yaml", "testnode")
	if err != nil {
		t.Fatal(err)
	}
	modifiedConfig, err := parseConfigFromFile("testdata/basic.yaml", "testnode")
	if err != nil {
		t.Fatal(err)
	}
	// introduce no-effect options in the default and filesystem configs
	for _, noEffectOption := range noEffectBeegfsConfOptions {
		modifiedConfig.DefaultConfig.BeegfsClientConf[noEffectOption] = "noeffectdefaultkey"
		modifiedConfig.FileSystemSpecificConfigs[0].Config.BeegfsClientConf[noEffectOption] = "noeffectfskey"
	}
	stripConfig(&modifiedConfig)
	if !reflect.DeepEqual(originalConfig, modifiedConfig) {
		t.Fatalf("stripConfig() did not strip correctly. Original: %v, Stripped: %v",
			originalConfig, modifiedConfig)
	}
}

// Verifies that stripping a config without unsupported or no-effect options does nothing to the config
func TestStripCleanConfig(t *testing.T) {
	originalConfig, err := parseConfigFromFile("testdata/basic.yaml", "testnode")
	if err != nil {
		t.Fatal(err)
	}
	modifiedConfig, err := parseConfigFromFile("testdata/basic.yaml", "testnode")
	if err != nil {
		t.Fatal(err)
	}
	stripConfig(&modifiedConfig)
	if !reflect.DeepEqual(originalConfig, modifiedConfig) {
		t.Fatalf("stripConfig() performed unexpected modification. Original: %v, Stripped: %v",
			originalConfig, modifiedConfig)
	}
}

// Verifies that stripping a config with unsupported options does not remove them
func TestStripUnsupportedConfig(t *testing.T) {
	originalConfig, err := parseConfigFromFile("testdata/basic.yaml", "testnode")
	if err != nil {
		t.Fatal(err)
	}
	modifiedConfig, err := parseConfigFromFile("testdata/basic.yaml", "testnode")
	if err != nil {
		t.Fatal(err)
	}
	// introduce unsupported options in the default and filesystem configs
	for _, unsupportedOption := range unsupportedBeegfsConfOptions {
		originalConfig.DefaultConfig.BeegfsClientConf[unsupportedOption] = "unsupporteddefaultkey"
		originalConfig.FileSystemSpecificConfigs[0].Config.BeegfsClientConf[unsupportedOption] = "unsupportedfskey"
		modifiedConfig.DefaultConfig.BeegfsClientConf[unsupportedOption] = "unsupporteddefaultkey"
		modifiedConfig.FileSystemSpecificConfigs[0].Config.BeegfsClientConf[unsupportedOption] = "unsupportedfskey"
	}
	stripConfig(&modifiedConfig)
	if !reflect.DeepEqual(originalConfig, modifiedConfig) {
		t.Fatalf("stripConfig() performed unexpected modification. Original: %v, Stripped: %v",
			originalConfig, modifiedConfig)
	}
}

func TestOverwriteFromBeegfsClientConfEmptyValue(t *testing.T) {
	writeTo := beegfsv1.BeegfsConfig{
		BeegfsClientConf: map[string]string{
			"setKey": "setValue",
		},
	}
	writeFrom := beegfsv1.BeegfsConfig{
		BeegfsClientConf: map[string]string{
			"setKey": "",
		},
	}
	want := beegfsv1.BeegfsConfig{
		BeegfsClientConf: map[string]string{
			"setKey": "",
		},
	}
	overWriteBeegfsConfig(&writeTo, writeFrom)
	if !reflect.DeepEqual(want, writeTo) {
		t.Fatalf("expected: %v, got: %v", want, writeTo)
	}
}
