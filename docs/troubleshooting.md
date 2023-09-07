# Troubleshooting Guide <!-- omit in toc -->

<a name="contents"></a>
## Contents <!-- omit in toc -->

- [Overview](#overview)
- [Kubernetes](#kubernetes)
  - [Determining the BeeGFS Client Configuration for a PVC](#determining-the-beegfs-client-configuration-for-a-pvc)
  - [Orphaned Mounts Remain on Nodes](#orphaned-mounts-remain-on-nodes)
    - [General Symptoms](#general-symptoms)
    - [Outdated Driver or Node Unstage Timeout Exceeded](#outdated-driver-or-node-unstage-timeout-exceeded)
    - [Missing vol\_data.json](#missing-vol_datajson)
    - [Cleanup](#cleanup)
- [Access Denied Issues](#access-denied-issues)
  - [Discretionary Access Control](#discretionary-access-control)
  - [SELinux](#selinux)
- [Frequent Slow Operations and/or gRPC ABORTED Response Codes](#frequent-slow-operations-andor-grpc-aborted-response-codes)
- [Pod Stuck In Terminating after Subpath Deletion](#pod-stuck-in-terminating-after-subpath-deletion)

***

<a name="overview"></a>
## Overview

This section provides guidance and tips around troubleshooting issues that
come up using the driver. For anything not covered here, please [submit an
issue](https://github.com/ThinkParQ/beegfs-csi-driver/issues) using the label
"question". Suspected bugs should be submitted with the label "bug". 

***

<a name="kubernetes"></a>
## Kubernetes

<a name="k8s-determining-the-beegfs-client-conf-for-a-pvc"></a>
### Determining the BeeGFS Client Configuration for a PVC

BeeGFS Client configuration is specified in a Kubernetes ConfigMap, that is
parsed out to generate the client configuration that applies to a PVC for a
particular BeeGFS file system mounted to a particular Kubernetes node. How to
define this configuration is described in depth in the
[Deployment](deployment.md#general-configuration) documentation. You can
validate the deployed ConfigMap as follows: 

1. Run the following to confirm the name of the ConfigMap currently being used
   by the driver: 
```
-> kubectl get pod -n beegfs-csi csi-beegfs-controller-0 -o=custom-columns="CURRENT-BEEGFS-CONFIGMAP:spec.volumes[?(@.name=='config-dir')].configMap.name"
CURRENT-BEEGFS-CONFIGMAP
csi-beegfs-config-9bk9mkb5k7
```
2. Use the resulting ConfigMap name to describe the contents as follows:
```
-> kubectl describe ConfigMap -n beegfs-csi csi-beegfs-config-9bk9mkb5k7
Name:         csi-beegfs-config-9bk9mkb5k7
Namespace:    beegfs-csi
Labels:       <none>
```

In some cases administrators may wish to validate the final configuration the
driver parsed out for a particular PVC. For the following steps to work the PVC
must have been bound to a PV, and that PVC must be in use by a running pod.

1. Determine the name of the volume that corresponds with the PVC you want to
   investigate with `kubectl get pvc`. In this example it is `pvc-3ad5dffc`.
```
-> kubectl get pvc -A
NAMESPACE   NAME                 STATUS   VOLUME         CAPACITY   ACCESS MODES   STORAGECLASS        AGE
default     csi-beegfs-dyn-pvc   Bound    pvc-3ad5dffc   1Gi        RWX            csi-beegfs-dyn-sc   22h
```
2. Determine the node a Pod consuming the PVC is running on with `kubectl get
   pod -o wide`. In this example it is `ictm1625h12`.
```
-> kubectl get pod -o wide
NAME                 READY   STATUS    RESTARTS   AGE   IP               NODE          NOMINATED NODE   READINESS GATES
csi-beegfs-dyn-app   1/1     Running   0          22h   10.233.124.237   ictm1625h12   <none>           <none>
```
3. SSH to the node in question then run `mount | grep <PV_NAME>`, for example: 
```
-> mount | grep pvc-3ad5dffc
beegfs_nodev on /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-3ad5dffc/globalmount/mount type beegfs (rw,relatime,cfgFile=/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-3ad5dffc/globalmount/beegfs-client.conf)
beegfs_nodev on /var/lib/kubelet/pods/aa002809-c443-42f1-970c-e2e9c7ca14ad/volumes/kubernetes.io~csi/pvc-3ad5dffc/mount type beegfs (rw,relatime,cfgFile=/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-3ad5dffc/globalmount/beegfs-client.conf)
```
4. The file referenced by cfgFile (both mounts reference the same file) is the
   actual BeeGFS client file used to mount the PVC: 
```
-> sudo cat /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-3ad5dffc/globalmount/beegfs-client.conf | grep quotaEnabled
quotaEnabled                 = true
```

<a name="orphan-mounts"></a>
### Orphaned Mounts Remain on Nodes

There are a number of circumstances that can cause orphaned mounts to remain on
the nodes of a container orchestrator after a BeeGFS volume is deleted. These
are largely outside the control of the BeeGFS CSI driver and occur due to how a
particular container orchestrator interacts with CSI drivers in general.
Starting in v1.2.1 the BeeGFS CSI driver introduced functionality that
mitigates common causes of this behavior in Kubernetes, but administrators
should be aware of the potential, and the measures taken by the driver to
mitigate it.

<a name="orphan-mounts-general-symptoms"></a>
#### General Symptoms

There are BeeGFS mounts on a worker node that are no longer associated 
with existing Persistent Volumes.

```bash
# On a workstation with kubectl access to the cluster:
-> kubectl get pv
No resources found

# On a worker node:
-> mount | grep beegfs
tmpfs on /var/lib/kubelet/pods/d8acdcaf-38ab-46c1-ab46-bbec0ca67e0b/volumes/kubernetes.io~secret/csi-beegfs-node-sa-token-j6msh type tmpfs (rw,relatime,seclabel)
beegfs_nodev on /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-12ff9a7a/globalmount/mount type beegfs (rw,relatime,cfgFile=/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-12ff9a7a/globalmount/beegfs-client.conf)
beegfs_nodev on /var/lib/kubelet/pods/72d39e7f-3685-4f17-9dd1-bd9796d92b75/volumes/kubernetes.io~csi/pvc-12ff9a7a/mount type beegfs (rw,relatime,cfgFile=/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-12ff9a7a/globalmount/beegfs-client.conf)
beegfs_nodev on /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-1b4d5347/globalmount/mount type beegfs (rw,relatime,cfgFile=/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-1b4d5347/globalmount/beegfs-client.conf)
beegfs_nodev on /var/lib/kubelet/pods/92c15f5b-7c13-4f52-87a1-b4400603e990/volumes/kubernetes.io~csi/pvc-1b4d5347/mount type beegfs (rw,relatime,cfgFile=/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-1b4d5347/globalmount/beegfs-client.conf)
beegfs_nodev on /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-5c645794/globalmount/mount type beegfs (rw,relatime,cfgFile=/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-5c645794/globalmount/beegfs-client.conf)
beegfs_nodev on /var/lib/kubelet/pods/92c15f5b-7c13-4f52-87a1-b4400603e990/volumes/kubernetes.io~csi/pvc-5c645794/mount type beegfs (rw,relatime,cfgFile=/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-5c645794/globalmount/beegfs-client.conf)
beegfs_nodev on /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-af832ede/globalmount/mount type beegfs (rw,relatime,cfgFile=/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-af832ede/globalmount/beegfs-client.conf)
beegfs_nodev on /var/lib/kubelet/pods/c8cb9229-ea8d-4430-88c3-418281db59bf/volumes/kubernetes.io~csi/pvc-af832ede/mount type beegfs (rw,relatime,cfgFile=/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-af832ede/globalmount/beegfs-client.conf)
```

The kubelet on the worker node reports errors while attempting to clean up
associated directories in `/var/lib/kubelet/pods`. Specific error messages vary
depending on the root cause of the issue.

```bash
-> journalctl -xe -t hyperkube
Sep 27 10:49:44 openshift-beegfs-rhel-worker-1 hyperkube: I0927 10:49:44.056774    2120 reconciler.go:196] "operationExecutor.UnmountVolume started for volume \"test-volume\" (UniqueName: \"kubernetes.io/csi/beegfs.csi.netapp.com^beegfs://scspa2058537001.rtp.openenglab.netapp.com/e2e-test/dynamic/pvc-5e90c0c8\") pod \"22e1f3ae-81b4-45e2-8010-95ccc3a2e9ad\" (UID: \"22e1f3ae-81b4-45e2-8010-95ccc3a2e9ad\") "
Sep 27 10:49:44 openshift-beegfs-rhel-worker-1 hyperkube: W0927 10:49:44.168265    2120 mount_helper_common.go:34] Warning: Unmount skipped because path does not exist: /var/lib/kubelet/pods/22e1f3ae-81b4-45e2-8010-95ccc3a2e9ad/volume-subpaths/pvc-5e90c0c8/test-container-subpath-dynamicpv-ccs7/0
Sep 27 10:49:44 openshift-beegfs-rhel-worker-1 hyperkube: E0927 10:49:44.168436    2120 nestedpendingoperations.go:301] Operation for "{volumeName:kubernetes.io/csi/beegfs.csi.netapp.com^beegfs://scspa2058537001.rtp.openenglab.netapp.com/e2e-test/dynamic/pvc-5e90c0c8 podName:22e1f3ae-81b4-45e2-8010-95ccc3a2e9ad nodeName:}"
```

<a name="orphan-mounts-unstage-timeout-exceeded"></a>
#### Outdated Driver or Node Unstage Timeout Exceeded

Prior to driver version 1.2.1, the primary cause of orphan mounts was the
possibility for Kubernetes to call DeleteVolume on the controller service before
calling NodeUnpublishVolume on all relevant node services. In this scenario, 
if DeleteVolume succeeds, NodeUnpublishVolume becomes impossible, leaving orphan 
mounts. 

Though the Kubernetes maintainers have been working to ensure that this
disruption in the order of CSI operations cannot occur in future Kubernetes
versions, we chose to add code to mitigate its effects in driver version 1.2.1.
This code is enabled by setting --node-unstage-timeout to something other than
0 (the deployment manifests do this automatically). The --node-unstage-timeout
flag causes a
`volDirBasePath/.csi/volumes/volumeName/nodes` directory to be created on the
BeeGFS file system for every new volume. When a node mounts a volume, its name
is added to this directory, and when a node unmounts a volume, its names is
removed from this directory. The controller service refuses to delete a volume
until either this directory is empty or --node-unstage-timeout is exceeded.

On an older version of the driver or when the driver is deployed into Kubernetes
with --node-unstage-timeout=0, no attempt is made to wait when DeleteVolume is
called. The controller service does not log any issue, as it has no way of
knowing whether or not a BeeGFS volume is still staged on some node. This can
result in the [general symptoms](#orphan-mounts-general-symptoms)
described above and kubelet logs indicating the failure to clean up a
non-existent directory (DeleteVolume has already successfully deleted a BeeGFS
directory that is bind mounted into a Kubernetes container). Note that the
failing directory might be a primary mount directory
(e.g. `/var/lib/kubelet/pods/pod/volumes/...`) or a subpath directory (e.g.
`var/lib/kubelet/pods/pod/volume-subpaths/...`).

```bash
-> journalctl -xe -t hyperkube
Sep 27 10:49:44 openshift-beegfs-rhel-worker-1 hyperkube: I0927 10:49:44.056774    2120 reconciler.go:196] "operationExecutor.UnmountVolume started for volume \"test-volume\" (UniqueName: \"kubernetes.io/csi/beegfs.csi.netapp.com^beegfs://scspa2058537001.rtp.openenglab.netapp.com/e2e-test/dynamic/pvc-5e90c0c8\") pod \"22e1f3ae-81b4-45e2-8010-95ccc3a2e9ad\" (UID: \"22e1f3ae-81b4-45e2-8010-95ccc3a2e9ad\") "
Sep 27 10:49:44 openshift-beegfs-rhel-worker-1 hyperkube: W0927 10:49:44.168265    2120 mount_helper_common.go:34] Warning: Unmount skipped because path does not exist: /var/lib/kubelet/pods/22e1f3ae-81b4-45e2-8010-95ccc3a2e9ad/volume-subpaths/pvc-5e90c0c8/test-container-subpath-dynamicpv-ccs7/0
Sep 27 10:49:44 openshift-beegfs-rhel-worker-1 hyperkube: E0927 10:49:44.168436    2120 nestedpendingoperations.go:301] Operation for "{volumeName:kubernetes.io/csi/beegfs.csi.netapp.com^beegfs://scspa2058537001.rtp.openenglab.netapp.com/e2e-test/dynamic/pvc-5e90c0c8 podName:22e1f3ae-81b4-45e2-8010-95ccc3a2e9ad nodeName:}"
```

Follow the [cleanup](#orphan-mounts-cleanup) instructions to clean up.
Then upgrade the driver or set --node-unstage-timeout to ensure the issue
doesn't occur again.

Even the latest driver with a properly configured --node-unstage-timeout can
produce this issue in extreme circumstances. For example, it can occur if
Kubernetes calls DeleteVolume prematurely and some unforeseen issue on a node
delays unpublishing for the entirety of the timeout interval. In this case
DeleteVolume will successfully delete the BeeGFS directory, making further
NodeUnpublishVolume attempts impossible. Note that this is intentional behavior.
Without it, if a cluster node was permanently lost, a DeleteVolume call could
never succeed because the node would never have the opportunity to remove its
name from the BeeGFS `.csi` directory. Fortunately, the v1.2.1 changes make this
situation easy to identify.

```bash
-> kubectl logs csi-beegfs-controller-0
I1216 16:41:04.893012       1 server.go:192]  "msg"="GRPC call" "reqID"="0006" "method"="/csi.v1.Controller/DeleteVolume" "request"="{\"volume_id\":\"beegfs://10.113.4.46/k8s/name/dyn/pvc-15ba5493\"}"
I1216 16:41:04.893633       1 beegfs_util.go:62]  "msg"="Writing client files" "reqID"="0006" "path"="/var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_k8s_name_dyn_pvc-15ba5493" "volumeID"="beegfs://10.113.4.46/k8s/name/dyn/pvc-15ba5493"
I1216 16:41:04.894929       1 beegfs_util.go:208]  "msg"="Mounting volume to path" "reqID"="0006" "mountOptions"=["rw","relatime","cfgFile=/var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_k8s_name_dyn_pvc-15ba5493/beegfs-client.conf"] "path"="/var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_k8s_name_dyn_pvc-15ba5493/mount" "volumeID"="beegfs://10.113.4.46/k8s/name/dyn/pvc-15ba5493"
I1216 16:41:04.896100       1 mount_linux.go:146] Mounting cmd (mount) with arguments (-t beegfs -o rw,relatime,cfgFile=/var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_k8s_name_dyn_pvc-15ba5493/beegfs-client.conf beegfs_nodev /var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_k8s_name_dyn_pvc-15ba5493/mount)
I1216 16:41:06.557851       1 controllerserver.go:452]  "msg"="Waiting for volume to unstage from all nodes" "reqID"="0006" "secondsLeft"=59 "volumeID"="beegfs://10.113.4.46/k8s/name/dyn/pvc-15ba5493"
...
I1216 16:42:03.549724       1 controllerserver.go:452]  "msg"="Waiting for volume to unstage from all nodes" "reqID"="0006" "secondsLeft"=2 "volumeID"="beegfs://10.113.4.46/k8s/name/dyn/pvc-15ba5493"
I1216 16:42:06.131629       1 controllerserver.go:464]  "msg"="Volume did not unstage on all nodes; orphan mounts may remain" "reqID"="0006" "remainingNodes"=["some-node"] "volumeID"="beegfs://10.113.4.46/k8s/name/dyn/pvc-15ba5493"
I1216 16:42:06.131693       1 controllerserver.go:469]  "msg"="Deleting BeeGFS directory" "reqID"="0006" "path"="/k8s/name/dyn/.csi/volumes/pvc-15ba5493" "volumeID"="beegfs://10.113.4.46/k8s/name/dyn/pvc-15ba5493"
I1216 16:42:06.556387       1 controllerserver.go:482]  "msg"="Deleting BeeGFS directory" "reqID"="0006" "path"="/k8s/name/dyn/pvc-15ba5493" "volumeID"="beegfs://10.113.4.46/k8s/name/dyn/pvc-15ba5493"
I1216 16:42:07.039244       1 beegfs_util.go:270]  "msg"="Unmounting volume from path" "reqID"="0006" "path"="/var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_k8s_name_dyn_pvc-15ba5493/mount" "volumeID"="beegfs://10.113.4.46/k8s/name/dyn/pvc-15ba5493"
I1216 16:42:07.145542       1 mount_helper_common.go:71] "/var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_k8s_name_dyn_pvc-15ba5493/mount" is a mountpoint, unmounting
I1216 16:42:07.145638       1 mount_linux.go:238] Unmounting /var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_k8s_name_dyn_pvc-15ba5493/mount
I1216 16:42:08.977916       1 mount_helper_common.go:85] "/var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_k8s_name_dyn_pvc-15ba5493/mount" is unmounted, deleting the directory
I1216 16:42:08.978110       1 beegfs_util.go:283]  "msg"="Cleaning up path" "reqID"="0006" "path"="/var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_k8s_name_dyn_pvc-15ba5493" "volumeID"="beegfs://10.113.4.46/k8s/name/dyn/pvc-15ba5493"
I1216 16:42:08.978688       1 server.go:202]  "msg"="GRPC response" "reqID"="0006" "method"="/csi.v1.Controller/DeleteVolume" "response"="{}"
```

First, take steps to determine why --node-unstage-timeout was exceeded. Then, 
follow the [cleanup](#orphan-mounts-cleanup) instructions to clean up.

<a name="orphan-mounts-missing-vol-data"></a>
#### Missing vol_data.json

Orphan mounts can also occur in older versions of Kubernetes for a reason that
the driver cannot mitigate. Along with the general symptoms, the kubelet logs on
an affected node indicate that UnmountVolume is continuously failing due to a
missing vol_data.json file.

```bash
-> ssh root@some.node
-> journalctl -e -t kubelet --since "5 minutes ago"
...
Nov 05 16:15:11 kubernetes-119-cluster-8 kubelet[2478]: E1105 16:15:11.740637    2478 reconciler.go:193] operationExecutor.UnmountVolume failed (controllerAttachDetachEnabled true) for volume "volume1" (UniqueName: "kubernetes.io/csi/beegfs.csi.netapp.com^beegfs://10.113.4.46/e2e-test/dynamic/pvc-380e4ac6") pod "06f28bf6-b4a2-4d4e-8104-e2d60a0682b8" (UID: "06f28bf6-b4a2-4d4e-8104-e2d60a0682b8") : UnmountVolume.NewUnmounter failed for volume "volume1" (UniqueName: "kubernetes.io/csi/beegfs.csi.netapp.com^beegfs://10.113.4.46/e2e-test/dynamic/pvc-380e4ac6") pod "06f28bf6-b4a2-4d4e-8104-e2d60a0682b8" (UID: "06f28bf6-b4a2-4d4e-8104-e2d60a0682b8") : kubernetes.io/csi: unmounter failed to load volume data file [/var/lib/kubelet/pods/06f28bf6-b4a2-4d4e-8104-e2d60a0682b8/volumes/kubernetes.io~csi/pvc-380e4ac6/mount]: kubernetes.io/csi: failed to open volume data file [/var/lib/kubelet/pods/06f28bf6-b4a2-4d4e-8104-e2d60a0682b8/volumes/kubernetes.io~csi/pvc-380e4ac6/vol_data.json]: open /var/lib/kubelet/pods/06f28bf6-b4a2-4d4e-8104-e2d60a0682b8/volumes/kubernetes.io~csi/pvc-380e4ac6/vol_data.json: no such file or directory
```

This is a bug in Kubelet itself. See [Kubernetes Issue 
\#101911](https://github.com/kubernetes/kubernetes/issues/101911) and its 
associated fix in [Kubernetes PR 
\#102576](https://github.com/kubernetes/kubernetes/pull/102576) for details. 
When affected by this bug, Kubelet fails to call NodeUnstageVolume while 
tearing down a Pod and Kubernetes calls DeleteVolume anyway. Kubelet cannot 
recover without manual intervention on the node.

This bug is fixed in the following Kubernetes versions:
* 1.22.0
* 1.21.4
* 1.20.10
* 1.19.14

Follow the [cleanup](#orphan-mounts-cleanup) instructions to clean up.
Then upgrade Kubernetes to ensure the issue doesn't occur again.

<a name="orphan-mounts-cleanup"></a>
#### Cleanup

On each node with orphan mounts, identify and unmount them.

```bash
mount | grep beegfs
umount <mount point>
```

Kubelet may no longer be able to clean up the orphaned Pod directories 
associated with the mounts. Delete orphaned pod directories manually.

```bash
journalctl -xe -t kubelet
# Look for messages about orphaned Pods that can't be cleaned up.
rm -rf /var/lib/kubelet/pods/<orphaned pod>
```

To prevent a reoccurrence, identify the root cause (if it is listed above) and 
take the required steps.

***

<a name="access-denied-issues"></a>
## Access Denied Issues

The BeeGFS CSI driver itself runs in a privileged container for a number of reasons and does not generally see "access 
denied" type issues. However, arbitrary containers attempting to read, write, or execute files stored in a BeeGFS file 
system may fail to do so. Typically, this results in crash-looping containers or containers that log "permission denied" 
type messages.

<a name="discretionary-access-control"></a>
### Discretionary Access Control

The most obvious culprits are Discretionary Access Control (DAC) permissions. These are the typical 0644 or -rw-r--r--
type permissions most Linux users are familiar with. See the [permissions section of usage.md](usage.md/permissions) for
a comprehensive discussion on how to correct these issues.

<a name="selinux"></a>
### SELinux

If you are confident DAC permissions are not an issue and still seeing "permission denied" in your container logs,
consider whether SELinux is involved. SELinux is enabled by default on Fedora, RedHat Enterprise Linux, RedHat CoreOS,
and similar distributions. When SELinux is enabled, the BeeGFS CSI driver automatically mounts BeeGFS file systems with
the `-o context=system_u:object_r:container_file_t:s0` mount flag (see the [SELinux section in
deployment.md](deployment.md/selinux) for more information). However, this may not be enough in all environments or for
all use cases.

Administrators with SELinux experience can:
* Change the security context the BeeGFS CSI driver uses to mount BeeGFS file systems with the `mountOptions` field in a
  Kubernetes Storage Class, the `mountOptions` field in a Kubernetes Persistent Volume, or a similar field when using
  another container orchestator. For example, supply `context=system_u:object_r:nfs_t:s0` as one of a set of mount
  options.
* Modify the existing SELinux security policy or add an additional SELinux security policy that allows container
  processes to access files and directories with the BeeGFS CSI driver default security context.

If these options do not work, or if you are running a non-production system, putting SELinux into permissive mode will
also alleviate this issue. It is not recommended to take this action on a production system.

***

<a name="#frequent-slow-operations-grpc-aborted-response-codes"></a>
## Frequent Slow Operations and/or gRPC ABORTED Response Codes

The controller service must execute multiple beegfs-ctl commands to fulfill a CreateVolume request. In an optimal
networking environment, each command takes milliseconds to complete and CreateVolume returns quickly. However, each
command can take considerably longer when a BeeGFS service (mgmtd, metadata, or storage) is improperly configured. If
the total time for all commands is longer than the gRPC client is willing to wait, the client may issue a duplicate
CreateVolume request. The driver immediately responds to this new request with the ABORTED response code and the
message, "volumeID is in use by another request; check BeeGFS network configuration if this problem persists". The
driver continues to work to create the underlying BeeGFS directory in the background. If the client sends a CreateVolume
request after the directory is created, the driver returns the OK response code and the container orchestrator continues
on to schedule the volume to a node.

The most common recurring cause for this issue is a BeeGFS service using an inaccessible primary interface. For example,
a BeeGFS storage service might listen on eth1 with IP address 10.10.10.10 (or more inappropriately docker0 with some
private IP address). The BeeGFS storage service advertises that it wants all clients to attempt to communicate with it
on this interface. When a BeeGFS client can't route traffic to the interface, it eventually falls back and attempts to
communicate with a different interface. This fallback takes (a currently unconfigurable) five seconds, and it occurs for
each new beegfs-ctl command.

Potential solutions:
* Configure BeeGFS services to use a
  [connInterfacesFile](#https://doc.beegfs.io/latest/advanced_topics/network_configuration.html) with interfaces ordered
  so that an interface BeeGFS clients can route traffic to appears first.
* Add specific subnets to the driver's
  [connNetFilter](https://doc.beegfs.io/latest/advanced_topics/network_configuration.html) configuration so that it
  automatically avoids contacting the inaccessible interface. (This may be necessary in environments where only specific
  clients cannot access the advertised interface.)

***
<a name="pod-stuck-in-terminating-after-subpath-delete"></a>
## Pod Stuck In Terminating after Subpath Deletion

Pods utilizing the BeeGFS CSI driver are susceptible to an issue where the Pod
may become stuck in the terminating state if there are multiple containers in
the Pod, one of the containers in the Pod has mounted a subpath of a volume used
by the other container, and the subpath is deleted from the container with the
entire volume.

See [Kubernetes issue
97031](https://github.com/kubernetes/kubernetes/issues/97031) and its
associated [Pull Request
106906](https://github.com/kubernetes/kubernetes/pull/106906) for more details.

This issue is fixed in the following Kubernetes versions:
 * 1.23.2
 * 1.22.6
 * 1.21.9
