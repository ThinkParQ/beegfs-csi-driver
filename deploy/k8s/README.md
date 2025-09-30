# Kustomize Specific Deployment Details <!-- omit in toc -->

<a name="contents"></a>
## Contents <!-- omit in toc -->

- [Basics](#basics)
- [Upgrade Notes](#upgrade-notes)
  - [Upgrading to v1.2.0](#upgrading-to-v120)
  - [Upgrading to v1.4.0](#upgrading-to-v140)
  - [Upgrading to v1.5.0](#upgrading-to-v150)

<a name="basics"></a>
## Basics

The BeeGFS CSI driver uses [Kustomize](https://kustomize.io/) as its default 
deployment mechanism. While Kustomize CAN be downloaded, installed, and run as 
a binary, an older version of Kustomize is built into kubectl. All BeeGFS 
CSI driver deployment manifests can be deployed using `kubectl apply -k`. For 
more general information about deploying the BeeGFS CSI driver, see [BeeGFS 
CSI Driver Deployment](../../docs/deployment.md)

There is no set standard for how Kustomize manifests should be laid out, but 
the BeeGFS CSI driver project follows some typical conventions. There are three 
main directories:
* *bases* contains the base deployment manifests. These manifests contain most 
  of the information required to deploy the driver. 
* *versions* contains Kubernetes version-specific patches (e.g. Kubernetes 
  v1.19 supports the fsGroupPolicy field on a CSIDriver object whereas v1.18 
  did not) that are applied over top of the base manifests during deployment. 
* *overlays* contains environment specific patches (e.g. different image names
  for air-gapped deployments or log levels for debugging) that are applied 
  over top of the base manifests and version patches during deployment. 
    * *overlays/default* is an example overlay that is ready for immediate 
      deployment if no custom configuration is required.
    * *overlays/examples* contains example configuration files and patches that 
      could be added to a custom overlay.

All of the above directories are maintained by the development team and may 
change between driver versions. **Modifying them directly will result in merge 
conflicts and other difficulties when upgrading the driver.**

As outlined in [BeeGFS CSI Driver Deployment](../../docs/deployment.md), the 
correct way to deploy the driver is to copy *overlays/default* to create a NEW 
overlay (e.g. *overlays/my-overlay*). Add patches or modify configuration 
in this overlay and use `kubectl apply -k overlays/my-overlay/` to deploy.
Modifications made to this overlay are completely protected. Any changes made 
by the development team to the base manifests or version patches will be picked 
up when you pull a new version of the project and your custom modifications will 
continue to work unless otherwise noted.

## Upgrade Notes

<a name="upgrade-1.2.0-kubernetes-deployment"></a>
### Upgrading to v1.2.0

v1.2.0 includes changes to the structure of the deployment manifests. To upgrade
from v1.1.0, follow these steps:

1. If you have made changes to the csi-beegfs-config.yaml and/or
   csi-beegfs-connauth.yaml files in the v1.1.0 *deploy/prod* directory, copy
   these files.
1. Check out v.1.2.0 of the project: `git checkout v1.2.0`.
1. Create an overlay as described
   in [Kubernetes Deployment](#kubernetes-deployment) (i.e. copy
   *deploy/k8s/overlays/default* to *deploy/k8s/overlays/my-overlay*).
1. Paste the copied files into *deploy/k8s/overlays/my-overlay*. This will
   overwrite the default (empty) files.
1. Deploy the driver: `kubectl apply -k deploy/k8s/overlays/my-overlay`.

Prior to v1.2.0, the beegfsClientConf field of the configuration file allowed 
string, integer, or boolean values. In v1.2.0, all beegfsClientConf values must 
be strings, so integers and booleans must be quoted. If you used v1.1.0 and 
specified parameters like: `connMgmtdPort: 8000` or `connUseRDMA: true`, 
modify your configuration file, specifying parameters like `connMgmtdPort: 
"8000"` or `connUseRDMA: "true"`.

<a name="upgrade-1.4.0-kubernetes-deployment"></a>
### Upgrading to v1.4.0

v1.4.0 changes the default driver container image name from
`netapp/beegfs-csi-driver` to `docker.io/netapp/beegfs-csi-driver` to
accommodate container engines (e.g. Podman) that do not treat Docker Hub as a
default. The actual hosting location of the image has not changed.

If you were previously using an overlay to override this image name, you must
update your overlay with the new default name.

This kustomization stanza:
```
images:
  - name: netapp/beegfs-csi-driver
    newName: some.location/beegfs-csi-driver
    newTag: some-tag
```

Becomes this kustomization stanza:
```
images:
  - name: docker.io/netapp/beegfs-csi-driver
    newName: some.location/beegfs-csi-driver
    newTag: some-tag
```

Note: Overriding driver container image name is not common and most overlays
will not need to be modified.

<a name="upgrade-1.5.0-kubernetes-deployment"></a>
### Upgrading to v1.5.0

v1.5.0 changes the default driver container image name from
`docker.io/netapp/beegfs-csi-driver` to
`ghcr.io/thinkparq/beegfs-csi-driver:v1.5.0` as part of migrating the BeeGFS CSI
driver to a new GitHub organization. It also changes the registry for the
Kubernetes CSI sidecar containers from `k8s.gcr.io` to `registry.k8s.io` to
accommodate the
[deprecation of k8s.gcr.io](https://kubernetes.io/blog/2023/03/10/image-registry-redirect/).

If you were previously using an overlay to override these image names, you must
update your overlay(s) with the new default names.

These kustomization stanzas:
```
images:
  - name: docker.io/netapp/beegfs-csi-driver
    newName: some.location/beegfs-csi-driver
    newTag: some-tag
  - name: k8s.gcr.io/sig-storage/csi-provisioner
    newName: some.location/csi-provisioner
    newTag: some-tag
  - name: k8s.gcr.io/sig-storage/csi-node-driver-registrar
    newName: some.location/csi-node-driver-registrar
    newTag: some-tag
  - name: k8s.gcr.io/sig-storage/liveness-probe
    newName: some.location/liveness-probe
    newTag: some-tag    
```

Becomes these kustomization stanzas:
```
images:
  - name: ghcr.io/thinkparq/beegfs-csi-driver
    newName: some.location/beegfs-csi-driver
    newTag: some-tag
  - name: registry.k8s.io/sig-storage/csi-provisioner
    newName: some.location/csi-provisioner
    newTag: some-tag
  - name: registry.k8s.io/sig-storage/csi-node-driver-registrar
    newName: some.location/csi-node-driver-registrar
    newTag: some-tag
  - name: registry.k8s.io/sig-storage/liveness-probe
    newName: some.location/liveness-probe
    newTag: some-tag           
```

Note: Overriding driver container image name is not common and most overlays
will not need to be modified.
