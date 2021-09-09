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
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	deploy "github.com/netapp/beegfs-csi-driver/deploy/k8s"
	beegfsv1 "github.com/netapp/beegfs-csi-driver/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=storage.k8s.io,resources=csidrivers,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,resourceNames=privileged,verbs=use

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

	// Get "clean" versions of all objects from deployment manifests. Ignore potential errors because unit testing in
	// the deploy package ensures they will not occur.
	sts, _ := deploy.GetControllerServiceStatefulSet()
	ds, _ := deploy.GetNodeServiceDaemonSet()
	rbacInterfaces, _ := deploy.GetRBAC()
	d, _ := deploy.GetCSIDriver()

	// -----------------------------------------------------------------------------------------------------------------
	// Start by getting everything we need to generate an up-to-date status, generating the up-to-date status, and
	// pushing the up-to-date status to the Kubernetes API server.

	oldStatus := *driver.Status.DeepCopy() // Remember previous status for comparison.

	statusCondition := metav1.Condition{}
	stsFromCluster := new(appsv1.StatefulSet)
	err = r.Get(ctx, types.NamespacedName{Name: sts.Name, Namespace: req.Namespace}, stsFromCluster)
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
		if stsFromCluster.Status.Replicas < 1 {
			statusCondition = metav1.Condition{
				Type:    beegfsv1.ConditionControllerServiceReady,
				Status:  metav1.ConditionFalse,
				Reason:  beegfsv1.ReasonPodsNotScheduled,
				Message: "0/1 controller service pods have been scheduled",
			}
		} else if stsFromCluster.Status.ReadyReplicas < 1 {
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

	statusCondition = metav1.Condition{}
	dsFromCluster := new(appsv1.DaemonSet)
	err = r.Get(ctx, types.NamespacedName{Name: ds.Name, Namespace: req.Namespace}, dsFromCluster)
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
		if dsFromCluster.Status.DesiredNumberScheduled < 1 {
			statusCondition = metav1.Condition{
				Type:    beegfsv1.ConditionNodeServiceReady,
				Status:  metav1.ConditionFalse,
				Reason:  beegfsv1.ReasonPodsNotScheduled,
				Message: "0 node service pods have been scheduled",
			}
		} else if dsFromCluster.Status.NumberReady < dsFromCluster.Status.DesiredNumberScheduled {
			statusCondition = metav1.Condition{
				Type:   beegfsv1.ConditionNodeServiceReady,
				Status: metav1.ConditionFalse,
				Reason: beegfsv1.ReasonPodsNotReady,
				Message: fmt.Sprintf("%d/%d node service pods are ready", dsFromCluster.Status.NumberReady,
					dsFromCluster.Status.DesiredNumberScheduled),
			}
		} else {
			statusCondition = metav1.Condition{
				Type:   beegfsv1.ConditionNodeServiceReady,
				Status: metav1.ConditionTrue,
				Reason: beegfsv1.ReasonPodsReady,
				Message: fmt.Sprintf("%d/%d node service pods are ready", dsFromCluster.Status.NumberReady,
					dsFromCluster.Status.DesiredNumberScheduled),
			}
		}
	}
	meta.SetStatusCondition(&driver.Status.Conditions, statusCondition)

	// Don't bother the Kubernetes API server unless we think the status has changed.
	if !equality.Semantic.DeepEqual(driver.Status, oldStatus) {
		log.Info("Updating status")
		err = r.Status().Update(ctx, driver)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Handle finalizers, either by adding them if they are missing or executing on them if the CR is being deleted.

	// Some resources needed by the BeeGFS CSI driver are cluster-scoped. These resources cannot be owned by our
	// namespace-scoped CRD and thus cannot be garbage collected. This finalizer enables us to manually delete these
	// cluster-scoped resources before garbage collection occurs.
	const clusterResourceDeletionFinalizer = "beegfs.csi.netapp.com/clusterResourceDeletion"

	if driver.ObjectMeta.DeletionTimestamp.IsZero() {
		// The CR is not being deleted. Let's add our finalizer and update it if necessary.
		if !containsString(driver.GetFinalizers(), clusterResourceDeletionFinalizer) {
			controllerutil.AddFinalizer(driver, clusterResourceDeletionFinalizer)
			log.Info("Adding finalizer", "finalizer", clusterResourceDeletionFinalizer)
			if err = r.Update(ctx, driver); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The CR is being deleted. Execute on our finalizer by deleting cluster-scoped resources.
		if containsString(driver.GetFinalizers(), clusterResourceDeletionFinalizer) {
			log.Info("Deleting cluster-scoped objects")
			// There are some number of RBAC objects from the deployment manifests on the cluster. We do not hard code
			// the exact nature of these objects here.
			for _, i := range rbacInterfaces {
				switch object := i.(type) {
				case *rbacv1.ClusterRole:
					if err = r.Delete(ctx, object); err != nil && !errors.IsNotFound(err) {
						return ctrl.Result{}, err
					}
				case *rbacv1.ClusterRoleBinding:
					if err = r.Delete(ctx, object); err != nil && !errors.IsNotFound(err) {
						return ctrl.Result{}, err
					}
				}
			}
			if err = r.Delete(ctx, d); err != nil && !errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(driver, clusterResourceDeletionFinalizer)
			if err = r.Update(ctx, driver); err != nil {
				return ctrl.Result{}, err
			}
		}
		// There is no point in continuing to reconcile a deleting CR.
		return ctrl.Result{}, nil
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Now attempt to get the rest of the expected objects and push them to the Kubernetes API server as necessary.

	// Completely recreate the Config Map to ensure all fields specified in the deployment manifest are propagated.
	cm := new(corev1.ConfigMap)
	if cm, err = newConfigMap(driver); err != nil {
		return ctrl.Result{}, err
	}
	if err = r.setCommonObjectMetadata(req, driver, cm); err != nil {
		return ctrl.Result{}, err
	}
	cmFromCluster := new(corev1.ConfigMap)
	err = r.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: req.Namespace}, cmFromCluster)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err // Something we aren't prepared for went wrong.
		} else {
			// The Config Map doesn't exist. Let's create it.
			log.Info("Creating Config Map")
			if err = r.Create(ctx, cm); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else if !equality.Semantic.DeepEqual(cm.Data, cmFromCluster.Data) {
		// The Config Map needs to be updated.
		log.Info("Updating Config Map")
		if err = r.Update(ctx, cm); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		cm = cmFromCluster // We need the correct resourceVersion later on.
	}

	// A connauth Secret named "csi-beegfs-connauth" in the operator's namespace is required for driver operation. If
	// it does not exist, we create it, own it, and garbage collect it. If it already exists (pre-created by an
	// administrator) we do nothing.
	s := newSecret()
	if err = r.setCommonObjectMetadata(req, driver, s); err != nil {
		return ctrl.Result{}, err
	}
	sFromCluster := new(corev1.Secret)
	err = r.Get(ctx, types.NamespacedName{Name: s.Name, Namespace: req.Namespace}, sFromCluster)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err // Something we aren't prepared for went wrong.
		} else {
			// The Secret doesn't exist. Let's create it.
			log.Info("Creating Secret")
			if err = r.Create(ctx, s); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// Many of the other objects created by this controller may need to be updated to keep them in sync with the
		// CRD. We expect the Secret to be updated manually and have no meaningful changes to make here.
		s = sFromCluster // We need the correct resourceVersion later on.
	}

	// There are some number of RBAC objects from the deployment manifests on the cluster. We do not hard code
	// the exact nature of these objects here.
	for _, i := range rbacInterfaces {
		switch object := i.(type) {
		// Completely recreate the Service Account to ensure all fields specified in the deployment manifests are
		// propagated.
		case *corev1.ServiceAccount:
			if err = r.setCommonObjectMetadata(req, driver, object); err != nil {
				return ctrl.Result{}, err
			}
			saFromCluster := new(corev1.ServiceAccount)
			err = r.Get(ctx, types.NamespacedName{Name: object.Name, Namespace: req.Namespace}, saFromCluster)
			if err != nil {
				if !errors.IsNotFound(err) {
					return ctrl.Result{}, err // Something we aren't prepared for went wrong.
				} else {
					// The Service Account doesn't exist. Let's create it.
					log.Info("Creating Service Account", "name", object.Name)
					if err = r.Create(ctx, object); err != nil {
						return ctrl.Result{}, err
					}
				}
			} else {
				// Intentionally empty.
				// We do not monitor and continuously update Service Accounts because they don't have any useful fields.
			}

		case *rbacv1.ClusterRole:
			// Completely recreate the Cluster Role to ensure all fields specified in the deployment manifests are
			// propagated. Don't call setCommonObjectMetadata because this is a cluster-scoped object (it doesn't have a
			// namespace and our namespace-scoped CRD can't own it).
			crFromCluster := new(rbacv1.ClusterRole)
			err = r.Get(ctx, types.NamespacedName{Name: object.Name}, crFromCluster)
			if err != nil {
				if !errors.IsNotFound(err) {
					return ctrl.Result{}, err // Something we aren't prepared for went wrong.
				} else {
					// The Cluster Role doesn't exist. Let's create it.
					log.Info("Creating Cluster Role", "name", object.Name)
					if err = r.Create(ctx, object); err != nil {
						return ctrl.Result{}, err
					}
				}
			} else if !equality.Semantic.DeepEqual(object.Rules, crFromCluster.Rules) {
				// The Cluster Role on the cluster needs to be updated.
				log.Info("Updating Cluster Role", "name", object.Name)
				if err = r.Update(ctx, object); err != nil {
					return ctrl.Result{}, err
				}
			}

		case *rbacv1.ClusterRoleBinding:
			// Completely recreate the Cluster Role Binding to ensure all fields specified in the deployment manifests
			// are propagated. Don't call setCommonObjectMetadata because this is a cluster-scoped object (it doesn't
			// have a namespace and our namespace-scoped CRD can't own it).
			for j, _ := range object.Subjects {
				object.Subjects[j].Namespace = req.Namespace
			}
			crbFromCluster := new(rbacv1.ClusterRoleBinding)
			err = r.Get(ctx, types.NamespacedName{Name: object.Name}, crbFromCluster)
			if err != nil {
				if !errors.IsNotFound(err) {
					return ctrl.Result{}, err // Something we aren't prepared for went wrong.
				} else {
					// The Cluster Role Binding doesn't exist. Let's create it.
					log.Info("Creating Cluster Role Binding", "name", object.Name)
					if err = r.Create(ctx, object); err != nil {
						return ctrl.Result{}, err
					}
				}
			} else if !equality.Semantic.DeepEqual(object.Subjects, crbFromCluster.Subjects) ||
				!equality.Semantic.DeepEqual(object.RoleRef, crbFromCluster.RoleRef) {
				// The Cluster Role Binding on the cluster needs to be updated.
				log.Info("Updating Cluster Role Binding", "name", object.Name)
				if err = r.Update(ctx, object); err != nil {
					return ctrl.Result{}, err
				}
			}

		case *rbacv1.Role:
			// Completely recreate the Role o ensure all fields specified in the deployment manifest are propagated.
			if err = r.setCommonObjectMetadata(req, driver, object); err != nil {
				return ctrl.Result{}, err
			}
			rFromCluster := new(rbacv1.Role)
			err = r.Get(ctx, types.NamespacedName{Name: object.Name, Namespace: req.Namespace}, rFromCluster)
			if err != nil {
				if !errors.IsNotFound(err) {
					return ctrl.Result{}, err // Something we aren't prepared for went wrong.
				} else {
					// The Role doesn't exist. Let's create it.
					log.Info("Creating Role", "name", object.Name)
					if err = r.Create(ctx, object); err != nil {
						return ctrl.Result{}, err
					}
				}
			} else if !equality.Semantic.DeepEqual(object.Rules, rFromCluster.Rules) {
				// The Cluster Role on the cluster needs to be updated.
				log.Info("Updating Role", "name", object.Name)
				if err = r.Update(ctx, object); err != nil {
					return ctrl.Result{}, err
				}
			}

		case *rbacv1.RoleBinding:
			// Completely recreate the Role Binding to ensure all fields specified in the deployment manifest are
			// propagated.
			if err = r.setCommonObjectMetadata(req, driver, object); err != nil {
				return ctrl.Result{}, err
			}
			for j, _ := range object.Subjects {
				object.Subjects[j].Namespace = req.Namespace
			}
			crbFromCluster := new(rbacv1.RoleBinding)
			err = r.Get(ctx, types.NamespacedName{Name: object.Name, Namespace: req.Namespace}, crbFromCluster)
			if err != nil {
				if !errors.IsNotFound(err) {
					return ctrl.Result{}, err // Something we aren't prepared for went wrong.
				} else {
					// The Role Binding doesn't exist. Let's create it.
					log.Info("Creating Role Binding", "name", object.Name)
					if err = r.Create(ctx, object); err != nil {
						return ctrl.Result{}, err
					}
				}
			} else if !equality.Semantic.DeepEqual(object.Subjects, crbFromCluster.Subjects) ||
				!equality.Semantic.DeepEqual(object.RoleRef, crbFromCluster.RoleRef) {
				// The Role Binding on the cluster needs to be updated.
				log.Info("Updating Role Binding", "name", object.Name)
				if err = r.Update(ctx, object); err != nil {
					return ctrl.Result{}, err
				}
			}
		}
	}

	// Completely recreate the CSI Driver object to ensure all fields specified in the deployment manifest are
	// propagated. Don't call setCommonObjectMetadata because this is a cluster-scoped object (it doesn't have a
	// namespace and our namespace-scoped CRD can't own it).
	dFromCluster := new(storagev1.CSIDriver)
	err = r.Get(ctx, types.NamespacedName{Name: d.Name}, dFromCluster)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err // Something we aren't prepared for went wrong.
		} else {
			// The CSI Driver object doesn't exist. Let's create it.
			log.Info("Creating CSI Driver object")
			if err = r.Create(ctx, d); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// Intentionally empty.
		// We do not monitor and continuously update the CSI Driver object for two reasons:
		//   1. Feature gating makes it very difficult to determine whether the spec on the Kubernetes API server needs
		//      to be updated. E.g., the API server spec shows fsGroupPolicy=nil even if we set fsGroupPolicy=true when
		//      the CSIVolumeFSGroupPolicy feature gate is disabled.
		//   2. Most of the fields in the spec are immutable.
		// If we anticipate a need to change the CSI Driver object, we should advise users to delete it manually so the
		// operator can recreate it.
	}

	// Completely recreate the Stateful Set to ensure all fields specified in the deployment manifest are propagated.
	if err = r.setCommonObjectMetadata(req, driver, sts); err != nil {
		return ctrl.Result{}, err
	}
	setResourceVersionAnnotations(log, cm, s, &sts.Spec.Template)
	setImages(log, sts.Spec.Template.Spec.Containers, driver.Spec.ContainerImageOverrides)
	setLogLevel(log, driver.Spec.LogLevel, sts.Spec.Template.Spec.Containers)
	setNodeAffinity(log, &driver.Spec.NodeAffinityControllerService, &sts.Spec.Template.Spec)
	if meta.FindStatusCondition(driver.Status.Conditions, beegfsv1.ConditionControllerServiceReady).Reason ==
		beegfsv1.ReasonServiceNotCreated {
		// The Stateful Set doesn't exist. Let's create it.
		log.Info("Creating controller service Stateful Set")
		if err = r.Create(ctx, sts); err != nil {
			return ctrl.Result{}, err
		}
	} else if !equality.Semantic.DeepDerivative(sts.Spec, stsFromCluster.Spec) {
		// The Stateful Set needs to be updated.
		log.Info("Updating controller service Stateful Set")
		if err = r.Update(ctx, sts); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Completely recreate the Daemon Set to ensure all fields specified in the deployment manifest are propagated.
	if err = r.setCommonObjectMetadata(req, driver, ds); err != nil {
		return ctrl.Result{}, err
	}
	setResourceVersionAnnotations(log, cm, s, &ds.Spec.Template)
	setImages(log, ds.Spec.Template.Spec.Containers, driver.Spec.ContainerImageOverrides)
	setLogLevel(log, driver.Spec.LogLevel, ds.Spec.Template.Spec.Containers)
	setNodeAffinity(log, &driver.Spec.NodeAffinityNodeService, &ds.Spec.Template.Spec)
	if meta.FindStatusCondition(driver.Status.Conditions, beegfsv1.ConditionNodeServiceReady).Reason ==
		beegfsv1.ReasonServiceNotCreated {
		// The Daemon Set doesn't exist. Let's create it.
		log.Info("Creating node service Daemon Set")
		if err = r.Create(ctx, ds); err != nil {
			return ctrl.Result{}, err
		}
	} else if !equality.Semantic.DeepDerivative(ds.Spec, dsFromCluster.Spec) {
		// The Daemon Set needs to be updated.
		log.Info("Updating controller service Daemon Set")
		if err = r.Update(ctx, ds); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BeegfsDriverReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&beegfsv1.BeegfsDriver{}).
		// Only own (i.e. watch) objects if there is some action we should take if they change.
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

// newConfigMap creates a new Config Map containing the configuration specified in the provided BeegfsDriver.
func newConfigMap(driver *beegfsv1.BeegfsDriver) (*corev1.ConfigMap, error) {
	const warning = "# This file is managed by the BeeGFS CSI driver operator. Do not modify it directly."
	data, err := yaml.Marshal(driver.Spec.PluginConfigFromFile)
	if err != nil {
		return nil, err
	}
	stringData := fmt.Sprintf("%s\n%s", warning, string(data))

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: deploy.ResourceNameConfigMap,
		},
		Data: map[string]string{deploy.KeyNameConfigMap: stringData},
	}

	return cm, nil
}

