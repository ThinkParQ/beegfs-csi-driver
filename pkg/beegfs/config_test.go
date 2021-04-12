/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package beegfs

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// TestParseConfig makes use of .yaml files in the testdata directory. Numerical values in these files attempt to
// follow a convention in which 0 represents a value that is unmodified (e.g. 8000 or 127.0.0.0), 1 represents a value
// that has been modified once (e.g. 8001 or 127.0.0.1), etc.
func TestParseConfig(t *testing.T) {
	fs = afero.NewOsFs()
	fsutil = afero.Afero{Fs: fs}
	tests := map[string]struct {
		configFile string
		nodeID     string
		want       PluginConfig
	}{
		"basic all fields correct": {
			configFile: "testdata/basic.yaml",
			nodeID:     "testnode",
			want: PluginConfig{
				DefaultConfig: beegfsConfig{
					ConnInterfaces:    []string{"ib0"},
					ConnNetFilter:     []string{"127.0.0.0/24"},
					ConnTcpOnlyFilter: []string{"127.0.0.0"},
					BeegfsClientConf:  map[string]string{"connMgmtdPort": "8000"},
				},
				FileSystemSpecificConfigs: []FileSystemSpecificConfig{
					{
						SysMgmtdHost: "127.0.0.0",
						Config: beegfsConfig{
							ConnInterfaces:    []string{"ib0"},
							ConnNetFilter:     []string{"127.0.0.0/24"},
							ConnTcpOnlyFilter: []string{"127.0.0.0"},
							BeegfsClientConf:  map[string]string{"connMgmtdPort": "8000"},
						},
					},
				},
			},
		},
		"node default override (matching nodeid) all fields correct": {
			// because "testnode" is in nodeList, default values should be overridden
			configFile: "testdata/node-default-override.yaml",
			nodeID:     "testnode",
			want: PluginConfig{
				DefaultConfig: beegfsConfig{
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
			want: PluginConfig{
				DefaultConfig: beegfsConfig{
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
			want: PluginConfig{
				DefaultConfig: beegfsConfig{
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
			want: PluginConfig{
				DefaultConfig: beegfsConfig{
					ConnInterfaces:    []string{"ib0"},
					ConnNetFilter:     []string{"127.0.0.0/24"},
					ConnTcpOnlyFilter: []string{"127.0.0.0"},
					BeegfsClientConf:  map[string]string{"connMgmtdPort": "8000"},
				},
				FileSystemSpecificConfigs: []FileSystemSpecificConfig{
					{
						SysMgmtdHost: "127.0.0.1",
						Config: beegfsConfig{
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

func TestValidateConfig(t *testing.T) {
	basicConfig, err := parseConfigFromFile("testdata/basic.yaml", "testnode")
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]struct {
		expectedError error
		config        PluginConfig
	}{
		"basic config passes validation": {
			nil,
			basicConfig,
		},
		"sysMgmtdHost with domain name": {
			nil,
			PluginConfig{
				FileSystemSpecificConfigs: []FileSystemSpecificConfig{
					{
						SysMgmtdHost: "subdomain.somewebsite.com",
					},
				},
			},
		},
		"invalid sysMgmtdHost": {
			errors.New("invalid SysMgmtdHost testinvalid"),
			PluginConfig{
				FileSystemSpecificConfigs: []FileSystemSpecificConfig{
					{
						SysMgmtdHost: "testinvalid",
					},
				},
			},
		},
		"invalid connNetFilter": {
			errors.New("invalid ConnNetFilter testinvalid"),
			PluginConfig{
				FileSystemSpecificConfigs: []FileSystemSpecificConfig{
					{
						SysMgmtdHost: "127.0.0.0",
						Config: beegfsConfig{
							ConnNetFilter: []string{"testinvalid"},
						},
					},
				},
			},
		},
		"invalid ConnTCPOnlyFilter": {
			errors.New("invalid ConnTCPOnlyFilter testinvalid"),
			PluginConfig{
				DefaultConfig: beegfsConfig{
					ConnTcpOnlyFilter: []string{"127.0.0.0", "testinvalid"},
				},
				FileSystemSpecificConfigs: []FileSystemSpecificConfig{
					{
						SysMgmtdHost: "127.0.0.0",
					},
				},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.config.validateConfig()
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
	modifiedConfig.stripConfig()
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
	modifiedConfig.stripConfig()
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
	modifiedConfig.stripConfig()
	if !reflect.DeepEqual(originalConfig, modifiedConfig) {
		t.Fatalf("stripConfig() performed unexpected modification. Original: %v, Stripped: %v",
			originalConfig, modifiedConfig)
	}
}

func TestOverwriteFromBeegfsClientConfEmptyValue(t *testing.T) {
	writeTo := beegfsConfig{
		BeegfsClientConf: map[string]string{
			"setKey": "setValue",
		},
	}
	writeFrom := beegfsConfig{
		BeegfsClientConf: map[string]string{
			"setKey": "",
		},
	}
	want := beegfsConfig{
		BeegfsClientConf: map[string]string{
			"setKey": "",
		},
	}
	writeTo.overwriteFrom(writeFrom)
	if !reflect.DeepEqual(want, writeTo) {
		t.Fatalf("expected: %v, got: %v", want, writeTo)
	}
}
