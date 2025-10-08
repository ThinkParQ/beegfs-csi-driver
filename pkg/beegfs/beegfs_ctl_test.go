/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package beegfs

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
)

func TestConstructSetPatternForVolumeArgs(t *testing.T) {
	tests := map[string]struct {
		config        stripePatternConfig
		wantArgs      []string
		wantToExecute bool
		isV8          bool
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
		"everything example (v8)": {
			config: stripePatternConfig{
				storagePoolID:           "2",
				stripePatternChunkSize:  "2m",
				stripePatternNumTargets: "4",
			},
			wantArgs:      []string{"--mount=none", "entry", "set", "--pool=2", "--chunk-size=2m", "--num-targets=4"},
			wantToExecute: true,
			isV8:          true,
		},
		"nothing example (v8)": {
			config: stripePatternConfig{
				storagePoolID:           "",
				stripePatternChunkSize:  "",
				stripePatternNumTargets: "",
			},
			wantArgs:      []string{},
			wantToExecute: false,
			isV8:          true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, needToExecute := constructSetPatternForVolumeArgs(tc.config, tc.isV8)
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
		isV8     bool
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
		"all modified (v8)": {
			config: permissionsConfig{
				uid:  1000,
				gid:  1000,
				mode: 0o2755,
			},
			// For both v7 and v8 the leading digit in octal notation is always dropped.
			wantArgs: []string{"--mount=none", "entry", "create", "directory", "--permissions=755", "--uid=1000", "--gid=1000"},
			isV8:     true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := constructCreateDirForVolumeArgs(tc.config, tc.isV8)
			if !reflect.DeepEqual(tc.wantArgs, got) {
				t.Fatalf("expected: %s, got: %s", tc.wantArgs, got)
			}
		})
	}
}

func TestErrorsTypes(t *testing.T) {
	stdout := "stdOut"
	stderr := "stdErr"
	wantString := "beegfs-ctl failed with stdOut: stdOut and stdErr: stdErr"
	tests := map[string]struct {
		err       error
		errorType interface{}
	}{
		"ctlNotExistError": {
			err:       newCtlNotExistError(stdout, stderr),
			errorType: &ctlNotExistError{},
		},
		"ctlExistError": {
			err:       newCtlExistError(stdout, stderr),
			errorType: &ctlExistError{},
		},
		"ctlConnAuthError": {
			err:       newCtlConnAuthError(stdout, stderr),
			errorType: &ctlConnAuthError{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Any ctlError should output a similar string when Error() is called.
			gotString := tc.err.Error()
			if wantString != gotString {
				t.Fatalf("expected: %s, got: %s", wantString, gotString)
			}

			// Any ctlError should unwrap to its expected type.
			err := errors.Wrap(tc.err, "some wrapping message")
			if !errors.As(err, tc.errorType) {
				t.Fatalf("expected error to unwrap to type %s", name)
			}
		})
	}
}
