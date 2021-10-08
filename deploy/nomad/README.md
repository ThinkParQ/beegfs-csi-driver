# BeeGFS CSI Driver

This directory includes a demo using the [BeeGFS CSI
Driver](https://github.com/NetApp/beegfs-csi-driver) to create
local "BeeGFS" volumes that can be mounted via the Nomad CSI
implementation.

## What Is This For?

At this time the BeeGFS CSI plugin is NOT SUPPORTED on Nomad, and is for demonstration and development purposes only. 
It shouldn't be used for production. If you want to get a quick idea of how BeeGFS CSI plugin works on 
Nomad in a single node environment, this demo is a good option. At a high-level, the BeeGFS CSI Plugin 
is for a single node deployment (for now), two volumes are created, volumes are mounted into a Redis workload,
status is shown, Redis workload is stopped, volumes are deleted, and the plugin is stopped. 
Overall, this shows the entire lifecycle of the plugin. You may run/modify "./run.sh" as is 
or run each command separately (recommended for debugging purposes). 

## Requirements

* Preinstall the beegfs-client-dkms (https://doc.beegfs.io/latest/advanced_topics/client_dkms.html)
and beegfs-util package on any Nomad node that will be used with BeeGFS.

* A running Nomad cluster with `docker.privileged.enabled = true`.

* Copy/Move the Nomad agent config file `nomad-agent.hcl` in this repo to the default path that Nomad 
recognizes of `/etc/nomad.d/nomad.hcl` (unless specified elsewhere).

Running the `run.sh` script in this directory will output the Nomad command
used to run the demo, as well as their outputs:

```
$ nomad job run ./plugin.nomad 
==> Monitoring evaluation "faa39692" 
Evaluation triggered by job "beegfs-csi-plugin" 
Allocation "8e2d591a" created: node "d71b39c2", group "csi" 
Evaluation status changed: "pending" -> "complete" 
==> Evaluation "faa39692" finished with status "complete" 
Nodes Healthy = 1 

$ nomad plugin status 
beegfs ID = beegfs-plugin0 
Provider = beegfs.csi.netapp.com 
Version = v1.2.0-0-gc65b537 
Controllers Healthy = 1 
Controllers Expected = 1 
Nodes Healthy = 1 
Nodes Expected = 1 

Allocations 
ID Node ID Task Group Version Desired Status Created Modified 
8e2d591a d71b39c2 csi 10 run running 4s ago 1s ago 

$ cat volume.hcl | sed | nomad volume create
$ sed -e "s/VOLUME_NAME/${VOLUME_BASE_NAME}[0]/" "${DEPLOY_DIR}/volume.hcl" | nomad volume create -
- Created external volume beegfs://scspa2058537001.rtp.openenglab.netapp.com/nomad/vol/test-volume%5B0%5D with ID test-volume[0] 

$ cat volume.hcl | sed | nomad volume create
$ sed -e "s/VOLUME_NAME/${VOLUME_BASE_NAME}[1]/" "${DEPLOY_DIR}/volume.hcl" | nomad volume create -
- Created external volume beegfs://scspa2058537001.rtp.openenglab.netapp.com/nomad/vol/test-volume%5B1%5D with ID test-volume[1] 

$ An example Nomad job that uses the volumes we created can be found in examples/nomad/redis.nomad
$ nomad job run ../../examples/nomad/redis.nomad 
==> Monitoring evaluation "e5a5118f" 
Evaluation triggered by job "example" 
Allocation "89cf02b6" created: node "d71b39c2", group "cache" 
Allocation "efe63fe4" created: node "d71b39c2", group "cache" 
==> Monitoring evaluation "e5a5118f" 
Evaluation within deployment: "08673535" 
Evaluation status changed: "pending" -> "complete" 
==> Evaluation "e5a5118f" finished with status "complete" 

$ nomad volume status 
Container Storage Interface ID Name Plugin ID Schedulable Access Mode 
test-volume[0] test-volume[0] beegfs-plugin0 true single-node-reader-only 
test-volume[1] test-volume[1] beegfs-plugin0 true single-node-reader-only 

$ nomad job stop example 
==> Monitoring evaluation "d6a99c92" 
Evaluation triggered by job "example" 
==> Monitoring evaluation "d6a99c92" 
Evaluation within deployment: "08673535" 
Evaluation status changed: "pending" -> "complete" 
==> Evaluation "d6a99c92" finished with status "complete" 

$ nomad volume delete test-volume[0] 

$ nomad volume delete test-volume[1] 

$ nomad job stop beegfs 
Are you sure you want to stop job "beegfs-csi-plugin"? [y/N] y 
==> Monitoring evaluation "1906534d" 
Evaluation triggered by job "beegfs-csi-plugin" 
==> Monitoring evaluation "1906534d" 
Evaluation status changed: "pending" -> "complete" 
==> Evaluation "1906534d" finished with status "complete"
```
