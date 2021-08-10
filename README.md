# BeeGFS CSI Driver

## Contents
<a name="contents"></a>

* [Overview](#overview)
* [Getting Started](#getting-started)
* [Basic Use and Examples](#basic-use)
* [Requesting Enhancements and Reporting
  Issues](#requesting-enhancements-and-reporting-issues)
* [License](#license)
* [Maintainers](#maintainers)

## Overview 
<a name="overview"></a>
The BeeGFS Container Storage Interface (CSI) driver provides high performing and
scalable storage for workloads running in container orchestrators like
Kubernetes. This driver allows containers to access existing datasets or request
on-demand ephemeral or persistent high speed storage backed by [BeeGFS parallel
file systems](https://blog.netapp.com/beegfs-for-beginners/). 

### Notable Features
<a name="notable-features"></a>
* Integration of [Storage Classes in Kubernetes](docs/usage.md#create-a-storage-class) with [storage
  pools](https://doc.beegfs.io/latest/advanced_topics/storage_pools.html) in
  BeeGFS, allowing different tiers of storage within the same file system to be
  exposed to end users. 
* Management of global and node specific BeeGFS client configuration applied to
  Kubernetes nodes, simplifying use in large environments. 
* Specify permissions in BeeGFS from Storage Classes in Kubernetes simplifying 
  integration with [BeeGFS quotas](https://doc.beegfs.io/latest/advanced_topics/quota.html#project-directory-quota-tracking) 
  and providing [visibility and control over user consumption](docs/quotas.md) 
  of the shared file system. 
* Set [striping
  parameters](https://doc.beegfs.io/latest/advanced_topics/striping.html) in
  BeeGFS from Storage Classes in Kubernetes to [optimize for diverse workloads](https://netapp.io/2021/04/06/tackling-diverse-workloads-with-beegfs-in-kubernetes/)
  sharing the same file system.
* Support for ReadWriteOnce, ReadOnlyMany, and ReadWriteMany [access
  modes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes)
  in Kubernetes allow workloads distributed across multiple Kubernetes nodes to
  share access to the same working directories and enable multi-user/application
  access to common datasets.

### Interoperability and CSI Feature Matrix
<a name="interoperability-and-csi-feature-matrix"></a>
| beegfs.csi.netapp.com  | K8s Versions     | BeeGFS Versions | CSI Version  | Persistence | Supported Access Modes   | Dynamic Provisioning |
| -----------------------| ---------------- | --------------- | ------------ | ----------- | ------------------------ | -------------------- |
| v1.0.0                 | 1.19             | 7.2, 7.1.5      | v1.3.0       | Persistent  | Read/Write Multiple Pods | Yes                  |  
| v1.1.0                 | 1.18, 1.19, 1.20 | 7.2.1, 7.1.5    | v1.3.0       | Persistent  | Read/Write Multiple Pods | Yes                  |  

Additional Notes:
* This matrix indicates tested BeeGFS and Kubernetes versions. The driver
  may work with other versions of Kubernetes, but they have not been tested. 
  Changes to the deployment manifests are likely required, especially for 
  earlier versions of Kubernetes.
* The driver has not been tested with SELinux.
* For environments where the driver is used with both BeeGFS 7.1.x and 
  7.2.x, Kubernetes nodes should have the 7.2 BeeGFS DKMS client installed.

## Getting Started 
<a name="getting-started"></a>

### Prerequisite(s) 
<a name="prerequisites"></a>
* Deploying the driver requires access to a terminal with kubectl. 
* The [BeeGFS DKMS
  client](https://doc.beegfs.io/latest/advanced_topics/client_dkms.html) must be
  preinstalled to each Kubernetes node that needs BeeGFS access.
  * Note: As part of this setup the beegfs-helperd and beegfs-utils packages must 
  be installed, and the `beegfs-helperd` service must be started and enabled.  
* Each BeeGFS mount point uses an ephemeral UDP port. On Linux the selected
  ephemeral port is constrained by the values of [IP
  variables](https://www.kernel.org/doc/html/latest/networking/ip-sysctl.html#ip-variables).
  [Ensure that firewalls allow UDP
  traffic](https://doc.beegfs.io/latest/advanced_topics/network_tuning.html#firewalls-network-address-translation-nat)
  between BeeGFS management/metadata/storage nodes and ephemeral ports on
  Kubernetes nodes.
* One or more existing BeeGFS file systems should be available to the Kubernetes
  nodes over a TCP/IP and/or RDMA (InfiniBand/RoCE) capable network (not
  required to deploy the driver).

### Quick Start
<a name="quick-start"></a>
The steps in this section allow you to get the driver up and running quickly.
For production use cases or air-gapped environments it is recommended to read
through the full [deployment guide](docs/deployment.md). 

1. On a machine with kubectl and access to the Kubernetes cluster where you want
   to deploy the BeeGFS CSI driver clone this repository: `git clone
   https://github.com/NetApp/beegfs-csi-driver.git`
2. Change to the BeeGFS CSI driver directory (`cd beegfs-csi-driver`) and run:
   `kubectl apply -k deploy/k8s/prod`
    * Note by default the beegfs-csi-driver image will be pulled from
      [DockerHub](https://hub.docker.com/r/netapp/beegfs-csi-driver).
3. Verify all components are installed and operational: `kubectl get pods -n
   kube-system | grep csi-beegfs`

As a one-liner: `git clone https://github.com/NetApp/beegfs-csi-driver.git && cd
beegfs-csi-driver && kubectl apply -k deploy/k8s/prod && kubectl get pods -n
kube-system | grep csi-beegfs`

Provided all Pods are running the driver is now ready for use. See the following
sections for how to get started using the driver.

## Basic Use
<a name="basic-use"></a>
 This section provides a quick summary of basic driver use and functionality.
 Please see the full [usage documentation](docs/usage.md) for a complete
 overview of all available functionality. The driver was designed to support
 both dynamic and static storage provisioning and allows directories in BeeGFS
 to be used as [Persistent
 Volumes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) (PVs)
 in Kubernetes. Pods with Persistent Volume Claims (PVCs) are only able to
 see/access the specified directory (and any subdirectories), providing
 isolation between multiple applications and users using the same BeeGFS file
 system when desired. 

### Dynamic Storage Provisioning:
<a name="dynamic-storage-provisioning"></a>
Administrators create a Storage Class in Kubernetes referencing at minimum a
specific BeeGFS file system and parent directory within that file system. Users
can then submit PVCs against the Storage Class, and are provided isolated access
to new directories under the parent specified in the Storage Class. 

### Static Provisioning:
<a name="static-provisioning"></a>
Administrators create a PV and PVC representing an existing directory in a
BeeGFS file system. This is useful for exposing some existing dataset or shared
directory to Kubernetes users and applications.

### Examples
<a name="examples"></a>
[Example Kubernetes manifests](examples/k8s/README.md) of how to use the driver are
provided. These are meant to be repurposed to simplify creating objects related
to the driver including Storage Classes, Persistent Volumes, and Persistent
Volume Claims in your environment.

## Requesting Enhancements and Reporting Issues 
<a name="requesting-enhancements-and-reporting-issues"></a>
If you have any questions, feature requests, or would like to report an issue
please submit them at https://github.com/NetApp/beegfs-csi-driver/issues. 

## License 
<a name="license"></a>
Apache License 2.0

## Maintainers 
<a name="maintainers"></a>
* Austin Major (@austinmajor).
* Eric Weber (@ejweber).
* Jason Eastburn
* Joe McCormick (@iamjoemccormick).
* Joey Parnell (@unwieldy0). 
* Justin Bostian (@jb5n).
