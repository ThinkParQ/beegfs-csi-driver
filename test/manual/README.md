## Purpose

These files facilitate the creation of the following resources:
* 4x statically provisioned pvcs, 2x from one file system and 2x from another
* 1x statically provisioned pvc intended for read-only access
* 4x dynamically provisioned pvcs, 2x from one file system and 2x from another
* a stateful set with 4x pods where each pod mounts and writes to all 8x 
  writeable pvcs

Use these resources to perform various manual tests. For example:
* Verify that all pods can read and write from all volumes
* Verify that data written in one pod can be read from another
* Verify that the driver cleans up on a DeleteVolume call as expected
* Verify that various failures do not cause data loss
* Verfiy that a non-standard file system configuration (e.g. 
  connMgmtdPortTCP != 8008) does not break functionality

## Setup

### BeeGFS

Create 2x BeeGFS file systems. For example, the ones used to test driver v1.0.0 
functionality are addressable (NetApp internal) at:

BeeGFS 7.2
* scspa2058537001.rtp.openenglab.netapp.com (standard 8008 mgmtd port)
* scspa2060446001.rtp.openenglab.netapp.com (9009 mgmtd port)

BeeGFS 7.1.5
* scspa2059245001.rtp.openenglab.netapp.com (standard 8008 mgmtd port)
* scspa2061750001.rtp.openenglab.netapp.com (9009 mgmtd port)

Ensure both filesystems have multiple storage targets and at least one 
non-default storage pool (ID = 2).

Within each filesystem create the following directories:
* /test/static/pv1/ (simulates a pre-existing, statically provisioned volume)
* /test/static/pv2/ (simulates a pre-existing, statically provisioned volume)
* /test/static/pvro/ (simulates a pre-existing, statically provisioned volume 
  intended for read-only access)
* /test/dyn/ (volDirBasePath for dynamic provisioning)

### Kubernetes

1. Create a Kubernetes cluster with at least 2x worker nodes and 1x master node. 
1. Ensure the correct version of BeeGFS is installed on all nodes.
1. Ensure your environment is set up (e.g. set the KUBECONFIG environment 
   variable) for `kubectl` access to the cluster.

#### Driver Deployment

1. Copy ./<overlay>/csi-beegfs-config.yaml to deploy/dev/csi-beegfs-config.yaml. 
   This allows access to the second file system on its non-standard 9009 mgmtd 
   port. 
1. Copy ./<overlay>/csi-beegfs-connauth.yaml (if it exists) to 
   deploy/dev/csi-beegfs-connauth.yaml. This allows access to the second file 
   system using a connAuthFile.
1. Do the standard `kubectl apply -k deploy/dev/`.

#### Resource Deployment

The inline patching used by the kustomization.yaml files in this directory is 
NOT supported in kubectl. As of Kubernetes v1.19.0, kubectl includes Kustomize 
v2.0.3, and Kustomize v3+ is required.

```
go get sigs.k8s.io/kustomize/kustomize/v3
# ensure that your path includes #GOPATH/bin
kustomize build test/manual/<overlay> | kubectl apply -f -
```