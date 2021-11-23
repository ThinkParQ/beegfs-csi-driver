# Troubleshooting Guide

## Contents
<a name="contents"></a>

* [Overview](#overview)
* [Kubernetes](#kubernetes)
  * [Determining the BeeGFS Client Configuration 
  for a PVC](#k8s-determining-the-beegfs-client-conf-for-a-pvc)
  * [Orphaned BeeGFS Mounts Remain on Nodes](#orphan-mounts)

## Overview
<a name="overview"></a>
This section provides guidance and tips around troubleshooting issues that
come up using the driver. For anything not covered here, please [submit an
issue](https://github.com/NetApp/beegfs-csi-driver/issues) using the label
"question". Suspected bugs should be submitted with the label "bug". 

## Kubernetes
<a name="kubernetes"></a>

### Determining the BeeGFS Client Configuration for a PVC
<a name="k8s-determining-the-beegfs-client-conf-for-a-pvc"></a>

BeeGFS Client configuration is specified in a Kubernetes ConfigMap, that is
parsed out to generate the client configuration that applies to a PVC for a
particular BeeGFS file system mounted to a particular Kubernetes node. How to
define this configuration is described in depth in the
[Deployment](deployment.md#general-configuration) documentation. You can
validate the deployed ConfigMap as follows: 

1. Run the following to confirm the name of the ConfigMap currently being used
   by the driver: 
```
joe-mac-0:Desktop joe$ kubectl get pod -n kube-system csi-beegfs-controller-0 -o=custom-columns="CURRENT-BEEGFS-CONFIGMAP:spec.volumes[?(@.name=='config-dir')].configMap.name"
CURRENT-BEEGFS-CONFIGMAP
csi-beegfs-config-9bk9mkb5k7
```
2. Use the resulting ConfigMap name to describe the contents as follows:
```
joe-mac-0:Desktop joe$ kubectl describe ConfigMap -n kube-system csi-beegfs-config-9bk9mkb5k7
Name:         csi-beegfs-config-9bk9mkb5k7
Namespace:    kube-system
Labels:       <none>
```

In some cases administrators may wish to validate the final configuration the
driver parsed out for a particular PVC. For the following steps to work the PVC
must have been bound to a PV, and that PVC must be in use by a running pod.

1. Determine the name of the volume that corresponds with the PVC you want to
   investigate with `kubectl get pvc`. In this example it is `pvc-3ad5dffc`.
```
joe-mac-0:Desktop joe$ kubectl get pvc -A
NAMESPACE   NAME                 STATUS   VOLUME         CAPACITY   ACCESS MODES   STORAGECLASS        AGE
default     csi-beegfs-dyn-pvc   Bound    pvc-3ad5dffc   1Gi        RWX            csi-beegfs-dyn-sc   22h
```
2. Determine the node a Pod consuming the PVC is running on with `kubectl get
   pod -o wide`. In this example it is `ictm1625h12`.
```
joe-mac-0:Desktop joe$ kubectl get pod -o wide
NAME                 READY   STATUS    RESTARTS   AGE   IP               NODE          NOMINATED NODE   READINESS GATES
csi-beegfs-dyn-app   1/1     Running   0          22h   10.233.124.237   ictm1625h12   <none>           <none>
```
3. SSH to the node in question then run `mount | grep <PV_NAME>`, for example: 
```
user@ictm1625h12:~$ mount | grep pvc-3ad5dffc
beegfs_nodev on /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-3ad5dffc/globalmount/mount type beegfs (rw,relatime,cfgFile=/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-3ad5dffc/globalmount/beegfs-client.conf)
beegfs_nodev on /var/lib/kubelet/pods/aa002809-c443-42f1-970c-e2e9c7ca14ad/volumes/kubernetes.io~csi/pvc-3ad5dffc/mount type beegfs (rw,relatime,cfgFile=/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-3ad5dffc/globalmount/beegfs-client.conf)
```
4. The file referenced by cfgFile (both mounts reference the same file) is the
   actual BeeGFS client file used to mount the PVC: 
```
user@ictm1625h12:~$ sudo cat /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-3ad5dffc/globalmount/beegfs-client.conf | grep quotaEnabled
quotaEnabled                 = true
```

### Orphaned Mounts Remain on Nodes
<a href="orphan-mounts"></a>

There are a number of circumstances that can cause orphaned mounts to remain on
the nodes of a container orchestrator after a BeeGFS volume is deleted. These
are largely outside the control of the BeeGFS CSI driver and occur due to how a
particular container orchestrator interacts with CSI drivers in general.
Starting in v1.2.1 the BeeGFS CSI driver introduced functionality that can
mitigate this behavior in most circumstances, but administrators should be aware
of the potential, and the measures taken by the driver to mitigate it.

#### General Symptoms

The driver controller service logs indicate that the controller service waited 
for the maximum allowed time before deleting a BeeGFS directory. This 
indicates that Kubernetes called DeleteVolume before NodeUnpublish and 
NodeUnstageVolume completed on at least one node and these operations never 
completed before the timeout.

```bash
-> kubectl logs csi-beegfs-controller-0 | grep 380e4ac6
...
I1105 17:54:00.421874       1 server.go:189]  "msg"="GRPC call" "reqID"="001d" "method"="/csi.v1.Controller/DeleteVolume" "request"="{\"volume_id\":\"beegfs://10.113.4.46/e2e-test/dynamic/pvc-380e4ac6\"}"
I1105 17:54:00.422079       1 beegfs_util.go:62]  "msg"="Writing client files" "reqID"="001d" "path"="/var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_e2e-test_dynamic_pvc-380e4ac6" "volumeID"="beegfs://10.113.4.46/e2e-test/dynamic/pvc-380e4ac6"
I1105 17:54:00.423025       1 beegfs_util.go:208]  "msg"="Mounting volume to path" "reqID"="001d" "mountOptions"=["rw","relatime","cfgFile=/var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_e2e-test_dynamic_pvc-380e4ac6/beegfs-client.conf"] "path"="/var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_e2e-test_dynamic_pvc-380e4ac6/mount" "volumeID"="beegfs://10.113.4.46/e2e-test/dynamic/pvc-380e4ac6"
I1105 17:54:00.445937       1 controllerserver.go:441]  "msg"="Waiting for volume to unstage from all nodes" "reqID"="001d" "volumeID"="beegfs://10.113.4.46/e2e-test/dynamic/pvc-380e4ac6"
I1105 17:54:02.450102       1 controllerserver.go:441]  "msg"="Waiting for volume to unstage from all nodes" "reqID"="001d" "volumeID"="beegfs://10.113.4.46/e2e-test/dynamic/pvc-380e4ac6"
...
I1105 17:54:58.559323       1 controllerserver.go:441]  "msg"="Waiting for volume to unstage from all nodes" "reqID"="001d" "volumeID"="beegfs://10.113.4.46/e2e-test/dynamic/pvc-380e4ac6"
I1105 17:55:00.562626       1 controllerserver.go:446]  "msg"="Deleting BeeGFS directory" "reqID"="001d" "path"="/e2e-test/dynamic/.csi/volumes/pvc-380e4ac6" "volumeID"="beegfs://10.113.4.46/e2e-test/dynamic/pvc-380e4ac6"
I1105 17:55:00.565337       1 controllerserver.go:461]  "msg"="Deleting BeeGFS directory" "reqID"="001d" "path"="/e2e-test/dynamic/pvc-380e4ac6" "volumeID"="beegfs://10.113.4.46/e2e-test/dynamic/pvc-380e4ac6"
I1105 17:55:00.570070       1 beegfs_util.go:270]  "msg"="Unmounting volume from path" "reqID"="001d" "path"="/var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_e2e-test_dynamic_pvc-380e4ac6/mount" "volumeID"="beegfs://10.113.4.46/e2e-test/dynamic/pvc-380e4ac6"
I1105 17:55:00.570734       1 mount_helper_common.go:71] "/var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_e2e-test_dynamic_pvc-380e4ac6/mount" is a mountpoint, unmounting
I1105 17:55:01.871999       1 mount_helper_common.go:85] "/var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_e2e-test_dynamic_pvc-380e4ac6/mount" is unmounted, deleting the directory
I1105 17:55:01.872095       1 beegfs_util.go:283]  "msg"="Cleaning up path" "reqID"="001d" "path"="/var/lib/kubelet/plugins/beegfs.csi.netapp.com/10.113.4.46_e2e-test_dynamic_pvc-380e4ac6" "volumeID"="beegfs://10.113.4.46/e2e-test/dynamic/pvc-380e4ac6"
...
```

BeeGFS mounts still exist on a node indefinitely.

```bash
-> ssh root@some.node mount | grep beegfs | grep pvc-380e4ac6
beegfs_nodev on /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-380e4ac6/globalmount/mount type beegfs (rw,relatime,cfgFile=/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-380e4ac6/globalmount/beegfs-client.conf)
beegfs_nodev on /var/lib/kubelet/pods/06f28bf6-b4a2-4d4e-8104-e2d60a0682b8/volumes/kubernetes.io~csi/pvc-380e4ac6/mount type beegfs (rw,relatime,cfgFile=/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-380e4ac6/globalmount/beegfs-client.conf)
```

#### Outdated Driver or --node-unstage-timeout=0

Code to mitigate the primary cause of orphan mounts was added in BeeGFS CSI
driver v1.2.1. This code is enabled by setting --node-unstage-timeout to
something other than 0 (the deployment manifests do this automatically). On an
older version of the driver or when the driver is deployed into Kubernetes with
--node-unstage-timeout=0, it is possible for Kubernetes to call DeleteVolume
before all nodes have called NodeUnpublishVolume. If the DeleteVolume succeeds,
NodeUnpublishVolume becomes impossible, leaving orphan mounts. Under these
circumstances, the controller service will NOT log anything about waiting for
the node service.

Follow the [cleanup](#orphan-mounts-cleanup) instructions to clean up.
Then upgrade the driver or set --node-unstage-timeout to ensure the issue
doesn't occur again.


#### Missing vol_data.json

Along with the general symptoms, the journal on an affected node indicates that 
UnmountVolume is continuously failing due to a missing vol_data.json file.

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

#### Cleanup
<a href="orphan-mounts-cleanup"></a>

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
