# BeeGFS CSI Driver Operator <!-- omit in toc -->

## Contents <!-- omit in toc -->
<a name="contents"></a>

- [Overview](#overview)
- [Requirements](#requirements)
- [Verify the Operator Image Signature](#verify-the-operator-image-signature)
  - [Validate the BeeGFS CSI Driver Operator Image Signature](#validate-the-beegfs-csi-driver-operator-image-signature)
- [Install the Operator](#install-the-operator)
  - [Install from OperatorHub](#install-from-operatorhub)
  - [Install from the OpenShift Console (deprecated)](#install-from-the-openshift-console-deprecated)
  - [Install from Manifests](#install-from-manifests)
- [Install the Driver](#install-the-driver)
  - [BeegfsDriver Custom Resource Fields](#beegfsdriver-custom-resource-fields)
  - [ConnAuth Configuration](#connauth-configuration)
  - [TLS Certificate Configuration](#tls-certificate-configuration)
  - [Verify the BeeGFS CSI Driver Image Signature](#verify-the-beegfs-csi-driver-image-signature)
  - [Install from the OpenShift Console (deprecated)](#install-from-the-openshift-console-deprecated-1)
  - [Install Using kubectl](#install-using-kubectl)
- [Modify the Driver Configuration](#modify-the-driver-configuration)
- [Upgrade the Driver](#upgrade-the-driver)
- [Uninstall the Driver and/or Operator](#uninstall-the-driver-andor-operator)

## Overview
<a name="overview"></a>

An [operator](https://operatorframework.io/what/) can be used to deploy the
BeeGFS CSI driver to a Kubernetes cluster and manage its configuration/state
within that cluster. The BeeGFS CSI driver  operator is designed for integration
with [Operator Lifecycle Manager (OLM)](https://olm.operatorframework.io/) and
is primarily intended as an easy way for OpenShift administrators or
administrators with clusters running OLM to install the driver directly from
[OperatorHub](https://operatorhub.io/) and keep it updated.

Management of the BeeGFS CSI driver with the operator involves two "phases":

1. Install the operator itself. This step can be done simply through the OLM 
   interface with virtually no configuration.
1. Use the operator to install the BeeGFS CSI driver operand (i.e. install a 
   BeegfsDriver custom resource). This step allows for full configuration and 
   is what actually enables a cluster to handle BeeGFS volumes.

## Requirements
<a name="requirements"></a>

A full compatibility matrix is included in the main driver 
[README](../README.md).

The BeeGFS CSI driver is only supported on nodes running a [BeeGFS supported 
Linux 
distribution](https://doc.beegfs.io/latest/release_notes.html#supported-linux-distributions-and-kernels). 
Red Hat CoreOS (RHCOS), the default distribution in OpenShift environments, is
NOT supported. However, Red Hat Enterprise Linux (RHEL) nodes can be added to an 
OpenShift cluster and RHEL is supported for running BeeGFS. By default, the 
operator will install the driver in an OpenShift cluster with a node affinity 
that ensures it does not run on RHCOS nodes.

## Verify the Operator Image Signature
<a name="verify-operator-signature"></a>

The BeeGFS CSI Driver Operator is signed with the same key as the BeeGFS CSI
Driver container images. You can choose to manually verify the signature
associated with the BeeGFS CSI Driver Operator image starting with version 1.4.0
of the driver and operator.

### Validate the BeeGFS CSI Driver Operator Image Signature
<a name="validate-the-beegfs-csi-driver-operator-image-signature"></a>

First find the container image reference used for the operator.

If you are installing from the OpenShift Console:
 * Navigate to the OperatorHub pane of the console.
 * Search for and click on the BeeGFS CSI driver operator.
 * Within the details of the operator you can find the container image which is
   the reference to the image used for the operator. Note this image reference.

If you are installing from OperatorHub:
  * Navigate to [OperatorHub.io](https://operatorhub.io/).
  * search for and click on the BeeGFS CSI driver operator. It is in the Storage
    category.
  * Within the details for the BeeGFS CSI driver operator find the container
    image. Note this image reference.

On a host with the cosign command installed and the BeeGFS signing key file
available, verify the image signature using the cosign command. Remember to use
the image reference of the version you want to install.

Example validation using the certificate file.
```
cosign verify --key <PUBLIC_KEY_FILE> ghcr.io/thinkparq/beegfs-csi-driver-operator:<TAG>
```

If the image signature is properly validated then continue installing the
operator.


## Install the Operator
<a name="install-operator"></a>

### Install from OperatorHub
<a name="install-operator-operator-hub"></a>

Operators can be installed from [OperatorHub.io](https://operatorhub.io/) on 
any Operator Lifecycle Manager (OLM)-enabled cluster.

As a user with administrative permissions and "kubectl" access to a cluster:

1. Navigate to [OperatorHub.io](https://operatorhub.io/).
1. Search for and click on the BeeGFS CSI driver operator. It is a tagged for 
   "storage".
1. Click "Install".
1. Copy the installation command and run it in your cluster.  
   NOTE: Because the BeeGFS CSI driver operator is configured to install in a 
   single namespace, the installation command will configure an Operator Group 
   and install it in the "my-beegfs-csi-driver" namespace
   
It is also possible to install an operator listed on OperatorHub in a more 
manual way by creating a Subscription that references a Catalog Source. See the 
[OLM documentation](https://olm.operatorframework.io/docs/tasks/install-operator-with-olm/) 
for details.

### Install from the OpenShift Console (deprecated)
<a name="install-operator-openshift-console"></a>

Operators are first-class citizens in OpenShift and are easy to install from 
the OpenShift console. As a user with administrative permissions:

1. Navigate to the OperatorHub pane of the console.
1. Search for and click on the BeeGFS CSI driver operator. It is a "community" 
   operator tagged for "storage".
1. Click "Install", verify the defaults, and click "Install" again.

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
<a name="install-driver"></a>

The BeegfsDriver Custom Resource Definition (CRD) defines the API for BeeGFS 
CSI Driver installation using the operator. The operator watches for a Custom 
Resource (CR) written according to the CRD and installs the driver and/or 
modifies the driver's installation based on the information contained within.

NOTE: The operator ONLY watches for a CR in its own namespace.

NOTE: There can only be ONE CR in the cluster at a time. It must be named 
csi-beegfs-cr.

### BeegfsDriver Custom Resource Fields
<a name="crd-fields"></a>

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
    csiResizer:
      image: some.registry/sig-storage/csi-resizer
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

### ConnAuth Configuration
<a name="connauth-configuration"></a>

The Kubernetes deployment of the BeeGFS CSI driver makes use of a Secret to 
store the contents of [BeeGFS 
connAuthFiles](../docs/deployment.md#connauth-configuration). The driver 
expects the Secret to be named *csi-beegfs-connauth* and to exist in the driver
namespace. If the Secret does not exist when a BeegfsDriver resource is applied,
the operator creates it with an empty data field.

To use connAuth information with a driver deployed by the operator, either:
1. Pre-create a Secret named *csi-beegfs-connauth* in the driver namespace, or
2. Modify the Secret after it has been created by the operator.  
   NOTE: A Secret's `data` field must be Base64-encoded, making it cumbersome to 
   modify directly. A simpler approach is to add connAuth information to the 
   existing Secret's `stringData` field. Kubernetes will automatically encode 
   the string and add the result to the `data` field in the background.
   
A correctly formatted Secret looks like this:

```yaml
kind: Secret
metadata:
  name: csi-beegfs-connauth
  namespace: beegfs-csi
type: Opaque
data:
  csi-beegfs-connauth.yaml: |
    - sysMgmtdHost: some.specific.file.system
      connAuth: some-secret
      encoding: encoding-type # raw or base64
    - sysMgmtdHost: some.other.specific.file.system
      connAuth: some-other-secret
      encoding: encoding-type # raw or base64
```

### TLS Certificate Configuration
<a name="tls-certificate-configuration"></a>

The Kubernetes deployment of the BeeGFS CSI driver makes use of a Secret to store the contents of
[BeeGFS TLS Certificates](../docs/deployment.md#tls-certificate-configuration). The driver expects
the Secret to be named *csi-beegfs-tlscerts* and to exist in the driver namespace. If the Secret
does not exist when a BeegfsDriver resource is applied, the operator creates it with an empty data
field.

To use TLS certificates with a driver deployed by the operator, either:
1. Pre-create a Secret named *csi-beegfs-connauth* in the driver namespace, or
2. Modify the Secret after it has been created by the operator.  
   NOTE: A Secret's `data` field must be Base64-encoded, making it cumbersome to modify directly. A
   simpler approach is to add TLS cert information to the existing Secret's `stringData` field.
   Kubernetes will automatically encode the string and add the result to the `data` field in the
   background.
   
A correctly formatted Secret looks like this:

```yaml
kind: Secret
apiVersion: v1
metadata:
  name: csi-beegfs-tlscerts
stringData:
  csi-beegfs-tlscerts.yaml: |
    - sysMgmtdHost: some.specific.file.system
      tlsCert: |+
        -----BEGIN CERTIFICATE-----
        ...
        -----END CERTIFICATE-----
    - sysMgmtdHost: some.other.specific.file.system
      tlsCert: |+
        -----BEGIN CERTIFICATE-----
        ...
        -----END CERTIFICATE-----        
```

### Verify the BeeGFS CSI Driver Image Signature

If you want to verify the signature of the BeeGFS CSI Driver image deployed by
the operator you can follow the steps in the [BeeGFS CSI Driver Deployment
guide](../docs/deployment.md#verify-the-signing-certificate-is-trusted) for
verifying the signature of the driver image.

Note that by default the operator will deploy an image using the same version
tag as is used by the operator.

For example, if you have the following operator version installed.

```
beegfs-csi-driver-operator.v1.5.0
```

The version of the driver that would be deployed would match ```v1.5.0``` and
would deploy the image ```ghcr.io/thinkparq/beegfs-csi-driver:v1.5.0```. In this
case the image ```ghcr.io/thinkparq/beegfs-csi-driver:v1.5.0``` would be the
image to use with for the signature verification.

### Install from the OpenShift Console (deprecated)
<a name="install-driver-openshift-console"></a>

Operator "operands" are first-class citizens in OpenShift and are easy to 
install from the OpenShift console. As a user with administrative permissions:

1. Navigate to the BeeGFS CSI driver page in the "Installed Operators" pane of 
   the OpenShift console.
1. Click the "BeeGFS Driver" tab and "Create BeegfsDriver".
1. Do one (or both) of the following:
   * Use "Form view" to modify the driver's configuration.  
     NOTE: beegfsClientConf fields of the pluginConfig section are not
     represented in "Form view". Use "YAML view" instead
   * Use "YAML view" to modify the driver's configuration.  
     NOTE: The "Schema" navigator to the right in "YAML view" makes it easy to 
     figure out what fields exist for configuration.
1. Click "Create".

### Install Using kubectl
<a name="install-driver-kubectl-oc"></a>

Use this method if you have installed the operator either from OperatorHub or 
directly from manifests.

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

## Modify the Driver Configuration
<a name="modify-driver-configuration"></a>

Aside from [connAuth configuration](#connauth-configuration), all other aspects 
of the BeeGFS CSI driver's 
[configuration](../docs/deployment.md#managing-beegfs-client-configuration) are 
handled through the BeegfsDriver CRD. Modifying the applied CR results in the 
redeployment of the driver with updated configuration.

## Upgrade the Driver
<a name="upgrade-driver"></a>

The BeeGFS CSI Driver and the operator are versioned together. Unless a 
specific version of the driver is tagged in the 
`spec.containerImageOverrides.beegfsCsiDriver.tag` field of the applied CR, 
upgrading the operator to a new minor version results in an automatic upgrade 
of the driver to the same version.

## Uninstall the Driver and/or Operator
<a name="uninstall"></a>

To uninstall the driver, simply delete the applied CR (named *csi-beegfs-cr*) 
and wait a few minutes for the operator to clean the driver up. This can be 
done from the OpenShift console (if avalable) or by using `kubectl`/`oc`.

To uninstall the operator, reverse the [steps you used to install 
it](#install-operator). These steps differ depending on the environment.

NOTE: Uninstalling the operator will typically not uninstall the driver, and 
the operator is required for driver uninstallation. To completely clean up a 
cluster, uninstall the driver BEFORE uninstalling the operator.
