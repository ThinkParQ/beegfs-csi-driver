/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package deploy

import (
	"testing"
)

func TestGetControllerServiceRBAC(t *testing.T) {
	cr, crb, sa, err := GetControllerServiceRBAC()
	if err != nil {
		t.Fatal(err)
	}
	if cr == nil {
		t.Fatal("no Cluster Role in RBAC manifest")
	}
	if crb == nil {
		t.Fatal("no Cluster Role Binding in RBAC manifest")
	}
	if sa == nil {
		t.Fatal("no Service Account in RBAC manifest")
	}
	if numSubjects := len(crb.Subjects); numSubjects > 1 {
		t.Fatalf("the operator expects only 1 subject in a Cluster Role Binding, found %d", numSubjects)
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
