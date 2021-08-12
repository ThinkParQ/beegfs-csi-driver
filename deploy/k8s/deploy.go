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

// TODO(webere, A263): Duplicate the 1.18 Kustomizations in operator code. This change should be made elsewhere, but
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

//go:embed bases/csi-beegfs-controller.yaml
var csBytes []byte

//go:embed bases/csi-beegfs-driverinfo.yaml
var driverBytes []byte

//go:embed bases/csi-beegfs-node.yaml
var dsBytes []byte

//go:embed bases/csi-beegfs-controller-rbac.yaml
var rbacBytes []byte

// These are expected container names within the Stateful Set and Daemon Set manifests. Some operator logic is based
// off the expectation that containers have these names. deploy_test.go attempts to ensure that a developer can not
// change these names without understanding that operator code must be refactored.
const (
	ContainerNameBeegfsCsiDriver        = "beegfs"
	ContainerNameCsiNodeDriverRegistrar = "node-driver-registrar"
	ContainerNameCsiProvisioner         = "csi-provisioner"
	ContainerNameLivenessProbe          = "liveness-probe"
)

// These are expected Kubernetes resource names within the manifests. Some operator logic is based off the expectation
// that resources have these names. deploy_test.go attempts to ensure that a developer can not change these names
// without understanding that operator code must be refactored.
const (
	ResourceNameConfigMap = "csi-beegfs-config"
	ResourceNameSecret    = "csi-beegfs-connauth"
)

// These are expected Config Map and Secret keys within the manifests. Some operator logic is based off the expectation
// that keys have these names. deploy_test.go attempts to ensure that a developer can not change these names
//without understanding that operator code must be refactored.
const (
	KeyNameConfigMap = "csi-beegfs-config.yaml"
	KeyNameSecret    = "csi-beegfs-connauth.yaml"
)

// GetControllerServiceRBAC returns a pointer to a Cluster Role, a pointer to a Cluster Role Binding, and a pointer
// to a Service Account contained in the embedded RBAC manifest. GetControllerServiceRBAC returns an error if it finds
// multiple of any of these object kinds or if it finds an object kind it does not expect. It returns a nil pointer
// for if it does not find an expected object.
func GetControllerServiceRBAC() (*rbacv1.ClusterRole, *rbacv1.ClusterRoleBinding, *corev1.ServiceAccount, error) {
	var cr *rbacv1.ClusterRole
	var crb *rbacv1.ClusterRoleBinding
	var sa *corev1.ServiceAccount

	// cs-beegfs-rbac.yaml includes multiple YAML documents, each with a different structure.
	splitRBACBytes := bytes.Split(rbacBytes, []byte("---"))

	for _, singleRBACBytes := range splitRBACBytes {
		// Consider ClusterRoleBinding first because a typical ClusterRoleBinding includes a reference to a
		// ClusterRole, but not vice versa.
		if bytes.Contains(singleRBACBytes, []byte("kind: ClusterRoleBinding")) {
			if crb != nil {
				return cr, crb, sa, errors.New("multiple Cluster Role Bindings in RBAC manifest")
			}
			crb = new(rbacv1.ClusterRoleBinding)
			if err := yaml.UnmarshalStrict(singleRBACBytes, crb); err != nil {
				return cr, crb, sa, err
			}
		} else if bytes.Contains(singleRBACBytes, []byte("kind: ClusterRole")) {
			if cr != nil {
				return cr, crb, sa, errors.New("multiple Cluster Roles in RBAC manifest")
			}
			cr = new(rbacv1.ClusterRole)
			if err := yaml.UnmarshalStrict(singleRBACBytes, cr); err != nil {
				return cr, crb, sa, err
			}
		} else if bytes.Contains(singleRBACBytes, []byte("kind: ServiceAccount")) {
			if sa != nil {
				return cr, crb, sa, errors.New("multiple Service Accounts in RBAC manifest")
			}
			sa = new(corev1.ServiceAccount)
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
	sts := new(appsv1.StatefulSet)
	err := yaml.UnmarshalStrict(csBytes, sts)
	return sts, err
}

func GetCSIDriver() (*storagev1.CSIDriver, error) {
	d := new(storagev1.CSIDriver)
	err := yaml.UnmarshalStrict(driverBytes, d)
	return d, err
}

func GetNodeServiceDaemonSet() (*appsv1.DaemonSet, error) {
	ds := new(appsv1.DaemonSet)
	err := yaml.UnmarshalStrict(dsBytes, ds)
	return ds, err
}
