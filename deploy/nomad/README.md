# BeeGFS CSI Driver HashiCorp Nomad Deployment

## Contents

* [Overview](#overview)
* [Requirements](#requirements)
* [Steps](#steps)
  * [Deploy](#deploy)
  * [Test](#test)
  * [Clean Up](#clean-up)

***

<a name="overview"></a>
## Overview

At a high level, these manifests consist of two separate Nomad jobs. Together,
they get the BeeGFS CSI driver up and running in a Nomad cluster. Apply them
BEFORE any [example volumes or jobs](../../examples/nomad/README.md) that
require the BeeGFS CSI driver.
1. `controller.nomad` runs the CSI controller service as a "service" type job
   with a single replica.
1. `node.nomad` runs the CSI node service as a "system" type job so that a
   replica runs on every node in the cluster.

***

<a name="requirements"></a>
## Requirements

* An existing BeeGFS file system.
* An existing Nomad cluster running an [appropriate version of
  Nomad](../../docs/nomad.md#maturity-compatibility). On each client node:
    * Preinstall the
      [beegfs-client-dkms](https://doc.beegfs.io/latest/advanced_topics/client_dkms.html)
      and beegfs-util packages.
    * Enable the [Docker task
      driver](https://www.nomadproject.io/docs/drivers/docker#docker-driver).
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

<a name="steps"></a>
## Steps

<a name="deploy"></a>
### Deploy

If desired, manually verify image signatures by following the [documented
steps](../../docs/deployment.md#manual-image-verification).

`nomad job run test/nomad/beegfs-7.3-rh8/docker/controller.nomad`
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

`nomad job run test/nomad/beegfs-7.3-rh8/docker/node.nomad`
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

<a name="test"></a>
### Test

See the [Nomad usage examples](../../examples/nomad/README.md).

<a name="clean-up"></a>
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
