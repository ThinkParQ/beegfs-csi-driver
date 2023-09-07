/*
Copyright 2022 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package utils

import (
	"context"
	"strings"

	beegfsv1 "github.com/netapp/beegfs-csi-driver/operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	e2eframework "k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	"sigs.k8s.io/yaml"
)

var beegfsDriverResource = schema.GroupVersionResource{Group: "beegfs.csi.netapp.com", Version: "v1", Resource: "beegfsdrivers"}

// GetBeegfsDriverInUse returns a pointer to a BeegfsDriver if one and only one exists on the cluster. Consuming tests fail if:
//   - The cluster knows about the BeegfsDriver resource but has non.
//   - The cluster has more than one BeegfsDriver.
//   - The attempt to list BeegfsDrivers fails for an unknown reason.
//
// GetBeegfsDriverInUse returns nil if the cluster doesn't know about BeegfsDriver resources.
func GetBeegfsDriverInUse(dc dynamic.Interface) *beegfsv1.BeegfsDriver {
	// Use the client-go dynamic client to get a list of unstructured objects that match the BeegfsDriver schema.
	beegfsDriverList, err := dc.Resource(beegfsDriverResource).Namespace("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil // The BeegfsDriver API resource doesn't exist. Hopefully we're in a non-operator cluster.
		}
		e2eframework.ExpectNoError(err)
	}
	e2eframework.ExpectEqual(len(beegfsDriverList.Items), 1, "expected exactly one BeegfsDriver on an operator-enabled cluster")

	// Turn the unstructured object into a BeegfsDriver.
	beegfsDriver := new(beegfsv1.BeegfsDriver)
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(beegfsDriverList.Items[0].Object, beegfsDriver)
	e2eframework.ExpectNoError(err)
	return beegfsDriver
}

// GetConfigMapInUse returns the ConfigMap being used to configure the BeeGFS CSI driver. Consuming tests fail if:
//   - The BeeGFS CSI driver controller service isn't using a ConfigMap (this is virtually impossible).
//   - The ConfigMap can't be retrieved.
func GetConfigMapInUse(cs clientset.Interface) corev1.ConfigMap {
	// There may be many old ConfigMaps on the cluster. We need to get the one actually being used.
	controllerPod := GetRunningControllerPodOrFail(cs)
	var configMapName string
	for _, volume := range controllerPod.Spec.Volumes {
		if volume.ConfigMap != nil && strings.HasPrefix(volume.ConfigMap.Name, "csi-beegfs-config") {
			configMapName = volume.ConfigMap.Name
			break
		}
	}
	if configMapName == "" {
		e2eframework.Fail("no appropriate ConfigMap in controller service Pod spec")
	}
	configMap, err := cs.CoreV1().ConfigMaps(controllerPod.Namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	e2eframework.ExpectNoError(err, "failed to get ConfigMap")
	return *configMap
}

// GetPluginConfigInUse returns a PluginConfigFromFile from the BeegfsDriver if a cluster is operator-enabled or the
// ConfigMap otherwise. Consuming tests fail if this isn't possible.
func GetPluginConfigInUse(cs clientset.Interface, dc dynamic.Interface) beegfsv1.PluginConfigFromFile {
	// If the cluster is operator-enabled, we can't change its ConfigMap directly. We'll get its configuration from
	// the BeegfsDriver.
	beegfsDriver := GetBeegfsDriverInUse(dc)
	if beegfsDriver != nil {
		return beegfsDriver.Spec.PluginConfigFromFile
	}

	configMap := GetConfigMapInUse(cs)
	var config beegfsv1.PluginConfigFromFile
	configBytes := []byte(configMap.Data["csi-beegfs-config.yaml"])
	err := yaml.UnmarshalStrict(configBytes, &config)
	e2eframework.ExpectNoError(err, "failed to parse csi-beegfs-config.yaml from ConfigMap")
	return config
}

// UpdatePluginConfigInUse updates the BeegfsDriver (if it exists) or the ConfigMap with a new PluginConfigFromFile.
// Consuming tests fail if this does not work.
func UpdatePluginConfigInUse(cs clientset.Interface, dc dynamic.Interface, config beegfsv1.PluginConfigFromFile) {
	// Archive Pod logs so we don't lose them.
	err := ArchiveServiceLogs(cs, e2eframework.TestContext.ReportDir)
	// Fail test to ensure visibility of missing service logs.
	e2eframework.ExpectNoError(err, "expected to successfully archive service logs")

	beegfsDriver := GetBeegfsDriverInUse(dc)
	if beegfsDriver != nil {
		beegfsDriver.Spec.PluginConfigFromFile = config
		beegfsDriverMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(beegfsDriver)
		e2eframework.ExpectNoError(err)
		beegfsDriverUnstructured := unstructured.Unstructured{Object: beegfsDriverMap}
		_, err = dc.Resource(beegfsDriverResource).Namespace(beegfsDriver.Namespace).Update(context.TODO(),
			&beegfsDriverUnstructured, metav1.UpdateOptions{})
		e2eframework.ExpectNoError(err)
		return // The operator will take it from here.
	}

	configMap := GetConfigMapInUse(cs)
	configBytes, err := yaml.Marshal(config)
	e2eframework.ExpectNoError(err, "failed to marshal config into csi-beegfs-config.yaml")
	configString := string(configBytes)
	configMap.Data["csi-beegfs-config.yaml"] = configString
	_, err = cs.CoreV1().ConfigMaps(configMap.Namespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
	e2eframework.ExpectNoError(err)

	// Delete Pods so they can pick up the changes to the ConfigMap. Don't use the "other" e2epod.Delete functions
	// because they wait for the Pod to no longer exist. That will not happen here, as the StatefulSet and DaemonSet
	// controllers will recreate them.
	controllerPod := GetRunningControllerPodOrFail(cs)
	e2epod.DeletePodOrFail(cs, controllerPod.Namespace, controllerPod.Name)
	nodePods := GetRunningNodePodsOrFail(cs)
	for _, nodePod := range nodePods {
		e2epod.DeletePodOrFail(cs, nodePod.Namespace, nodePod.Name)
	}

	// Wait for Pods to be running again.
	_ = GetRunningControllerPodOrFail(cs)
	_ = GetRunningNodePodsOrFail(cs)
}
