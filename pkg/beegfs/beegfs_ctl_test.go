/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package beegfs

import (
	"reflect"
	"testing"
)

func TestConstructSetPatternForVolumeArgs(t *testing.T) {
	tests := map[string]struct {
		config        stripePatternConfig
		wantArgs      []string
		wantToExecute bool
	}{
		"everything example": {
			config: stripePatternConfig{
				storagePoolID:           "2",
				stripePatternChunkSize:  "2m",
				stripePatternNumTargets: "4",
			},
			wantArgs:      []string{"--unmounted", "--setpattern", "--storagepoolid=2", "--chunksize=2m", "--numtargets=4"},
			wantToExecute: true,
		},
		"nothing example": {
			config: stripePatternConfig{
				storagePoolID:           "",
				stripePatternChunkSize:  "",
				stripePatternNumTargets: "",
			},
			wantArgs:      []string{},
			wantToExecute: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, needToExecute := constructSetPatternForVolumeArgs(tc.config)
			if !reflect.DeepEqual(tc.wantArgs, got) {
				t.Fatalf("expected: %s, got: %s", tc.wantArgs, got)
			}
			if needToExecute != tc.wantToExecute {
				t.Fatalf("want to execute: %t, need to execute: %t", tc.wantToExecute, needToExecute)
			}
		})
	}
}

func TestConstructCreateDirForVolumeArgs(t *testing.T) {
	tests := map[string]struct {
		config   permissionsConfig
		wantArgs []string
	}{
		"default": {
			config: permissionsConfig{
				mode: defaultPermissionsMode,
			},
			wantArgs: []string{"--unmounted", "--createdir", "--access=777"},
		},
		"uid modified": {
			config: permissionsConfig{
				uid:  1000,
				mode: defaultPermissionsMode,
			},
			wantArgs: []string{"--unmounted", "--createdir", "--access=777", "--uid=1000"},
		},
		"gid modified": {
			config: permissionsConfig{
				gid:  1000,
				mode: defaultPermissionsMode,
			},
			wantArgs: []string{"--unmounted", "--createdir", "--access=777", "--gid=1000"},
		},
		"mode modified": {
			config: permissionsConfig{
				mode: 0o0755,
			},
			wantArgs: []string{"--unmounted", "--createdir", "--access=755"},
		},
		"mode modified with special permissions": {
			config: permissionsConfig{
				mode: 0o2755,
			},
			// We explicitly do not use the leading digit in octal notation because beegfs-ctl ignores it anyway.
			wantArgs: []string{"--unmounted", "--createdir", "--access=755"},
		},
		"all modified": {
			config: permissionsConfig{
				uid:  1000,
				gid:  1000,
				mode: 0o2755,
			},
			// We explicitly do not use the leading digit in octal notation because beegfs-ctl ignores it anyway.
			wantArgs: []string{"--unmounted", "--createdir", "--access=755", "--uid=1000", "--gid=1000"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := constructCreateDirForVolumeArgs(tc.config)
			if !reflect.DeepEqual(tc.wantArgs, got) {
				t.Fatalf("expected: %s, got: %s", tc.wantArgs, got)
			}
		})
	}
}
