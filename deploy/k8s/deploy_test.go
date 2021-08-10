/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package deploy

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
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
	var sts *appsv1.StatefulSet
	var err error

	if sts, err = GetControllerServiceStatefulSet(); err != nil {
		t.Fatal(err)
	}

	// Ensure that expected containers with expected names exist in the embedded manifest. Some operator logic depends
	// on these names.
	for _, containerName := range []string{ContainerNameBeegfsCsiDriver, ContainerNameCsiProvisioner} {
		foundContainer := false
		for _, container := range sts.Spec.Template.Spec.Containers {
			if container.Name == containerName {
				foundContainer = true
				break
			}
		}
		if !foundContainer {
			t.Fatalf("expected a container named %s in Stateful Set", containerName)
		}
	}
}

func TestGetCSIDriver(t *testing.T) {
	if _, err := GetCSIDriver(); err != nil {
		t.Fatal(err)
	}
}

func TestGetNodeServiceDaemonSet(t *testing.T) {
	var ds *appsv1.DaemonSet
	var err error

	if ds, err = GetNodeServiceDaemonSet(); err != nil {
		t.Fatal(err)
	}

	// Ensure that expected containers with expected names exist in the embedded manifest. Some operator logic depends
	// on these names.
	for _, containerName := range []string{ContainerNameBeegfsCsiDriver, ContainerNameLivenessProbe, ContainerNameCsiNodeDriverRegistrar} {
		foundContainer := false
		for _, container := range ds.Spec.Template.Spec.Containers {
			if container.Name == containerName {
				foundContainer = true
				break
			}
		}
		if !foundContainer {
			t.Fatalf("expected a container named %s in Stateful Set", containerName)
		}
	}
}
