# BeeGFS CSI Driver HashiCorp Nomad Examples

While the BeeGFS CSI driver is primarily tested on (and intended for integration
with) Kubernetes, adherence to the Container Storage Interface (CSI) enables its
use with other container orchestrators as well. As a "simple and flexible
scheduler and orchestrator to deploy and manage containers and non-containerized
applications across on-prem and clouds at scale", [HashiCorp
Nomad](https://www.nomadproject.io/) is one such orchestrator.

## Contents

* [Important Warnings](#important-warnings)
* [What This Is For](#what-this-is-for)
* [Requirements](#requirements)
* [Steps](#steps)
  * [Deploy](#deploy)
  * [Test](#test)
  * [Clean Up](#clean-up)

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

At a high level, these manifests create a Nomad volume (backed by a BeeGFS CSI
driver volume) and a Nomad job that consumes it. Apply them AFTER the BeeGFS CSI
driver is [deployed](../../deploy/nomad/README.md) in a Nomad cluster.
1. `volume.hcl` creates a new Nomad volume backed by a BeeGFS CSI driver volume
   that is automatically provisioned by the BeeGFS CSI driver controller
   service.
1. `job.nomad` runs a simple Alpine container that writes the ID of its Nomad
   allocation to the underlying BeeGFS volume.

***

## Requirements
* An existing BeeGFS file system.
* An existing Nomad cluster running an [appropriate version of
  Nomad](#important-warnings) and the [BeeGFS CSI
  driver](../../deploy/nomad/README.md).
* Modify `sysMgmtdHost` and `volDirBasePath` in `volume.hcl` as appropriate for
  the existing file system.
* Modify any additional fields marked LIKELY TO REQUIRE MODIFICATION in
  `job.nomad`.

***

## Steps

### Deploy

`nomad plugin status --verbose`
```
Container Storage Interface
ID                 Provider               Controllers Healthy/Expected  Nodes Healthy/Expected
beegfs-csi-plugin  beegfs.csi.netapp.com  1/1                           2/2
```

`nomad volume create volume.hcl`
```
Created external volume beegfs://10.113.4.71/nomad/vol/beegfs-csi-volume with ID beegfs-csi-volume
```

`nomad job run test/nomad/beegfs-7.3-rh8/job.nomad`
```
==> 2022-07-25T16:11:20-05:00: Monitoring evaluation "a2b70577"
    2022-07-25T16:11:20-05:00: Evaluation triggered by job "beegfs-csi-job"
==> 2022-07-25T16:11:21-05:00: Monitoring evaluation "a2b70577"
    2022-07-25T16:11:21-05:00: Evaluation within deployment: "3f0d8e03"
    2022-07-25T16:11:21-05:00: Allocation "5d102b33" created: node "de35373e", group "beegfs-group"
    2022-07-25T16:11:21-05:00: Evaluation status changed: "pending" -> "complete"
==> 2022-07-25T16:11:21-05:00: Evaluation "a2b70577" finished with status "complete"
==> 2022-07-25T16:11:21-05:00: Monitoring deployment "3f0d8e03"
  ✓ Deployment "3f0d8e03" successful
    
    2022-07-25T16:11:34-05:00
    ID          = 3f0d8e03
    Job ID      = beegfs-csi-job
    Job Version = 0
    Status      = successful
    Description = Deployment completed successfully
    
    Deployed
    Task Group    Desired  Placed  Healthy  Unhealthy  Progress Deadline
    beegfs-group  1        1       1        0          2022-07-25T16:21:33-05:00
```

`nomad job status beegfs-csi-job`
```
ID            = beegfs-csi-job
Name          = beegfs-csi-job
Submit Date   = 2022-07-25T16:11:20-05:00
Type          = service
Priority      = 50
Datacenters   = dc1
Namespace     = default
Status        = running
Periodic      = false
Parameterized = false

Summary
Task Group    Queued  Starting  Running  Failed  Complete  Lost  Unknown
beegfs-group  0       0         1        0       0         0     0

Latest Deployment
ID          = 3f0d8e03
Status      = successful
Description = Deployment completed successfully

Deployed
Task Group    Desired  Placed  Healthy  Unhealthy  Progress Deadline
beegfs-group  1        1       1        0          2022-07-25T16:21:33-05:00

Allocations
ID        Node ID   Task Group    Version  Desired  Status   Created   Modified
5d102b33  de35373e  beegfs-group  0        run      running  1m2s ago  50s ago
```

### Test

At this point, it's probably interesting to poke around and try to understand whether Nomad is "actually" consuming a BeeGFS directory. Here are a couple of ideas.

The running job has written a file with its allocation ID to the expected internal location.

`nomad alloc exec 5d102b33 ls -l /mnt/beegfs-csi-volume`
```total 0
-rw-r--r--    1 root     root             0 Jul 25 21:11 touched-by-5d102b33-9312-36a8-4a5b-27f8fd78a9b4
```

The BeeGFS file system is mounted appropriately on the Nomad client node.

`ssh root@client2.esg-solutions-nomad mount | grep beegfs`
```
beegfs_nodev on /opt/nomad/client/csi/node/beegfs-csi-plugin/staging/beegfs-csi-volume/rw-file-system-multi-node-multi-writer/mount type beegfs (rw,relatime,context=system_u:object_r:container_file_t:s0,cfgFile=/opt/nomad/client/csi/node/beegfs-csi-plugin/staging/beegfs-csi-volume/rw-file-system-multi-node-multi-writer/beegfs-client.conf)
beegfs_nodev on /opt/nomad/client/csi/node/beegfs-csi-plugin/per-alloc/5d102b33-9312-36a8-4a5b-27f8fd78a9b4/beegfs-csi-volume/rw-file-system-multi-node-multi-writer type beegfs (rw,relatime,context=system_u:object_r:container_file_t:s0,cfgFile=/opt/nomad/client/csi/node/beegfs-csi-plugin/staging/beegfs-csi-volume/rw-file-system-multi-node-multi-writer/beegfs-client.conf)
tmpfs on /opt/nomad/alloc/5d102b33-9312-36a8-4a5b-27f8fd78a9b4/beegfs-task/secrets type tmpfs (rw,noexec,relatime,seclabel,size=1024k)
```

The written file can be found by traversing the mounted BeeGFS file system. (This is a lot easier if the BeeGFS file system also happens to be mounted to an accessible workstation and can be browsed from there.)

`ssh root@client2.esg-solutions-nomad ls -l /opt/nomad/client/csi/node/beegfs-csi-plugin/staging/beegfs-csi-volume rw-file-system-multi-node-multi-writer/mount/nomad/vol/beegfs-csi-volume`
```
total 0
-rw-r--r--. 1 root root 0 Jul 25 16:11 touched-by-5d102b33-9312-36a8-4a5b-27f8fd78a9b4
```

### Clean Up

`nomad job stop -purge beegfs-csi-job`
```
==> 2022-07-25T16:23:17-05:00: Monitoring evaluation "073a029f"
    2022-07-25T16:23:17-05:00: Evaluation triggered by job "beegfs-csi-job"
    2022-07-25T16:23:17-05:00: Evaluation within deployment: "3f0d8e03"
    2022-07-25T16:23:17-05:00: Evaluation status changed: "pending" -> "complete"
==> 2022-07-25T16:23:17-05:00: Evaluation "073a029f" finished with status "complete"
==> 2022-07-25T16:23:17-05:00: Monitoring deployment "3f0d8e03"
  ✓ Deployment "3f0d8e03" successful
    
    2022-07-25T16:23:17-05:00
    ID          = 3f0d8e03
    Job ID      = beegfs-csi-job
    Job Version = 0
    Status      = successful
    Description = Deployment completed successfully
    
    Deployed
    Task Group    Desired  Placed  Healthy  Unhealthy  Progress Deadline
    beegfs-group  1        1       1        0          2022-07-25T16:21:33-05:00
```

`nomad volume delete beegfs-csi-volume`
```
Successfully deleted volume "beegfs-csi-volume"!
```
