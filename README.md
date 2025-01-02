# BeeGFS CSI Driver <!-- omit in toc -->

[![Go report card](https://goreportcard.com/badge/github.com/thinkparq/beegfs-csi-driver)](https://goreportcard.com/report/github.com/thinkparq/beegfs-csi-driver)
[![License](https://img.shields.io/github/license/thinkparq/beegfs-csi-driver)](LICENSE)
[![Build, Test, and Publish](https://github.com/ThinkParQ/beegfs-csi-driver/actions/workflows/build-test-publish.yaml/badge.svg)](https://github.com/ThinkParQ/beegfs-csi-driver/actions/workflows/build-test-publish.yaml)
[![Code scanning using CodeQL](https://github.com/ThinkParQ/beegfs-csi-driver/actions/workflows/codeql.yaml/badge.svg)](https://github.com/ThinkParQ/beegfs-csi-driver/actions/workflows/codeql.yaml)

## Contents <!-- omit in toc -->

- [Overview](#overview)
  - [Notable Features](#notable-features)
- [Compatibility](#compatibility)
  - [Known Incompatibilities](#known-incompatibilities)
    - [BeeGFS CSI Driver compatibility with BeeGFS 7.2.7+ and 7.3.1+](#beegfs-csi-driver-compatibility-with-beegfs-727-and-731)
- [Support](#support)
- [Getting Started](#getting-started)
  - [Prerequisite(s)](#prerequisites)
  - [Quick Start](#quick-start)
- [Basic Use](#basic-use)
  - [Dynamic Storage Provisioning:](#dynamic-storage-provisioning)
  - [Static Provisioning:](#static-provisioning)
  - [Examples](#examples)
- [Contributing to the Project](#contributing-to-the-project)
- [Releases](#releases)
- [Versioning](#versioning)
- [License](#license)

***

<a name="overview"></a>
## Overview 

The BeeGFS Container Storage Interface (CSI) driver provides high performing and
scalable storage for workloads running in container orchestrators like
Kubernetes. This driver allows containers to access existing datasets or request
on-demand ephemeral or persistent high speed storage backed by [BeeGFS parallel
file systems](https://blog.netapp.com/beegfs-for-beginners/). 

The driver can be easily deployed using the provided Kubernetes manifests.
Optionally the [BeeGFS CSI Driver Operator](operator/README.md) can be used to
automate day-1 (install/ configure) and day-2 (reconfigure/update) tasks for the
driver. This especially simplifies discovery and installation from Operator
Lifecycle Manger (OLM) enabled clusters. Multi-arch images supporting amd64 and
arm64 Kubernetes nodes are provided for the BeeGFS CSI driver and operator.

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

***

<a name="compatibility"></a>
## Compatibility

The BeeGFS CSI driver must interact with both Kubernetes and a BeeGFS
filesystem. To ensure compatibility with relevant versions of these key software
components regular testing is done throughout each release cycle. The following
table describes the versions of each component used in testing each release of
the BeeGFS CSI driver. These configurations should be considered compatible and
supported.

| BeeGFS CSI Driver | K8s Versions                              | BeeGFS Client Versions | CSI Version |
| ----------------- | ----------------------------------------- | ---------------------- | ----------- |
| v1.7.0            | 1.27.11, 1.28.7, 1.29.10, 1.30.6, 1.31.2  | 7.3.4, 7.4.5           | v1.10.0     |
| v1.6.0            | 1.25.16, 1.26.14, 1.27.11, 1.28.7         | 7.3.4, 7.4.2           | v1.8.0      |
| v1.5.0            | 1.23.17, 1.24.15, 1.25.11, 1.26.3, 1.27.3 | 7.3.4, 7.4.0           | v1.7.0      |
| v1.4.0            | 1.22.6, 1.23.5, 1.24.1, 1.25.2            | 7.3.2, 7.2.8           | v1.7.0      |
| v1.3.0            | 1.21.4, 1.22.3, 1.23.1, 1.24.1            | 7.3.1, 7.2.7           | v1.6.0      |
| v1.2.2            | 1.20.11, 1.21.4, 1.22.3, 1.23.1           | 7.3.0, 7.2.6 [^1]      | v1.5.0      |
| v1.2.1            | 1.19.15, 1.20.11, 1.21.4, 1.22.3          | 7.2.5 [^1]             | v1.5.0      |
| v1.2.0            | 1.18, 1.19, 1.20, 1.21                    | 7.2.4 [^1]             | v1.5.0      |
| v1.1.0            | 1.18, 1.19, 1.20                          | 7.2.1 [^1]             | v1.3.0      |
| v1.0.0            | 1.19                                      | 7.2 [^1]               | v1.3.0      |

Additional notes:
* Starting with v1.6.0 official multi-arch container images are provided for both amd64 and arm64.
* The BeeGFS CSI driver offers experimental support for [Hashicorp Nomad](docs/nomad.md).
* As of v1.5.0 the BeeGFS CSI driver is [no longer tested](docs/compatibility.md#openshift) with Red Hat OpenShift.

See the [compatibility guide](docs/compatibility.md) for more details on
expectations of compatibility for the BeeGFS CSI driver.

### Known Incompatibilities

#### BeeGFS CSI Driver compatibility with BeeGFS 7.2.7+ and 7.3.1+
Versions of the BeeGFS CSI driver prior to v1.3.0 are known to have issues
initializing the driver when used in conjunction with BeeGFS clients 7.2.7 or
7.3.1. These issues relate to the changes in these BeeGFS versions that require
[Connection Authentication configuration to be
set](docs/deployment.md#connauth-configuration). The v1.3.0 release of the
driver resolves these issues and maintains compatibility with the prior BeeGFS
versions (7.2.6 and 7.3.0). Therefore, in an environment where an existing
installation is upgrading from BeeGFS 7.2.6 or 7.3.0 to 7.2.7 or 7.3.1 the
recommendation would be to upgrade the BeeGFS CSI driver to v1.3.0 or later
before upgrading the BeeGFS clients.

[^1]: Support for the BeeGFS 7.1.5 filesystem is provided when the BeeGFS 7.2.x
    client is used. These configurations were tested in that manner.

***
<a name="support"></a>
## Support

Support for the BeeGFS CSI driver is "best effort". The following policy is in
no way binding and may change without notice.

Only the latest version of the BeeGFS CSI driver is supported. Bugs or
vulnerabilities found in this version may be fixed in a patch release or may be
fixed in a new minor version. If they are fixed in a new minor version,
upgrading to this version may be required to obtain the fix. 

Fixes for old minor versions of the driver or backporting fixes to an older
minor release of the driver should not be expected. The maintainers may choose
to release a fix in a patch for an older release at their discretion.

Support for the BeeGFS driver can be expected when the driver is used with
components listed in the compatibility table. The ability to provide support for
issues with components outside of the compatibility matrix will depend on the
details of the issue.

If you have any questions, feature requests, or would like to report an issue
please submit them at https://github.com/ThinkParQ/beegfs-csi-driver/issues. 

***

<a name="getting-started"></a>
## Getting Started

<a name="prerequisites"></a>
### Prerequisite(s)

* Deploying the driver requires access to a terminal with kubectl. 
* The [BeeGFS DKMS
  client](https://doc.beegfs.io/latest/advanced_topics/client_dkms.html) must be
  preinstalled to each Kubernetes node that needs BeeGFS access.
  * As part of this setup the beegfs-helperd and beegfs-utils packages must be
    installed, and the `beegfs-helperd` service must be started and enabled.
  * For BeeGFS versions 7.3.1+ or 7.2.7+, the `beegfs-helperd` service must be
    configured with `connDisableAuthentication = true` or `connAuthFile = <path
    to a connAuthFile shared by all file systems>`. See [BeeGFS Helperd
    Configuration](docs/deployment.md#beegfs-helperd-configuration) for other
    options or more details.
  * [Experimental support](deploy/openshift-beegfs-client/README.md) for
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
See them in action on [NetApp
TV](https://www.youtube.com/watch?v=uu7q_PcHUXA&list=PLdXI3bZJEw7kJrbLsSFvDGXWJEmqHOFj_).
For production use cases or air-gapped environments it is recommended to read
through the full [kubectl deployment guide](docs/deployment.md) or [operator
deployment guide](operator/README.md).

1. On a machine with kubectl and access to the Kubernetes cluster where you want
   to deploy the BeeGFS CSI driver clone this repository: `git clone
   https://github.com/ThinkParQ/beegfs-csi-driver.git`.
2. Change to the BeeGFS CSI driver directory (`cd beegfs-csi-driver`).
3. In BeeGFS versions 7.3.1+ or 7.2.7+, explicit connAuth configuration is
   required. Do one of the following or see [ConnAuth
   Configuration](#docs/deployment.md#connauth-configuration) for more details.
   * Set connDisableAuthentication to true in `csi-beegfs-config.yaml` if your
     existing file system does not use connection authentication.
     ```yaml
     config:
       beegfsClientConf:
         connDisableAuthentication: "true" # note the quotes are required
     ```
   * Provide connAuth details in `csi-beegfs-connauth.yaml` if your existing
     file system does use connection authentication.
     ```yaml
     - sysMgmtdHost: <sysMgmtdHost>
       connAuth: <connAuthSecret>
       encoding: <encodingType> # raw or base64
     ```
4. Run `kubectl apply -k deploy/k8s/overlays/default`. Note by default the
   beegfs-csi-driver image will be pulled from
   [GitHub Container Registry](https://github.com/ThinkParQ/beegfs-csi-driver/pkgs/container/beegfs-csi-driver).
5. Verify all components are installed and operational: `kubectl get pods -n
   beegfs-csi`.

Provided all Pods are running the driver is now ready for use. See the following
sections for how to get started using the driver.

***

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
to new directories under the parent specified in the Storage Class. See the process
in action on [NetApp TV](https://www.youtube.com/watch?v=vJ9nrb_1aXY&list=PLdXI3bZJEw7kJrbLsSFvDGXWJEmqHOFj_&index=2).

<a name="static-provisioning"></a>
### Static Provisioning:

Administrators create a PV and PVC representing an existing directory in a
BeeGFS file system. This is useful for exposing some existing dataset or shared
directory to Kubernetes users and applications. See the process in action on 
[NetApp TV](https://www.youtube.com/watch?v=-KTCFuA5-Cc&list=PLdXI3bZJEw7kJrbLsSFvDGXWJEmqHOFj_&index=3).

<a name="examples"></a>
### Examples

[Example Kubernetes manifests](examples/k8s/README.md) of how to use the driver are
provided. These are meant to be repurposed to simplify creating objects related
to the driver including Storage Classes, Persistent Volumes, and Persistent
Volume Claims in your environment.

***

<a name="contributing"></a>
## Contributing to the Project
The BeeGFS CSI Driver maintainers welcome improvements from the BeeGFS and 
open source community! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for how 
to get started.

***

<a name="releases"></a>
## Releases
The goal is to release a new driver version three to four times per year 
(roughly quarterly). Releases may be major, minor, or patch at the discretion 
of the maintainers in accordance with needs of the community (i.e. large 
features, small features, or miscellaneous bug fixes).

***

<a name="versioning"></a>
## Versioning

The BeeGFS CSI driver versioning is based on the semantic versioning scheme 
outlined at [semver.org](https://semver.org/). According to this scheme, 
given a version number MAJOR.MINOR.PATCH, we increment the:
  * MAJOR version when:
    * We make significant code changes beyond just a new feature.
    * Backwards incompatible changes are made.
  * MINOR version when:
    * New driver features are added.
    * New versions of Kubernetes or BeeGFS are supported.
  * PATCH version when: small bug or security fixes are needed in a more timely
    manner.

When upgrading the driver using default Kustomize (`kubectl`) deployment option,
it is recommended to reference the [upgrade
notes](deploy/k8s/README.md#upgrade-notes) for your particular upgrade path.
While the driver itself may be backwards compatible, if you used a non-standard
file layout or customized the Kubernetes manifests used to deploy the driver,
you may need to make adjustments to your manifests.

***

<a name="license"></a>
## License 

Apache License 2.0

***

