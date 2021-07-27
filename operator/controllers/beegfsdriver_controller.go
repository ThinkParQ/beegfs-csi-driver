/*
Copyright 2021 NetApp, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/netapp/beegfs-csi-driver/deploy"
	beegfsv1 "github.com/netapp/beegfs-csi-driver/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// BeegfsDriverReconciler reconciles a BeegfsDriver object
type BeegfsDriverReconciler struct {
	client.Client
	Log    logr.Logger // TODO(webere, A277): Reexamine the logger used.
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=beegfs.csi.netapp.com,resources=beegfsdrivers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=beegfs.csi.netapp.com,resources=beegfsdrivers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=beegfs.csi.netapp.com,resources=beegfsdrivers/finalizers,verbs=update

// The operator must have the following permissions to deploy the driver.
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=storage.k8s.io,resources=csidrivers,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update

// The operator must have the following permissions in order to grant them to the driver.
//+kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;update
//+kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=events,verbs=list;watch;create;update;patch
//+kubebuilder:rbac:groups=storage.k8s.io,resources=csinodes,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BeegfsDriverReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("beegfsDriver", req.NamespacedName)
	log.Info("Reconciling")

	driver := &beegfsv1.BeegfsDriver{}
	err := r.Get(ctx, req.NamespacedName, driver)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found and could have been deleted after reconcile request. Return and don't requeue.
			log.Info("BeegfsDriver resource not found. It has probably already been deleted.")
			return ctrl.Result{}, nil
		}
		// Failed to read request object. Requeue and try again.
		return ctrl.Result{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Start by getting everything we need to generate an up-to-date status, generating the up-to-date status, and
	// pushing the up-to-date status to the Kubernetes API server.

	sts := new(appsv1.StatefulSet)
	err = r.Get(ctx, types.NamespacedName{Name: "csi-beegfs-controller", Namespace: req.Namespace}, sts)
	statusCondition := metav1.Condition{}
	if err != nil {
		if !errors.IsNotFound(err) {
			// Something we aren't prepared for went wrong.
			return ctrl.Result{}, err
		} else {
			// We didn't find a Stateful Set.
			statusCondition = metav1.Condition{
				Type:    beegfsv1.ConditionControllerServiceReady,
				Status:  metav1.ConditionFalse,
				Reason:  beegfsv1.ReasonServiceNotCreated,
				Message: "controller service stateful set has not been created",
			}
		}
	} else {
		// We found a Stateful Set. Let's update our status based on its status.
		if sts.Status.Replicas < 1 {
			statusCondition = metav1.Condition{
				Type:    beegfsv1.ConditionControllerServiceReady,
				Status:  metav1.ConditionFalse,
				Reason:  beegfsv1.ReasonPodsNotScheduled,
				Message: "0/1 controller service pods have been scheduled",
			}
		} else if sts.Status.ReadyReplicas < 1 {
			statusCondition = metav1.Condition{
				Type:    beegfsv1.ConditionControllerServiceReady,
				Status:  metav1.ConditionFalse,
				Reason:  beegfsv1.ReasonPodsNotReady,
				Message: "0/1 controller service pods are ready",
			}
		} else {
			statusCondition = metav1.Condition{
				Type:    beegfsv1.ConditionControllerServiceReady,
				Status:  metav1.ConditionTrue,
				Reason:  beegfsv1.ReasonPodsReady,
				Message: "1/1 controller service pods are ready",
			}
		}
	}
	meta.SetStatusCondition(&driver.Status.Conditions, statusCondition)

	ds := new(appsv1.DaemonSet)
	err = r.Get(ctx, types.NamespacedName{Name: "csi-beegfs-node", Namespace: req.Namespace}, ds)
	statusCondition = metav1.Condition{}
	if err != nil {
		if !errors.IsNotFound(err) {
			// Something we aren't prepared for went wrong.
			return ctrl.Result{}, err
		} else {
			statusCondition = metav1.Condition{
				Type:    beegfsv1.ConditionNodeServiceReady,
				Status:  metav1.ConditionFalse,
				Reason:  beegfsv1.ReasonServiceNotCreated,
				Message: "node service daemon set has not been created",
			}
		}
	} else {
		// We found a Daemon Set. Let's update our status based on its status.
		if ds.Status.DesiredNumberScheduled < 1 {
			statusCondition = metav1.Condition{
				Type:    beegfsv1.ConditionNodeServiceReady,
				Status:  metav1.ConditionFalse,
				Reason:  beegfsv1.ReasonPodsNotScheduled,
				Message: "0 node service pods have been scheduled",
			}
		} else if ds.Status.NumberReady < ds.Status.DesiredNumberScheduled {
			statusCondition = metav1.Condition{
				Type:   beegfsv1.ConditionNodeServiceReady,
				Status: metav1.ConditionFalse,
				Reason: beegfsv1.ReasonPodsNotReady,
				Message: fmt.Sprintf("%d/%d node service pods are ready", ds.Status.NumberReady,
					ds.Status.DesiredNumberScheduled),
			}
		} else {
			statusCondition = metav1.Condition{
				Type:   beegfsv1.ConditionNodeServiceReady,
				Status: metav1.ConditionTrue,
				Reason: beegfsv1.ReasonPodsReady,
				Message: fmt.Sprintf("%d/%d node service pods are ready", ds.Status.NumberReady,
					ds.Status.DesiredNumberScheduled),
			}
		}
	}
	meta.SetStatusCondition(&driver.Status.Conditions, statusCondition)

	log.Info("Updating status")
	err = r.Status().Update(ctx, driver)
	if err != nil {
		return ctrl.Result{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Now attempt to get the rest of the expected objects and push them to the Kubernetes API server as necessary.

	// When managed by this operator, the Config Map is an internal implementation detail (it should not be externally
	// modified). Any time we reconcile, we ensure the Config Map contains exactly the information we expect.
	cm := new(corev1.ConfigMap)
	err = r.Get(ctx, types.NamespacedName{Name: "csi-beegfs-config", Namespace: req.Namespace}, cm)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err // Something we aren't prepared for went wrong.
		} else {
			// The Config Map doesn't exist. Let's create it.
			if cm, err = newConfigMap(driver); err != nil {
				return ctrl.Result{}, err
			}
			if _, err = r.setCommonObjectMetadata(req, driver, cm); err != nil { // We will create, not update.
				return ctrl.Result{}, err
			}

			log.Info("Creating Config Map")
			err = r.Create(ctx, cm)
			if err != nil {
				log.Error(err, "Failed to create Config Map")
				return ctrl.Result{}, err
			}
		}
	} else {
		mustUpdate, err := setConfigMapData(driver, cm)
		if err != nil {
			return ctrl.Result{}, err
		} else if mustUpdate {
			// The Config Map doesn't agree with the configuration in the BeegsDriver. The BeegfsDriver is the source of
			// truth in an operator-managed deployment. Let's update the Config Map.
			log.Info("Updating Config Map")
			err = r.Update(ctx, cm)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// TODO(webere, A259): Remove ability to handle arbitrarily named secret. Make sure we don't own a secret that we
	// didn't create.
	// A connauth Secret is required for driver operation. If a ConnAuthSecretName is provided in the BeegfsDriverSpec,
	// we make sure it owned by this controller and referenced appropriately. If one is not provided, we create an
	// empty Secret with a default name and reference it instead. Users can do one of the following:
	//   - Pre-create a Secret with the default name.
	//   - Pre-create a Secret with a different name and provide ConnAuthSecretName.
	//   - Update the default Secret (this is somewhat unintuitive, as it involves pasting a base64 encoded .yaml file).
	sName := "csi-beegfs-connauth" // This is the default name.
	if len(driver.Spec.ConnAuthSecretName) > 0 {
		sName = driver.Spec.ConnAuthSecretName
	}
	s := new(corev1.Secret)
	mustUpdate := false
	err = r.Get(ctx, types.NamespacedName{Name: sName, Namespace: req.Namespace}, s)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err // Something we aren't prepared for went wrong.
		} else {
			if sName == "csi-beegfs-connauth" {
				// We are using the default Secret name and no such Secret exists. Lets create an empty Secret.
				s = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: sName,
					},
					Data: map[string][]byte{"csi-beegfs-connauth.yaml": nil},
				}
				if _, err = r.setCommonObjectMetadata(req, driver, s); err != nil { // We will create, not update.
					return ctrl.Result{}, err
				}
				log.Info("Creating Secret")
				err = r.Create(ctx, s)
				if err != nil {
					log.Error(err, "Failed to create Secret")
					return ctrl.Result{}, err
				}
			} else {
				// We are using a non-default Secret name and no such Secret exists. Lets populate the name so other
				// things don't break in this reconcile loop. Eventually the controller and node Pods will fail to
				// start because the Secret is missing.
				s.Name = sName
			}
		}
	} else if sName != "csi-beegfs-connauth" {
		// We found the Secret using its non-default name. We may need to update its controller reference. If we don't
		// own the Secret, we won't be notified if it is updated.
		mustUpdate, err = r.setCommonObjectMetadata(req, driver, s)
		if err != nil {
			return ctrl.Result{}, err
		} else if mustUpdate {
			log.Info("Updating Secret")
			if err = r.Update(ctx, s); err != nil {
				log.Error(err, "Failed to update Secret")
				return ctrl.Result{}, err
			}
		}
	}

	// Get RBAC related default objects in case we need them.
	newCR, newCRB, newSA, err := deploy.GetControllerServiceRBAC()
	if err != nil {
		return ctrl.Result{}, err
	}

	cr := new(rbacv1.ClusterRole)
	err = r.Get(ctx, types.NamespacedName{Name: "csi-beegfs-provisioner-role"}, cr)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err // Something we aren't prepared for went wrong.
		} else {
			// The Cluster Role doesn't exist. Let's create it.
			if _, err = r.setCommonObjectMetadata(req, driver, newCR); err != nil { // We never update Cluster Roles.
				return ctrl.Result{}, err
			}

			log.Info("Creating controller service Cluster Role")
			err = r.Create(ctx, newCR)
			if err != nil {
				log.Error(err, "Failed to create controller service Cluster Role")
				return ctrl.Result{}, err
			}
		}
	}

	crb := new(rbacv1.ClusterRoleBinding)
	err = r.Get(ctx, types.NamespacedName{Name: "csi-beegfs-provisioner-binding"}, crb)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err // Something we aren't prepared for went wrong.
		} else {
			// The Cluster Role Binding doesn't exist. Let's create it.
			if _, err = r.setCommonObjectMetadata(req, driver, newCRB); err != nil { // We never update Cluster Role Bindings.
				return ctrl.Result{}, err
			}
			newCRB.Subjects[0].Namespace = req.Namespace

			log.Info("Creating controller service Cluster Role Binding")
			err = r.Create(ctx, newCRB)
			if err != nil {
				log.Error(err, "Failed to create controller service Cluster Role Binding")
				return ctrl.Result{}, err
			}
		}
	}

	sa := new(corev1.ServiceAccount)
	err = r.Get(ctx, types.NamespacedName{Name: "csi-beegfs-controller-sa", Namespace: req.Namespace}, sa)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err // Something we aren't prepared for went wrong.
		} else {
			// The Service Account doesn't exist. Let's create it.
			if _, err = r.setCommonObjectMetadata(req, driver, newSA); err != nil { // We never update Service Accounts.
				return ctrl.Result{}, err
			}

			log.Info("Creating controller service Service Account")
			err = r.Create(ctx, newSA)
			if err != nil {
				log.Error(err, "Failed to create controller service Service Account")
				return ctrl.Result{}, err
			}
		}
	}

	d := new(storagev1.CSIDriver)
	err = r.Get(ctx, types.NamespacedName{Name: "beegfs.csi.netapp.com", Namespace: req.Namespace}, d)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err // Something we aren't prepared for went wrong.
		} else {
			// The CSI Driver object doesn't exist. Let's create it.
			d, _ = deploy.GetCSIDriver()
			if _, err = r.setCommonObjectMetadata(req, driver, d); err != nil { // We never update CSI Driver objects.
				return ctrl.Result{}, err
			}

			log.Info("Creating CSI Driver object")
			err = r.Create(ctx, d)
			if err != nil {
				log.Error(err, "Failed to create CSI Driver object")
				return ctrl.Result{}, err
			}
		}
	}

	if meta.FindStatusCondition(driver.Status.Conditions, beegfsv1.ConditionControllerServiceReady).Reason ==
		beegfsv1.ReasonServiceNotCreated {
		// The Stateful Set doesn't exist. Let's create it.
		sts, _ = deploy.GetControllerServiceStatefulSet()
		if _, err = r.setCommonObjectMetadata(req, driver, sts); err != nil { // We will create, not update.
			return ctrl.Result{}, err
		}
		setResourceVersionAnnotations(log, cm, s, &sts.Spec.Template)
		setVolumeReferences(log, cm, s, &sts.Spec.Template.Spec)

		log.Info("Creating controller service Stateful Set")
		err = r.Create(ctx, sts)
		if err != nil {
			log.Error(err, "Failed to create controller service Stateful Set")
			return ctrl.Result{}, err
		}
	} else {
		// The Stateful Set exists, but it may need to be updated.
		mustUpdate = setResourceVersionAnnotations(log, cm, s, &sts.Spec.Template)
		mustUpdate = mustUpdate || setVolumeReferences(log, cm, s, &sts.Spec.Template.Spec)
		if mustUpdate {
			log.Info("Updating controller service Stateful Set")
			if err = r.Update(ctx, sts); err != nil {
				log.Error(err, "Failed to update controller service Stateful Set")
				return ctrl.Result{}, err
			}
		}
	}

	if meta.FindStatusCondition(driver.Status.Conditions, beegfsv1.ConditionNodeServiceReady).Reason ==
		beegfsv1.ReasonServiceNotCreated {
		// The Daemon Set doesn't exist. Let's create it.
		ds, _ = deploy.GetNodeServiceDaemonSet()
		if _, err = r.setCommonObjectMetadata(req, driver, ds); err != nil { // We will create, not update.
			return ctrl.Result{}, err
		}
		setResourceVersionAnnotations(log, cm, s, &ds.Spec.Template)
		setVolumeReferences(log, cm, s, &ds.Spec.Template.Spec)

		log.Info("Creating node service Daemon Set")
		err = r.Create(ctx, ds)
		if err != nil {
			log.Error(err, "Failed to create node service Daemon Set")
			return ctrl.Result{}, err
		}
	} else {
		// The Daemon Set exists, but it may need to be updated.
		mustUpdate = setResourceVersionAnnotations(log, cm, s, &ds.Spec.Template)
		mustUpdate = mustUpdate || setVolumeReferences(log, cm, s, &ds.Spec.Template.Spec)
		if mustUpdate {
			log.Info("Updating node service Daemon Set")
			if err = r.Update(ctx, ds); err != nil {
				log.Error(err, "Failed to update node service Daemon Set")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BeegfsDriverReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&beegfsv1.BeegfsDriver{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

// newConfigMap creates a new Config Map containing the configuration specified in the provided BeegfsDriver.
func newConfigMap(driver *beegfsv1.BeegfsDriver) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "csi-beegfs-config",
		},
	}
	_, err := setConfigMapData(driver, cm)
	return cm, err
}

// setConfigMapData ensures the provided Config Map contains the configuration specified in the provided BeegfsDriver.
// It returns mustUpdate=true if it makes a change and mustUpdate=false if it does not.
func setConfigMapData(driver *beegfsv1.BeegfsDriver, cm *corev1.ConfigMap) (bool, error) {
	mustUpdate := false

	// Add a warning that the Config Map should not be modified directly.
	const warning = "# This file is managed by the BeeGFS CSI driver operator. Do not modify it directly."
	newData, err := yaml.Marshal(driver.Spec.PluginConfigFromFile)
	if err != nil {
		return false, err
	}
	newStringData := fmt.Sprintf("%s\n%s", warning, string(newData))

	if oldStringData, ok := cm.Data["csi-beegfs-config.yaml"]; !ok || // The required key isn't there.
		len(cm.Data) > 1 || // There is more than one key.
		newStringData != oldStringData { // The data is stale or has been modified.
		mustUpdate = true
		cm.Data = map[string]string{"csi-beegfs-config.yaml": newStringData}
	}

	return mustUpdate, nil
}

// setCommonObjectMetadata can be used on any namespaced Kubernetes object to ensure that:
//   - The object exists in the correct namespace (based on the namespace of the request).
//   - The object is owned by the BeegfsDriver object (for proper garbage collection).
// setCommonObjectMetadata returns mustUpdate=true if it changes something about the object, mustUpdate=false if it
// does not, and an error if a change fails.
// TODO(webere, A265): Handle the fact that cluster scoped resources cannot be owned by our namespaced BeegfsDriver.
func (r *BeegfsDriverReconciler) setCommonObjectMetadata(req ctrl.Request, driver *beegfsv1.BeegfsDriver, object metav1.Object) (bool, error) {
	mustUpdate := false

	if len(object.GetNamespace()) == 0 || object.GetNamespace() != req.Namespace {
		object.SetNamespace(req.Namespace) // This is only necessary for newly created objects.
		mustUpdate = true
	}

	if len(object.GetOwnerReferences()) == 0 {
		err := ctrl.SetControllerReference(driver, object, r.Scheme)
		mustUpdate = true
		if err != nil {
			return mustUpdate, err
		}
	}

	return mustUpdate, nil
}

// setResourceVersionAnnotations is an important part of our overall configuration scheme. It records the current name
// and resource version of the Config Map and Secret required by our driver in annotations on a PodTemplateSpec (for
// either a DaemonSet or a StatefulSet). If nothing changes, setResourceVersionAnnotations returns mustUpdate=false.
// If it returns true, the caller should update the DaemonSet or StatefulSet that owns the PodTemplateSpec.
func setResourceVersionAnnotations(log logr.Logger, cm *corev1.ConfigMap, s *corev1.Secret, podTemplate *corev1.PodTemplateSpec) (mustUpdate bool) {
	if podTemplate.Annotations == nil {
		podTemplate.Annotations = make(map[string]string)
	}

	correctCMNameAndVersion := fmt.Sprintf("%s/%s", cm.Name, cm.ResourceVersion)
	val, ok := podTemplate.Annotations["beegfs.csi.netapp.com/configMapVersion"]
	if !ok || val != fmt.Sprintf("%s/%s", cm.Name, cm.ResourceVersion) {
		podTemplate.Annotations["beegfs.csi.netapp.com/configMapVersion"] = correctCMNameAndVersion
		log.Info("Config Map changed", "oldMap", val, "newMap", correctCMNameAndVersion)
		mustUpdate = true
	}

	if s != nil { // We may not be using a Secret in this deployment of the driver.
		correctSecretNameAndVersion := fmt.Sprintf("%s/%s", s.Name, s.ResourceVersion)
		val, ok = podTemplate.Annotations["beegfs.csi.netapp.com/connauthSecretVersion"]
		if !ok || val != fmt.Sprintf("%s/%s", s.Name, s.ResourceVersion) {
			podTemplate.Annotations["beegfs.csi.netapp.com/connauthSecretVersion"] = correctSecretNameAndVersion
			log.Info("Secret changed", "oldSecret", val, "newSecret", correctSecretNameAndVersion)
			mustUpdate = true
		}
	}

	return
}

// setVolumeReferences ensures that Pod specs point correctly to Kubernetes objects. In particular, it ensures that
// the controller service Stateful Set and the node service Daemon Set know which Config Map and Secret (respectively)
// to reference. If nothing changes, setVolumeReferences returns mustUpdate=false. If it returns true, the caller
// should update the DaemonSet or StatefulSet that owns the PodSpec.
func setVolumeReferences(log logr.Logger, cm *corev1.ConfigMap, s *corev1.Secret, podSpec *corev1.PodSpec) (mustUpdate bool) {
	for _, vol := range podSpec.Volumes {
		if vol.Name == "config-dir" && vol.ConfigMap.Name != cm.Name {
			log.Info("Setting reference to a new Config Map")
			vol.ConfigMap.Name = cm.Name
			mustUpdate = true
		} else if vol.Name == "connauth-dir" && vol.Secret.SecretName != s.Name {
			log.Info("Setting reference to a new Secret")
			vol.Secret.SecretName = s.Name
			mustUpdate = true
		}
	}
	return
}
