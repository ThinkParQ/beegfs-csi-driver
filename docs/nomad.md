# BeeGFS CSI Driver on Hashicorp Nomad

## Contents

* [Overview](#overview)
* [Maturity and Compatibility](#maturity-compatibility)
* [Deployment](#deployment)
* [Usage](#usage)
* [Known Issues](#known-issues)
  * [Driver Logs Appear Empty](#empty-logs)
  * [Podman Task Driver Unsupported](#podman-unsupported)
  * [Error Listing Nomad Volumes (Verbose)](#list-verbose-error)
* [Troubleshooting](#troubleshooting)
  * [Failed to Load a Map File](#map-file-fail)

<a name="overview"></a>
## Overview

While the BeeGFS CSI driver is primarily tested on (and intended for integration
with) Kubernetes, adherence to the Container Storage Interface (CSI) enables its
use with other container orchestrators as well. As a "simple and flexible
scheduler and orchestrator to deploy and manage containers and non-containerized
applications across on-prem and clouds at scale", [HashiCorp
Nomad](https://www.nomadproject.io/) is one such orchestrator. This document
(and other linked documents) describes the process of deploying the BeeGFS CSI
driver to a Nomad cluster and running Nomad tasks that make use of it.

<a name="maturity-compatibility"></a>
## Maturity and Compatiblity

The BeeGFS CSI driver Nomad deployment and example manifests are provided with 
an ALPHA level of maturity. 
* Basic Nomad testing has been incorporated into the BeeGFS CSI driver CI 
  pipeline and future releases of the BeeGFS CSI driver project are expected to 
  maintain the ability to deploy into Nomad.
* The format of the deployment and example manifests may change at any time in 
  backwards incompatible ways (as a result of changes to the BeeGFS CSI driver 
  or changes to Nomad).

The current BeeGFS CSI driver release is tested with the following Nomad and 
Nomad task driver versions:

| Component                 | Version  | Status | Notes                                                          |
| ------------------------- | -------- | ------ | -------------------------------------------------------------- |
| Nomad                     | 1.4.2    | pass   | Other versions 1.3.3+ MAY work. Versions 1.3.2- will NOT work. |
| Docker Task Driver        | 1.4.2    | pass   | Can consume driver volumes and deploy the driver.              |
| Isolated Exec Task Driver | 1.4.2    | pass   | Can consume driver volumes.                                    |
| Podman Task Driver        | 0.4.0    | fail   | See known issues.                                              |

<a name="deployment"></a>
## Deployment

[Manifests](../deploy/nomad/) are provided to deploy a single BeeGFS CSI driver
controller service and one node service per node to a Nomad cluster (running the
Docker task driver). See the [deployment
instructions](../deploy/nomad/README.md) for complete details. 

<a name="usage"></a>
## Usage

[Manifests](../examples/nomad/) are provided to deploy an example BeeGFS CSI
driver volume and consuming job to a Nomad cluster (running the Docker task
driver). See the [deployment instructions](../examples/nomad/README.md) for
complete details.

Note that currently Nomad does not have a dynamic provisioning equivalent to the
Kubernetes Storage Class. A Kubernetes Storage Class allows an administrator to
predefine certain storage characteristics (e.g. the sysMgmtdHost of a running
BeeGFS file system) so that these characteristics are abstracted away from a
user provisioning a volume. In Nomad, complete volume details must be defined in
the file used to deploy it.

<a name="known issues"></a>
## Known Issues

<a name="empty-logs"></a>
### Driver Logs Appear Empty

The BeeGFS CSI driver outputs all logs to stderr. While many orchestration tools
(e.g. Kubernetes) display both stdout and stderr by default when container logs 
are requested, Nomad distinguishes between the two streams. When viewing alloc 
logs in the Nomad GUI, click the stderr button. When viewing alloc logs using 
the Nomad CLI, use "nomad alloc logs -stderr".

<a name="podman-task-driver"></a>
### Podman Task Driver Unsupported

As of Podman task driver v0.4.0, CSI drivers in general, and the BeeGFS CSI 
driver in particular, do not run correctly as Podman tasks. See these Github 
issues for additional details:
* [hashicorp/nomad-driver-podman #192](https://github.com/hashicorp/nomad-driver-podman/issues/192)
* [hasicorp/nomad #15014](https://github.com/hashicorp/nomad/issues/15014)

Podman MAY be able to consume driver volumes, but this behavior is untested. The
Podman and Docker task drivers are incompatible on the same node, so Podman 
must be able to deploy the driver in order to be useful.

<a name="list-verbose-error"></a>
### Error Listing Nomad Volumes (Verbose)

Using `nomad volume status` with the `--verbose` flag results in an HTTP 500 
error code in the output. The `--verbose` flag prompts Nomad to make an 
optional ControllerListVolumes to the BeeGFS CSI driver which the driver does 
not support. The `--verbose` flag will likely never be useful for the BeeGFS CSI
driver, but future versions of Nomad should display more appropriately. See 
[hashicorp/nomad #15040](https://github.com/hashicorp/nomad/issues/15040) for 
additional details.

<a name="troubleshooting"></a>
## Troubleshooting

<a name="map file fail"></a>
### Failed to Load a Map File

This issue usually first manifests after a volume has been successfully created, 
when a task running on a node is first attempting to consume it. The driver (and 
Nomad, after contacting the driver) logs something like:

```
rpc error: code = Internal desc = beegfs-ctl failed with stdOut:  and stdErr: 
Error: Failed to load map file: 
/opt/nomad/client/csi/node/beegfs-csi-plugin/staging/beegfs-csi-volume/rw-file-system-multi-node-multi-writer/beegfs-client.conf
```

While preparing a volume for use, the node service writes a beegfs-client.conf
file to a directory within its container, then issues a mount command that makes
use of it. The mount command executes (for all intents and purposes) outside the
container, so the internal and external path to the beegfs-client.conf file must
be the same. The error message above indicates that the mount command (running
outside the container) can't see the beegfs-client.conf file (written inside the
container).

The supplied manifests use the `csi_plugin.stage_publish_base_dir` field to
ensure Nomad issues workable staging and publishing paths to the driver.
However, this field must be configured differently for different Nomad clusters,
and likely must be modified if the above error occurs. See the comments above
this field in [node.nomad](../deploy/nomad/node.nomad) for guidance.
