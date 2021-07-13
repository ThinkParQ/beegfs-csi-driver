/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package deploy

import (
	"testing"
)

func TestGetControllerServiceRBAC(t *testing.T) {
	if _, _, _, err := GetControllerServiceRBAC(); err != nil {
		t.Fatal(err)
	}
}

func TestGetControllerServiceStatefulSet(t *testing.T) {
	if _, err := GetControllerServiceStatefulSet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetCSIDriver(t *testing.T) {
	if _, err := GetCSIDriver(); err != nil {
		t.Fatal(err)
	}
}

func TestGetNodeServiceDaemonSet(t *testing.T) {
	if _, err := GetNodeServiceDaemonSet(); err != nil {
		t.Fatal(err)
	}
}