// newSecret creates the new empty secret required for the BeeGFS CSI driver to operate.
func newSecret() *corev1.Secret {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: deploy.ResourceNameSecret,
		},
		Data: map[string][]byte{deploy.KeyNameSecret: nil},
	}
	return s
}

// setCommonObjectMetadata can be used on any namespaced Kubernetes object to ensure that:
//   - The object exists in the correct namespace (based on the namespace of the request).
//   - The object is owned by the BeegfsDriver object (for proper garbage collection). setCommonObjectMetadata will NOT
//     set an owner reference if one already exists.
func (r *BeegfsDriverReconciler) setCommonObjectMetadata(req ctrl.Request, driver *beegfsv1.BeegfsDriver,
	object metav1.Object) error {
	object.SetNamespace(req.Namespace)
	return ctrl.SetControllerReference(driver, object, r.Scheme)
}

// setResourceVersionAnnotations is an important part of our overall configuration scheme. It records the current name
// and resource version of the Config Map and Secret required by our driver in annotations on a Pod Template Spec (for
// either a Daemon Set or a Stateful Set).
func setResourceVersionAnnotations(log logr.Logger, cm *corev1.ConfigMap, s *corev1.Secret,
	podTemplate *corev1.PodTemplateSpec) {
	if podTemplate.Annotations == nil {
		podTemplate.Annotations = make(map[string]string)
	}

	correctCMNameAndVersion := fmt.Sprintf("%s/%s", cm.Name, cm.ResourceVersion)
	podTemplate.Annotations["beegfs.csi.netapp.com/configMapVersion"] = correctCMNameAndVersion
	log.V(5).Info("Setting Config Map version annotation", "versionAnnotation", correctCMNameAndVersion)

	if s != nil { // We may not be using a Secret in this deployment of the driver.
		correctSecretNameAndVersion := fmt.Sprintf("%s/%s", s.Name, s.ResourceVersion)
		podTemplate.Annotations["beegfs.csi.netapp.com/connauthSecretVersion"] = correctSecretNameAndVersion
		log.V(5).Info("Setting Secret version annotation", "versionAnnotation", correctSecretNameAndVersion)
	}
}

