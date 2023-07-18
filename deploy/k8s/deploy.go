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

import (
	"bytes"
	_ "embed" // This is required for the //go:embed directive (https://pkg.go.dev/embed#hdr-Strings_and_Bytes).

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

//go:embed bases/csi-beegfs-rbac.yaml
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
// without understanding that operator code must be refactored.
const (
	KeyNameConfigMap = "csi-beegfs-config.yaml"
	KeyNameSecret    = "csi-beegfs-connauth.yaml"
)

// GetRBAC returns a slice of pointers to the following RBAC objects as interfaces: Cluster Role, Cluster Role Binding,
// Role, Role Binding, Service Account. GetRBAC returns an error if it finds a different kind of object or if an object
// cannot be correctly unmarshalled. The caller MUST assert the type of each object in the slice before using it. This
// approach allows us to add additional Roles, Role Bindings, etc. to the deployment manifests without reworking
// GetRBAC or the dependent operator code.
func GetRBAC() ([]interface{}, error) {
	var objects []interface{}
	var cr *rbacv1.ClusterRole
	var crb *rbacv1.ClusterRoleBinding
	var r *rbacv1.Role
	var rb *rbacv1.RoleBinding
	var sa *corev1.ServiceAccount

	// cs-beegfs-rbac.yaml includes multiple YAML documents, each with a different structure.
	splitRBACBytes := bytes.Split(rbacBytes, []byte("---"))

	for _, singleRBACBytes := range splitRBACBytes {
		// Consider Cluster Role Binding first because a typical Cluster Role Binding includes a reference to a
		// Cluster Role, but not vice versa.
		if bytes.Contains(singleRBACBytes, []byte("kind: ClusterRoleBinding")) {
			crb = new(rbacv1.ClusterRoleBinding)
			if err := yaml.UnmarshalStrict(singleRBACBytes, crb); err != nil {
				return objects, err
			}
			objects = append(objects, crb)
		} else if bytes.Contains(singleRBACBytes, []byte("kind: ClusterRole")) {
			cr = new(rbacv1.ClusterRole)
			if err := yaml.UnmarshalStrict(singleRBACBytes, cr); err != nil {
				return objects, err
			}
			objects = append(objects, cr)
			// Consider Role Binding first because a typical Role Binding includes a reference to a Role, but not vice
			// versa.
		} else if bytes.Contains(singleRBACBytes, []byte("kind: RoleBinding")) {
			rb = new(rbacv1.RoleBinding)
			if err := yaml.UnmarshalStrict(singleRBACBytes, rb); err != nil {
				return objects, err
			}
			objects = append(objects, rb)
		} else if bytes.Contains(singleRBACBytes, []byte("kind: Role")) {
			r = new(rbacv1.Role)
			if err := yaml.UnmarshalStrict(singleRBACBytes, r); err != nil {
				return objects, err
			}
			objects = append(objects, r)
		} else if bytes.Contains(singleRBACBytes, []byte("kind: ServiceAccount")) {
			sa = new(corev1.ServiceAccount)
			if err := yaml.UnmarshalStrict(singleRBACBytes, sa); err != nil {
				return objects, err
			}
			objects = append(objects, sa)
		} else {
			return objects, errors.New("unexpected document found in RBAC manifest")
		}
	}
	return objects, nil
}

// GetControllerServiceStatefulSet returns the embedded byte slice as a Stateful Set object representing the controller
// service.
func GetControllerServiceStatefulSet() (*appsv1.StatefulSet, error) {
	sts := new(appsv1.StatefulSet)
	err := yaml.UnmarshalStrict(csBytes, sts)
	return sts, err
}

// GetCSIDriver returns the embedded byte slice as a CSI Driver object.
func GetCSIDriver() (*storagev1.CSIDriver, error) {
	d := new(storagev1.CSIDriver)
	err := yaml.UnmarshalStrict(driverBytes, d)
	return d, err
}

// GetNodeServiceDaemonSet returns the embedded byte slice as a Daemon Set object representing the node service.
func GetNodeServiceDaemonSet() (*appsv1.DaemonSet, error) {
	ds := new(appsv1.DaemonSet)
	err := yaml.UnmarshalStrict(dsBytes, ds)
	return ds, err
}
