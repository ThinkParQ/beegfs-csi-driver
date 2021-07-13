/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package deploy

// This file uses the Golang embed directive to package the base YAML deployment manifests into byte slices that can
// be accessed by importing code. For example, the operator package uses the manifests to instantiate the base
// Kubernetes objects (Stateful Set, Daemon Set, etc.) it deploys. This allows us to make MOST deployment changes
// once, in easy to manipulate YAML, and expect that both the Kustomize deployment method and the operator deployment
// method will pick the changes up.

// TODO(webere, A237): Duplicate the 1.18 Kustomizations in operator code. This change should be made elsewhere, but
// this is an easy place to reference it.

import (
	"bytes"
	_ "embed"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"sigs.k8s.io/yaml" // The "standard" gopkg.in/yaml.v2 decoder does not work for Kubernetes objects.
)

//go:embed base/csi-beegfs-controller.yaml
var csBytes []byte

//go:embed base/csi-beegfs-driverinfo.yaml
var driverBytes []byte

//go:embed base/csi-beegfs-node.yaml
var dsBytes []byte

//go:embed base/csi-beegfs-controller-rbac.yaml
var rbacBytes []byte

func GetControllerServiceRBAC() (*rbacv1.ClusterRole, *rbacv1.ClusterRoleBinding, *corev1.ServiceAccount, error) {
	cr := new(rbacv1.ClusterRole)
	crb := new(rbacv1.ClusterRoleBinding)
	sa := new(corev1.ServiceAccount)

	// cs-beegfs-rbac.yaml includes multiple YAML documents, each with a different structure.
	splitRBACBytes := bytes.Split(rbacBytes, []byte("---"))

	for _, singleRBACBytes := range splitRBACBytes {
		// Consider ClusterRoleBinding first because a typical ClusterRoleBinding includes a reference to a
		// ClusterRole, but not vice versa.
		if bytes.Contains(singleRBACBytes, []byte("ClusterRoleBinding")) {
			if err := yaml.UnmarshalStrict(singleRBACBytes, crb); err != nil {
				return cr, crb, sa, err
			}
		} else if bytes.Contains(singleRBACBytes, []byte("ClusterRole")) {
			if err := yaml.UnmarshalStrict(singleRBACBytes, cr); err != nil {
				return cr, crb, sa, err
			}
		} else if bytes.Contains(singleRBACBytes, []byte("ServiceAccount")) {
			if err := yaml.UnmarshalStrict(singleRBACBytes, sa); err != nil {
				return cr, crb, sa, err
			}
		} else {
			return cr, crb, sa, errors.New("unexpected document found in RBAC manifest")
		}
	}
	return cr, crb, sa, nil
}

func GetControllerServiceStatefulSet() (*appsv1.StatefulSet, error) {
	cs := new(appsv1.StatefulSet)
	err := yaml.UnmarshalStrict(csBytes, cs)
	return cs, err
}

func GetCSIDriver() (*storagev1.CSIDriver, error) {
	driver := new(storagev1.CSIDriver)
	err := yaml.UnmarshalStrict(driverBytes, driver)
	return driver, err
}

func GetNodeServiceDaemonSet() (*appsv1.DaemonSet, error) {
	ds := new(appsv1.DaemonSet)
	err := yaml.UnmarshalStrict(dsBytes, ds)
	return ds, err
}
