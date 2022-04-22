# BeeGFS CSI Driver

[![License](https://img.shields.io/github/license/netapp/beegfs-csi-driver)](LICENSE)
[![Docker pulls](https://img.shields.io/docker/pulls/netapp/beegfs-csi-driver)](https://hub.docker.com/r/netapp/beegfs-csi-driver)
[![Go report card](https://goreportcard.com/badge/github.com/netapp/beegfs-csi-driver)](https://goreportcard.com/report/github.com/netapp/beegfs-csi-driver)

<a name="contents"></a>
## Contents

* [Overview](#overview)
* [Getting Started](#getting-started)
* [Basic Use and Examples](#basic-use)
* [Requesting Enhancements and Reporting
  Issues](#requesting-enhancements-and-reporting-issues)
* [License](#license)
* [Maintainers](#maintainers)

<a name="overview"></a>
## Overview 

The BeeGFS Container Storage Interface (CSI) driver provides high performing and
scalable storage for workloads running in container orchestrators like
Kubernetes. This driver allows containers to access existing datasets or request
on-demand ephemeral or persistent high speed storage backed by [BeeGFS parallel
file systems](https://blog.netapp.com/beegfs-for-beginners/). 

The driver can be easily deployed using the provided Kubernetes manifests. Optionally the 
[BeeGFS CSI Driver Operator](operator/README.md) can be used to automate day-1 (install/
configure) and day-2 (reconfigure/update) tasks for the driver. This especially simplifies 
discovery and installation from Operator Lifecycle Manger (OLM) enabled clusters like Red Hat 
OpenShift. 

<a name="notable-features"></a>
### Notable Features

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

<a name="interoperability-and-csi-feature-matrix"></a>
### Interoperability and CSI Feature Matrix

| beegfs.csi.netapp.com  | K8s Versions                     | Red Hat OpenShift Versions           | BeeGFS Versions | CSI Version  |
| ---------------------- | -------------------------------- | ------------------------------------ | --------------- | ------------ |
| v1.0.0                 | 1.19                             |                                      | 7.2, 7.1.5      | v1.3.0       |
| v1.1.0                 | 1.18, 1.19, 1.20                 |                                      | 7.2.1, 7.1.5    | v1.3.0       |
| v1.2.0                 | 1.18, 1.19, 1.20, 1.21           | 4.8  (RHEL only)                     | 7.2.4, 7.1.5    | v1.5.0       |
| v1.2.1                 | 1.19.15, 1.20.11, 1.21.4, 1.22.3 | 4.9  (RHEL only)                     | 7.2.5, 7.1.5    | v1.5.0       |
| v1.2.2                 | 1.20.11, 1.21.4, 1.22.3, 1.23.1  | 4.10 (RHEL only; RHCOS experimental) | 7.2.6, 7.1.5    | v1.5.0       |

The following CSI features are supported by all versions of the driver:
* Access Modes: Read/Write Multiple Pods
* Dynamic Provisioning: Yes 
* Persistence: Yes

Additional Notes:
* The BeeGFS CSI driver is released according to the semantic versioning scheme 
  outlined at [semver.org](https://semver.org/). According to this scheme, 
  given a version number MAJOR.MINOR.PATCH, we increment the:
  * MAJOR version when we make incompatible API changes,
  * MINOR version when we add functionality in a backwards compatible manner, 
    and
  * PATCH version when we make backwards compatible bug fixes.
* This matrix indicates tested BeeGFS and Kubernetes versions. The driver
  may work with other versions of Kubernetes, but they have not been tested. 
  Changes to the deployment manifests are likely required, especially for 
  earlier versions of Kubernetes.
* It is generally recommended to run the driver on the latest version of 
  Kubernetes supported by a given version of the driver. While an older version 
  of Kubernetes may appear to work, it may not include critical fixes that 
  ensure driver stability.
* For environments where the driver is used with both BeeGFS 7.1.x and 
  7.2.x, Kubernetes nodes should have the 7.2 BeeGFS DKMS client installed.

### Support Policy

Support for the BeeGFS CSI driver is "best effort". The maintainers will make
every attempt to fix all known bugs, release new features, and maintain 
compatibility with new container orchestrators, but the following policy is in 
no way binding and may change over time.

Only the latest version of the BeeGFS CSI driver is supported. Bugs or 
vulnerabilities found in this version may be fixed in a patch release or may be 
fixed in a new minor version. If they are fixed in a new minor version, 
upgrading to this version may be required to obtain the fix. 

Note: The BeeGFS CSI driver maintainers may choose to release a patch for a 
previous minor version with a backported fix, but this is not the norm.

The latest version of the driver is only supported on certain versions of 
Kubernetes and OpenShift. It may be necessary to upgrade Kubernetes or 
OpenShift to maintain driver support.

The goal is to release a new driver version three to four times per year 
(roughly quarterly). Releases may be major, minor, or patch at the discretion 
of the maintainers in accordance with needs of the community (i.e. large 
features, small features, or miscellaneous bug fixes).

#### Kubernetes

A new minor version of the driver will be tested on, and will include deployment 
manifests for, any Kubernetes version that meets the following criteria:
* It is able to be set up via a released version of
  [Kubespray](https://github.com/kubernetes-sigs/kubespray) (used to maintain
  BeeGFS CSI driver test environments).
* It is still supported by the Kubernetes community (see the [currently 
  supported Kubernetes releases](https://kubernetes.io/releases/) and 
  [Kubernetes support 
  policy](https://kubernetes.io/releases/version-skew-policy/)) OR it is one 
  version out of support but provides no major obstacles to driver deployment 
  and operation.

Note: We make a "best effort" to maintain compatibility with one out-of-support 
version as an acknowledgement that Kubernetes has a fast moving release cycle 
and upgrading environments can take time. However, if any issues arise when 
using the driver on a Kubernetes version that is out of support, the first 
recommendation is to upgrade Kubernetes.

Occasionally, a particular Kubernetes patch version may be required to 
guarantee smooth driver operation. See the [Troubleshooting 
Guide](docs/troubleshooting.md) for known issues.

#### OpenShift

A new minor version of the driver and the operator that can be used to deploy 
and/or upgrade the driver will be tested on the latest supported version of 
OpenShift.

#### Nomad

While we have made [initial investments](deploy/nomad/README.md) into enabling 
the use of the BeeGFS CSI driver with HashiCorp Nomad, we may not test with 
Nomad for every driver release and do not currently consider Nomad to be a 
supported container orchestrator.

<a name="getting-started"></a>
## Getting Started

<a name="prerequisites"></a>
### Prerequisite(s)

* Deploying the driver requires access to a terminal with kubectl. 
* The [BeeGFS DKMS
  client](https://doc.beegfs.io/latest/advanced_topics/client_dkms.html) must be
  preinstalled to each Kubernetes node that needs BeeGFS access.
  * Note: As part of this setup the beegfs-helperd and beegfs-utils packages must 
    be installed, and the `beegfs-helperd` service must be started and enabled.
  * Note: [Experimental support](deploy/openshift-beegfs-client/README.md) for 
    OpenShift environments with RedHat CoreOS nodes negates this requirement.
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

<a name="quick-start"></a>
### Quick Start

The steps in this section allow you to get the driver up and running quickly.
For production use cases or air-gapped environments it is recommended to read
through the full [kubectl deployment guide](docs/deployment.md) or [operator 
deployment guide](operator/README.md).

1. On a machine with kubectl and access to the Kubernetes cluster where you want
   to deploy the BeeGFS CSI driver clone this repository: `git clone
   https://github.com/NetApp/beegfs-csi-driver.git`
2. Change to the BeeGFS CSI driver directory (`cd beegfs-csi-driver`) and run:
   `kubectl apply -k deploy/k8s/overlays/default`
    * Note by default the beegfs-csi-driver image will be pulled from
      [DockerHub](https://hub.docker.com/r/netapp/beegfs-csi-driver).
3. Verify all components are installed and operational: `kubectl get pods -n
   beegfs-csi`

As a one-liner: `git clone https://github.com/NetApp/beegfs-csi-driver.git && cd
beegfs-csi-driver && kubectl apply -k deploy/k8s/overlays/default && kubectl get 
pods -n beegfs-csi`

Provided all Pods are running the driver is now ready for use. See the following
sections for how to get started using the driver.

<a name="basic-use"></a>
## Basic Use

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

<a name="dynamic-storage-provisioning"></a>
### Dynamic Storage Provisioning:

Administrators create a Storage Class in Kubernetes referencing at minimum a
specific BeeGFS file system and parent directory within that file system. Users
can then submit PVCs against the Storage Class, and are provided isolated access
to new directories under the parent specified in the Storage Class. 

<a name="static-provisioning"></a>
### Static Provisioning:

Administrators create a PV and PVC representing an existing directory in a
BeeGFS file system. This is useful for exposing some existing dataset or shared
directory to Kubernetes users and applications.

<a name="examples"></a>
### Examples

[Example Kubernetes manifests](examples/k8s/README.md) of how to use the driver are
provided. These are meant to be repurposed to simplify creating objects related
to the driver including Storage Classes, Persistent Volumes, and Persistent
Volume Claims in your environment.

<a name="requesting-enhancements-and-reporting-issues"></a>
## Requesting Enhancements and Reporting Issues 

If you have any questions, feature requests, or would like to report an issue
please submit them at https://github.com/NetApp/beegfs-csi-driver/issues. 

## Contributing to the Project
The BeeGFS CSI Driver maintainers welcome improvements from the BeeGFS and 
open source community! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for how 
to get started.

<a name="license"></a>
## License 

Apache License 2.0

<a name="maintainers"></a>
## Maintainers 

* Joe McCormick (@iamjoemccormick).
* Eric Weber (@ejweber).
* Garrett Marks (@gmarks-ntap).
* Cole Krizek (@ckrizek).
