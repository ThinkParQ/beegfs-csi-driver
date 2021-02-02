package beegfs

import (
	"reflect"
	"testing"
)

func TestConstructSetPatternForVolume(t *testing.T) {
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
			got, needToExecute := constructSetPatternForVolume(tc.config)
			if !reflect.DeepEqual(tc.wantArgs, got) {
				t.Fatalf("expected: %s, got: %s", tc.wantArgs, got)
			}
			if needToExecute != tc.wantToExecute {
				t.Fatalf("want to execute: %t, need to execute: %t", tc.wantToExecute, needToExecute)
			}
		})
	}
}
