# BeeGFS CSI Driver Operator

An [operator](https://operatorframework.io/what/) can be used to deploy the
BeeGFS CSI driver to a cluster and manage its configuration/state within that
cluster. The operator is designed for integration with
[Operator Lifecycle Manager (OLM)](https://olm.operatorframework.io/) and is
primarily intended as an easy way for OpenShift administrators or
administrators with clusters running OLM to install the driver directly from
[Operator Hub](https://operatorhub.io/) and keep it updated.

## Contents
<a name="contents"></a>

* [Install the Operator](#install-operator)
   * [Install from the Openshift Console](#install-operator-openshift-console)
   * [Install the Operator from Operator Hub](#install-operator-operator-hub)
   * [Install from Manifests](#install-operator-manifests)
* [Install the Driver](#install-driver)
   * [Install from the OpenShift Console](#install-driver-openshift-console)
   * [Install Using kubectl or oc](#install-driver-kubectl-oc)

## Install the Operator
<a name="install-operator"></a>

### Install from the OpenShift Console
<a name="install-operator-openshift-console"></a>

### Install from Operator Hub
<a name="install-operator-operator-hub"></a>

### Install from Manifests
<a name="install-operator-manifests"></a>

It is possible, though not recommended, to install the operator (and to use the
operator to install the driver) directly from this repository. This is not a 
supported operation and the instructions provided are not intended for 
production environments.

1. Install the operator. The kustomizations scaffolded by the 
   operator-sdk are not supported by the version of Kustomize that ships with 
   kubectl (or oc). Kustomize must be installed and used (v3.8.7 is the 
   scaffolded Kustomize version).
   ```
   -> kustomize build operator/config/default | kubectl apply -f -
   namespace/beegfs-csi unchanged
   customresourcedefinition.apiextensions.k8s.io/beegfsdrivers.beegfs.csi.netapp.com configured
   serviceaccount/beegfs-csi-driver-operator-controller-manager created
   role.rbac.authorization.k8s.io/beegfs-csi-driver-operator-leader-election-role created
   clusterrole.rbac.authorization.k8s.io/beegfs-csi-driver-operator-manager-role created
   clusterrole.rbac.authorization.k8s.io/beegfs-csi-driver-operator-metrics-reader created
   clusterrole.rbac.authorization.k8s.io/beegfs-csi-driver-operator-proxy-role created
   rolebinding.rbac.authorization.k8s.io/beegfs-csi-driver-operator-leader-election-rolebinding created
   clusterrolebinding.rbac.authorization.k8s.io/beegfs-csi-driver-operator-manager-rolebinding created
   clusterrolebinding.rbac.authorization.k8s.io/beegfs-csi-driver-operator-proxy-rolebinding created
   configmap/beegfs-csi-driver-operator-manager-config created
   service/beegfs-csi-driver-operator-controller-manager-metrics-service created
   deployment.apps/beegfs-csi-driver-operator-controller-manager created

   ```
1. Verify the operator is running. If you did not modify the default 
   kustomization, it is running in the `beegfs-csi` namespace.
   ```
   -> kubectl get pods -n beegfs-csi
   NAME                                                             READY   STATUS    RESTARTS   AGE
   beegfs-csi-driver-operator-controller-manager-7b5cccff65-mzjxn   2/2     Running   0          61m
   ```

## Install the Driver

The BeegfsDriver Custom Resource Definition (CRD) defines the API for BeeGFS 
CSI Driver installation using the operator. The operator watches for a Custom 
Resource (CR) written according to the CRD and installs the driver and/or 
modifies the driver's installation based on the information contained within.

NOTE: The operator ONLY watches for a CR in its own namespace.

NOTE: There can only be ONE CR in the cluster at a time. It must be named 
csi-beegfs-cr.

### BeegfsDriver Custom Resource Fields

The CRD contains complete details on how the CR should be filled out. An 
example containing all allowed fields is below. 

NOTE: The driver is designed to run easily out-of-the-box using the minimal 
configuration defined in `operator/config/samples/beegfs_v1_beegfsdriver.yaml`. 
There is typically no need to modify the configuration. Some environments 
require BeeGFS client configuration in the pluginConfig section, and air-gapped 
environments can use the containerImageOverrides section to allow for an 
internal registry mirror.

```yaml
kind: BeegfsDriver
apiVersion: beegfs.csi.netapp.com/v1
metadata:
  name: csi-beegfs-cr  # CR must have this name.
spec:
  containerImageOverrides:
    beegfsCsiDriver:
      image: some.registry/netapp/beegfs-csi-driver
      tag: some-tag  # Changing this tag is not supported.
    csiNodeDriverRegistrar:
      image: some.registry/sig-storage/csi-node-driver-registrar
      tag: # Changing this tag is not supported.
    csiProvisioner:
      image: some.registry/sig-storage/csi-provisioner
      tag: # Changing this tag is not supported.
    livenessProbe:
      image: some.registry/sig-storage/livenessprobe
      tag: # Changing this tag is not supported.
  logLevel: 3
  nodeAffinityControllerService:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 50
        preference:
          matchExpressions:
            - key: node-role.kubernetes.io/master
              operator: Exists
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
        - matchExpressions:
            - key: node.openshift.io/os_id
              operator: NotIn
              values:
                - rhcos  # The BeeGFS CSI driver does not run on Red Hat CoreOS.
  nodeAffinityNodeService:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
        - matchExpressions:
            - key: node.openshift.io/os_id
              operator: NotIn
              values:
                - rhcos  # The BeeGFS CSI driver does not run on Red Hat CoreOS.
  # The pluginConfig field can contain significant detail. See the deployment 
  # documentation at docs/deployment (in addition to the CRD).
  pluginConfig:
    config:
    filesystemSpecificConfigs:
    nodeSpecificConfigs:
```

### Install from the OpenShift Console
<a name="install-driver-openshift-console"></a>

### Install Using kubectl or oc
<a name="install-driver-kubectl-oc"></a>

1. Create and/or modify a BeegfsDriver CR.
   ```
   -> cp operator/config/samples/beegfs_v1_beegfsdriver.yaml my_cr.yaml
   -> vim my_cr.yaml
   ```
1. Install the BeegfsDriver CR (this action kicks of the installation of the 
   BeeGFS CSI driver). The operator is typically running in the beegfs-csi 
   namespace.
   ```
   -> kubectl apply -k my_cr.yaml -n beegfs-csi
   beegfsdriver.beegfs.csi.netapp.com/csi-beegfs-cr created
   ```
1. Verify the driver is running. The total number of Pods running depends on the
   number of nodes in your cluster.
   ```
   -> kubectl get pods -n beegfs-csi
   NAME                                                             READY   STATUS    RESTARTS   AGE
   beegfs-csi-driver-operator-controller-manager-7b5cccff65-z8qdx   0/2     Running   0          53m
   csi-beegfs-controller-0                                          2/2     Running   0          37s
   csi-beegfs-node-hfxv2                                            3/3     Running   0          37s
   csi-beegfs-node-jhwgs                                            3/3     Running   0          37s
   csi-beegfs-node-jmpxk                                            3/3     Running   0          37s
   ```
