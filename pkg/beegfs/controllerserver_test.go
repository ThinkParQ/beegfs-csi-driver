/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package beegfs

import (
	"reflect"
	"testing"
)

func TestGetStripePatternParamsFromRequest(t *testing.T) {
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
				"stripePattern/storagePoolID": "2",
				"stripePattern/chunkSize":     "2m",
				"stripePattern/numTargets":    "4",
			},
			want: stripePatternConfig{
				storagePoolID:           "2",
				stripePatternChunkSize:  "2m",
				stripePatternNumTargets: "4",
			},
			wantErr: false,
		},
		"storagePoolIDKey example": {
			reqParams: map[string]string{
				"stripePattern/storagePoolID": "2",
				"stripePattern/chunkSize":     "",
				"stripePattern/numTargets":    "",
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
				"stripePattern/storagePoolID": "",
				"stripePattern/chunkSize":     "",
				"stripePattern/numTargets":    "4",
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
				"stripePattern/storagePoolID": "",
				"stripePattern/chunkSize":     "2m",
				"stripePattern/numTargets":    "",
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
			got, err := getStripePatternParamsFromRequest(tc.reqParams)
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
