# BeeGFS CSI Driver HashiCorp Nomad Deployment

While the BeeGFS CSI driver is primarily tested on (and intended for integration
with) Kubernetes, adherence to the Container Storage Interface (CSI) enables its
use with other container orchestrators as well. As a "simple and flexible
scheduler and orchestrator to deploy and manage containers and non-containerized
applications across on-prem and clouds at scale", [HashiCorp Nomad]
(https://www.nomadproject.io/) is one such orchestrator.

## Contents

* [Important Warnings](#important-warnings)
* [What This Is For](#what-this-is-for)
* [Requirements](#requirements)
* [Steps](#steps)
  * [Deploy](#deploy)
  * [Test](#test)
  * [Clean Up](#clean-up)
* [Troubleshooting](#troubleshooting)
  * [The driver doesn't appear to log anything.](#empty-logs)
  * [The node service fails to "load map file.](#map-file)

***

## Important Warnings

At this time the BeeGFS CSI driver is NOT SUPPORTED on Nomad, and is for
demonstration and development purposes only. DO NOT USE IT IN PRODUCTION.

These manifests rely on functionality introduced into Nomad with [Nomad PR 
#13919](https://github.com/hashicorp/nomad/pull/13919). This functionality is 
on track for inclusion in Nomad v1.3.3. However, the latest generally available 
version of Nomad (as of this writing) is v1.3.2. For now, testing of these 
manifests is done using developer builds of the Nomad [release/1.3.x 
branch](https://github.com/hashicorp/nomad/tree/release/1.3.x).

***

## What This Is For

At a high level, these manifests consists of two separate Nomad jobs. Together,
they get the BeeGFS CSI driver up and running in a Nomad cluster. Apply them
BEFORE any [example volumes or jobs](../../examples/nomad/README.md) that
require the BeeGFS CSI driver.
1. `controller.nomad` runs the CSI controller service as a "service" type job
   with a single replica.
1. `node.nomad` runs the CSI node service as a "system" type job so that a
   replica runs on every node in the cluster.

***

## Requirements

* An existing BeeGFS file system.
* An existing Nomad cluster running an [appropriate version of
  Nomad](#important-warnings). On each client node:
    * Preinstall the
      [beegfs-client-dkms](https://doc.beegfs.io/latest/advanced_topics/client_dkms.html)
      and beegfs-util packages.
    * Enable the [Docker task
      driver](#https://www.nomadproject.io/docs/drivers/docker#docker-driver).
    * [Configure the Docker task
      driver](https://www.nomadproject.io/docs/drivers/docker#docker-driver) so
      that allow_privileged = true. 
* Place valid
  [csi-beegfs-config.yaml](../../docs/deployment.md#managing-beegfs-client-configuration)
  contents in both controller.nomad and node.nomad if necessary for your
  environment.
* Place valid
  [csi-beegfs-connauth.yaml](../../docs/deployment.md#connauth-configuration)
  contents in both `controller.nomad` and `node.nomad` if necessary for your
  environment.
* Modify any additional fields marked LIKELY TO REQUIRE MODIFICATION in both
  `controller.nomad` and `node.nomad`.

***

## Steps

### Deploy

`nomad job run test/nomad/beegfs-7.3-rh8/controller.nomad`
```
==> 2022-07-25T15:47:23-05:00: Monitoring evaluation "c1581366"
    2022-07-25T15:47:23-05:00: Evaluation triggered by job "beegfs-csi-plugin-controller"
==> 2022-07-25T15:47:24-05:00: Monitoring evaluation "c1581366"
    2022-07-25T15:47:24-05:00: Evaluation within deployment: "c4de7e3a"
    2022-07-25T15:47:24-05:00: Allocation "7d141b6c" created: node "011c950b", group "controller"
    2022-07-25T15:47:24-05:00: Evaluation status changed: "pending" -> "complete"
==> 2022-07-25T15:47:24-05:00: Evaluation "c1581366" finished with status "complete"
==> 2022-07-25T15:47:24-05:00: Monitoring deployment "c4de7e3a"
  ✓ Deployment "c4de7e3a" successful
    
    2022-07-25T15:47:35-05:00
    ID          = c4de7e3a
    Job ID      = beegfs-csi-plugin-controller
    Job Version = 0
    Status      = successful
    Description = Deployment completed successfully
    
    Deployed
    Task Group  Desired  Placed  Healthy  Unhealthy  Progress Deadline
    controller  1        1       1        0          2022-07-25T15:57:33-05:00
```

`nomad job run test/nomad/beegfs-7.3-rh8/node.nomad`
```
==> 2022-07-25T15:47:44-05:00: Monitoring evaluation "842ad41f"
    2022-07-25T15:47:44-05:00: Evaluation triggered by job "beegfs-csi-plugin-node"
==> 2022-07-25T15:47:45-05:00: Monitoring evaluation "842ad41f"
    2022-07-25T15:47:45-05:00: Allocation "1891d9ca" created: node "de35373e", group "node"
    2022-07-25T15:47:45-05:00: Allocation "505e02fc" created: node "011c950b", group "node"
    2022-07-25T15:47:45-05:00: Evaluation status changed: "pending" -> "complete"
==> 2022-07-25T15:47:45-05:00: Evaluation "842ad41f" finished with status "complete"
```

`nomad job status beegfs-csi-plugin-controller`
```
ID            = beegfs-csi-plugin-controller
Name          = beegfs-csi-plugin-controller
Submit Date   = 2022-07-25T15:47:23-05:00
Type          = service
Priority      = 50
Datacenters   = dc1
Namespace     = default
Status        = running
Periodic      = false
Parameterized = false

Summary
Task Group  Queued  Starting  Running  Failed  Complete  Lost  Unknown
controller  0       0         1        0       0         0     0

Latest Deployment
ID          = c4de7e3a
Status      = successful
Description = Deployment completed successfully

Deployed
Task Group  Desired  Placed  Healthy  Unhealthy  Progress Deadline
controller  1        1       1        0          2022-07-25T15:57:33-05:00

Allocations
ID        Node ID   Task Group  Version  Desired  Status   Created  Modified
7d141b6c  011c950b  controller  0        run      running  28s ago  17s ago
```

`nomad job status beegfs-csi-plugin-node`
```
ID            = beegfs-csi-plugin-node
Name          = beegfs-csi-plugin-node
Submit Date   = 2022-07-25T15:47:44-05:00
Type          = system
Priority      = 50
Datacenters   = dc1
Namespace     = default
Status        = running
Periodic      = false
Parameterized = false

Summary
Task Group  Queued  Starting  Running  Failed  Complete  Lost  Unknown
node        0       0         2        0       0         0     0

Allocations
ID        Node ID   Task Group  Version  Desired  Status   Created  Modified
1891d9ca  de35373e  node        0        run      running  11s ago  11s ago
505e02fc  011c950b  node        0        run      running  11s ago  11s ago
```

`nomad plugin status --verbose`
```
Container Storage Interface
ID                 Provider               Controllers Healthy/Expected  Nodes Healthy/Expected
beegfs-csi-plugin  beegfs.csi.netapp.com  1/1                           2/2
```

### Test

See the [Nomad usage examples](../../examples/nomad/README.md).

### Clean Up

`nomad job stop -purge beegfs-csi-plugin-controller`
```
==> 2022-07-25T15:51:19-05:00: Monitoring evaluation "0fec5f9c"
    2022-07-25T15:51:19-05:00: Evaluation triggered by job "beegfs-csi-plugin-controller"
==> 2022-07-25T15:51:20-05:00: Monitoring evaluation "0fec5f9c"
    2022-07-25T15:51:20-05:00: Evaluation within deployment: "c4de7e3a"
    2022-07-25T15:51:20-05:00: Evaluation status changed: "pending" -> "complete"
==> 2022-07-25T15:51:20-05:00: Evaluation "0fec5f9c" finished with status "complete"
==> 2022-07-25T15:51:20-05:00: Monitoring deployment "c4de7e3a"
  ✓ Deployment "c4de7e3a" successful
    
    2022-07-25T15:51:20-05:00
    ID          = c4de7e3a
    Job ID      = beegfs-csi-plugin-controller
    Job Version = 0
    Status      = successful
    Description = Deployment completed successfully
    
    Deployed
    Task Group  Desired  Placed  Healthy  Unhealthy  Progress Deadline
    controller  1        1       1        0          2022-07-25T15:57:33-05:00
```

`nomad job stop -purge beegfs-csi-plugin-node`
```
==> 2022-07-25T15:51:48-05:00: Monitoring evaluation "aea2aacc"
    2022-07-25T15:51:48-05:00: Evaluation triggered by job "beegfs-csi-plugin-node"
==> 2022-07-25T15:51:49-05:00: Monitoring evaluation "aea2aacc"
    2022-07-25T15:51:49-05:00: Evaluation status changed: "pending" -> "complete"
==> 2022-07-25T15:51:49-05:00: Evaluation "aea2aacc" finished with status "complete"
```

## Troubleshooting

<a name="empty-logs"></a>
### The driver doesn't appear to log anything.

The BeeGFS CSI driver outputs all logs to stderr. While many orchestration tools
(e.g. Kubernetes) display both stdout and stderr by default when container logs 
are requested, Nomad distinguishes between the two streams. When viewing alloc 
logs in the Nomad GUI, click the stderr button. When viewing alloc logs using 
the Nomad CLI, use "nomad alloc logs -stderr".

<a name="map-file"></a>
### The node service fails to "load map file".

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
this field in [node.nomad](node.nomad) for guidance.