// setImages takes a slice of Container specs (containers) and a slice of ContainerImageOverrides (overrides). If the
// image field of a spec in containers is overriden in overrides, setImages modifies it. Otherwise, setImages assumes
// the image field is already correct and leaves it alone.
func setImages(log logr.Logger, containers []corev1.Container, overrides beegfsv1.ContainerImageOverrides) {
	// Match fields in overrides to expected container names for ease of lookup. Tests in deploy ensure default
	// containers maintain these expected names. This is not the only way we could determine whether a container's
	// image should be overriden (e.g. index in PodTemplateSpec.Containers or hard coding a particular image name we
	// expect to be overridden), but container name is one of the most reliable fields (i.e. least likely to change) in
	// the deployment manifests.
	containerNameToImageOverrideMap := map[string]beegfsv1.ContainerImageOverride{
		deploy.ContainerNameBeegfsCsiDriver:        overrides.BeegfsCsiDriver,
		deploy.ContainerNameCsiNodeDriverRegistrar: overrides.CsiNodeDriverRegistrar,
		deploy.ContainerNameCsiProvisioner:         overrides.CsiProvisioner,
		deploy.ContainerNameLivenessProbe:          overrides.LivenessProbe,
	}

	for i, container := range containers {
		if v, ok := containerNameToImageOverrideMap[container.Name]; ok && (len(v.Image) > 0 || len(v.Tag) > 0) {
			newImage := getImageStringWithOverride(container.Image, v)
			log.V(5).Info("Setting Container image", "containerName", container.Name,
				"containerImage", newImage)
			containers[i].Image = newImage
		}
	}
}

