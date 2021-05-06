Using BeeGFS Quotas with the CSI Driver
---------------------------------------

## Overview 

To provide administrators visibility and control over file system utilization,
BeeGFS supports both [quota
tracking](https://doc.beegfs.io/latest/advanced_topics/quota.html#quota-tracking)
and [quota
enforcement](https://doc.beegfs.io/latest/advanced_topics/quota.html#quota-enforcement).
To ensure administrators have a way to understand capacity consumed by
containers running in orchestrators like Kubernetes and limit consumption where
needed, the BeeGFS CSI Driver provides full support for BeeGFS quotas. 

This document covers configuring the driver to enable any required quota
configuration for the BeeGFS clients (i.e., Kubernetes nodes). This document
will also demonstrate how to leverage quotas to track BeeGFS consumption with an
example that can be extended to any number of use cases.

## Prerequisites

* The BeeGFS server nodes (Management, Metadata, and Storage) must already have
  quota tracking and optionally quota enforcement enabled. See the [BeeGFS
  documentation](https://doc.beegfs.io/latest/advanced_topics/quota.html) if you
  need to enable quotas on a new or existing BeeGFS file system.
* BeeGFS quota enforcement is an [enterprise
  feature](https://www.beegfs.io/c/enterprise/enterprise-features/) requiring
  a professional support subscription with a [partner like
  NetApp](https://blog.netapp.com/solution-support-for-beegfs-and-e-series/) for 
  use in production.

## Enabling Quotas

In addition to the prerequisite steps for BeeGFS server nodes, to enable quotas
each client must set `quotaEnabled = true` in the configuration file
corresponding with the mount point for that file system. For container
orchestrator nodes including Kubernetes this is entirely handled by the BeeGFS
CSI driver alongside the rest of the [BeeGFS client
configuration](deployment.md#managing-beegfs-client-configuration). 

To enable quotas append `quotaEnabled: true` under `beegfsClientConf` in the
appropriate section of your BeeGFS CSI driver configuration file (for Kubernetes
this is at `deploy/prod/csi-beegfs-config.yaml`). The appropriate section
depends if you want it to apply to all BeeGFS file systems, specific file
systems, or specific clients. See [General
Configuration](deployment.md/#general-configuration) for full details. The
following example shows how to set an unrelated parameter (`connUseRDMA: true`)
on all file systems, then only enable quotas for the BeeGFS file system with a
management IP of `192.168.1.100`: 

```
config:
beegfsClientConf:
    connUseRDMA: true

fileSystemSpecificConfigs:
- sysMgmtdHost: "192.168.1.100"
    config:
    beegfsClientConf:
        quotaEnabled: true
```

When deploying to Kubernetes run `kubectl apply -k deploy/prod/` to deploy the
driver or update an existing deployment. Note the changes will only take effect
for new BeeGFS volumes, or when existing volumes are remounted, for example if a
pod moves between nodes. For any volumes that don't have this setting enabled,
all I/O will continue to affect the quota consumption of the root user, instead
of the actual caller.

## Tracking BeeGFS Consumption by Storage Class

### Introduction 

If your containers are running with a meaningful user ID from a quota
perspective, the above configuration is all that is needed to take advantage of
BeeGFS quotas. However in some environments the user in a container may vary,
creating a challenge to effectively track and limit space consumed by
containers. 

One option enabled by the BeeGFS CSI driver is tracking file system utilization
by Storage Class. In general administrators may find this (or a similar)
approach provides a more effective way of tracking file system utilization when
using BeeGFS with Kubernetes, instead of having to specify the Linux user/group
a particular user's containers must run as.

This option works by combining BeeGFS quotas with the driver's support for
[setting BeeGFS permissions](usage.md#permissions) using Storage Classes.
Considering BeeGFS volumes in Kubernetes are simply directories in BeeGFS, we
can use the `setgid` flag to ensure all files created in a specific volume are
associated with a particular group used to track file system consumption for
that volume. For dynamically provisioned volumes this is achieved by creating a
Storage Class that configures setgid permissions and a specific group ID on each
directory it creates as a BeeGFS volume. 

Note: For statically provisioned volumes the same behavior can be configured out
of band of the BeeGFS CSI driver, for example by running `chown <USER>:<GROUP>
<PATH_TO_STATIC_PVC> && chmod g+s <PATH_TO_STATIC_PVC>` on the directory used as
a static PVC. For more details see the BeeGFS documentation on setting up
[project directory quota
tracking](https://doc.beegfs.io/latest/advanced_topics/quota.html#project-directory-quota-tracking).

IMPORTANT: If users are allowed to run containers as root, there is nothing
preventing them from changing the `setgid` flag. See the Kubernetes
documentation on [Pod Security
Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/)
for ways to prevent users from running containers as root if desired. 

### Linux User/Group IDs and Containers

Typically BeeGFS
[recommends](https://doc.beegfs.io/latest/advanced_topics/quota.html#requirements-and-general-notes)
ensuring the local systems of all nodes are configured correctly to query passwd
and group databases. In other words, you typically want running `getent passwd`
and `getent group` on each node to return the same view of the user and group
databases for consistent mapping of user and group IDs to their textual
representations (i.e. user ID `1000` corresponds to user `linus` on all servers
and clients). 

In practical application containers generally won't have access to the data
source used by `getent` to determine this mapping, hence the BeeGFS CSI driver
requires specifying user and group IDs as an integer. As a result it is not
strictly necessary for all Kubernetes nodes to have a synchronized view of the
passwd and group databases. However, at minimum the BeeGFS management and
storage nodes along with any client nodes used for administrative purposes to
query/set BeeGFS quota information using `beegfs-ctl`, should have a
synchronized view of this mapping to avoid confusion. 

### Example Steps to setup a Storage Class that use setgid to set a specific group

Note: Any steps specific to creating groups or querying quota information should
take place on a node with a synchronized view of the user and group IDs as
described in the last section. 

* Create a Linux group to track utilization of the storage class (change the
  group ID and name if needed): `groupadd -g 1000 k8s-sc`
* Create a file `beegfs-dyn-sc.yaml` defining the storage class, notably
  specifying the appropriate group ID with `permissions/gid` and including the
  `setgid` flag by setting `2` at the beginning of `permissions/mode` before
  applying it with `kubectl apply -f beegfs-dyn-sc.yaml`:

```
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: csi-beegfs-dyn-sc
provisioner: beegfs.csi.netapp.com
parameters:
  sysMgmtdHost: 192.168.1.100
  volDirBasePath: k8s/
  permissions/gid: "1000"
  permissions/mode: "2755"
reclaimPolicy: Delete
volumeBindingMode: Immediate
allowVolumeExpansion: false
```
* To test out the storage class create a file `dyn-pvc.yaml` defining a
  persistent volume claim that references the storage class, and apply it with
  `kubectl apply -f dyn-pvc.yaml`: 
```
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: csi-beegfs-dyn-pvc
spec:
  accessModes:
  - ReadWriteMany
  resources:
    requests:
      storage: 1Gi
  storageClassName: csi-beegfs-dyn-sc
```
* To test the PVC create a file `dyn-app.yaml` defining a pod with an Alpine
  container that simply runs a command to create a file in the PVC. Apply it
  with `kubectl apply -f dyn-app.yaml`: 
```
kind: Pod
apiVersion: v1
metadata:
  name: csi-beegfs-dyn-app
spec:
  containers:
    - name: csi-beegfs-dyn-app
      image: alpine:latest
      volumeMounts:
      - mountPath: "/mnt/dyn"
        name: csi-beegfs-dyn-volume
      command: [ "ash", "-c", 'touch "/mnt/dyn/touched-by-${POD_UUID}" && sleep 7d']
      env:
        - name: POD_UUID
          valueFrom:
            fieldRef:
              fieldPath: metadata.uid
  volumes:
    - name: csi-beegfs-dyn-volume
      persistentVolumeClaim:
        claimName: csi-beegfs-dyn-pvc # defined in dyn-pvc.yaml
```

* On a client that has a synchronized view of the user and group databases we
  can verify things are working correctly: 
  * Use `ls -l <PATH_TO_PVC>` to verify the file was created with the
    appropriate group:
    ```
    [user@client01 ~]# ls -l /mnt/beegfs/k8s/pvc-db7a3ec0/
    total 0
    -rw-r--r-- 1 root k8s-sc 0 May  5 20:06 touched-by-33342327-1734-4bba-8b95-974aa8eccb3f
    ```
    * Run `beegfs-ctl --getquota --gid 1000` to verify BeeGFS sees the group: 
    ```
    [user@client01 ~]# beegfs-ctl --getquota --gid 1000 

    Quota information for storage pool Default (ID: 1):

          user/group     ||           size          ||    chunk files    
         name     |  id  ||    used    |    hard    ||  used   |  hard   
    --------------|------||------------|------------||---------|---------
            k8s-sc|  1000||      0 Byte|   unlimited||        0|unlimited
    ```
    Note: As verified in the `ls -l` output the file we created takes 0 bytes so
    this is expected.

* Optionally cleanup the resources by running `kubectl delete -f dyn-app.yaml &&
  kubectl delete -f dyn-pvc.yaml && kubectl delete -f beegfs-dyn-sc.yaml`