// getImageStringWithOverride takes an imageString (e.g. k8s.gcr.io/some-image:some-tag) and a ContainerImageOverride
// (consisting of an image and a tag). The image string it returns includes any non-empty information from the override
// and information from imageString as required.
func getImageStringWithOverride(imageWithTag string, override beegfsv1.ContainerImageOverride) string {
	var image, tag string

	// Split the image from the tag, as they can be overridden separately.
	imageSlice := strings.SplitN(imageWithTag, ":", 2)
	image = imageSlice[0]
	if len(imageSlice) > 1 {
		tag = imageSlice[1]
	}

	if len(override.Image) > 0 {
		image = override.Image
	}
	if len(override.Tag) > 0 {
		tag = override.Tag
	}

	return fmt.Sprintf("%s:%s", image, tag)
}

// containsString checks if a string is contained in a slice of strings. Its implementation comes from the Kubebuilder
// book (https://book.kubebuilder.io/reference/using-finalizers.html).
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// setLogLevel sets the value of the environment variable LOG_LEVEL to level for any Container in containers. If a
// Container does not have the environment variable LOG_LEVEL, setLogLevel does nothing. If level is nil, setLogLevel
// does nothing. The ultimate result is that all containers with a configurable logging level in the deployment
// manifest will log at the specified level.
func setLogLevel(log logr.Logger, level *int, containers []corev1.Container) {
	if level == nil {
		return
	}
	log.V(5).Info("Setting log level in all Containers", "level", level)
	for i, container := range containers {
		for j, envVar := range container.Env {
			if envVar.Name == "LOG_LEVEL" {
				containers[i].Env[j].Value = strconv.Itoa(*level)
			}
		}
	}
}

// setNodeAffinity adds the passed NodeAffinity to the passed PodSpec (or replaces the PodSpec's existing NodeAffinity
// if it has one).
func setNodeAffinity(log logr.Logger, affinity *corev1.NodeAffinity, spec *corev1.PodSpec) {
	if affinity != nil {
		log.V(5).Info("Setting node affinity", "affinity", affinity)
		if spec.Affinity == nil {
			spec.Affinity = new(corev1.Affinity)
		}
		spec.Affinity.NodeAffinity = affinity
	}
}
