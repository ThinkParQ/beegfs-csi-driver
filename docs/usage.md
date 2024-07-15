# BeeGFS CSI Driver Usage <!-- omit in toc -->

<a name="contents"></a>
## Contents <!-- omit in toc -->

- [Important Concepts](#important-concepts)
  - [Definition of a "Volume"](#definition-of-a-volume)
  - [Capacity](#capacity)
  - [Static vs Dynamic Provisioning](#static-vs-dynamic-provisioning)
    - [Dynamic Provisioning Use Case](#dynamic-provisioning-use-case)
    - [Static Provisioning Use Case](#static-provisioning-use-case)
  - [Client Configuration and Tuning](#client-configuration-and-tuning)
- [Dynamic Provisioning Workflow](#dynamic-provisioning-workflow)
  - [Assumptions](#assumptions)
  - [High Level](#high-level)
  - [Create a Storage Class](#create-a-storage-class)
  - [Create a Persistent Volume Claim](#create-a-persistent-volume-claim)
  - [Create a Pod, Deployment, Stateful Set, etc.](#create-a-pod-deployment-stateful-set-etc)
- [Static Provisioning Workflow](#static-provisioning-workflow)
  - [Assumptions](#assumptions-1)
  - [High Level](#high-level-1)
  - [Create a Persistent Volume](#create-a-persistent-volume)
  - [Create a Persistent Volume Claim](#create-a-persistent-volume-claim-1)
  - [Create a Pod, Deployment, Stateful Set, etc.](#create-a-pod-deployment-stateful-set-etc-1)
- [Best Practices](#best-practices)
- [Managing ReadOnly Volumes](#managing-readonly-volumes)
  - [Configuring ReadOnly Volumes Within a Pod Specification](#configuring-readonly-volumes-within-a-pod-specification)
    - [Using The Container VolumeMount Method](#using-the-container-volumemount-method)
    - [Using the Volumes PersistentVolumeClaim Method](#using-the-volumes-persistentvolumeclaim-method)
    - [Additional Considerations](#additional-considerations)
  - [Configuring ReadOnly Volumes With MountOptions](#configuring-readonly-volumes-with-mountoptions)
    - [Configuring ReadOnly on a PersistentVolume](#configuring-readonly-on-a-persistentvolume)
    - [Configuring ReadOnly on a StorageClass](#configuring-readonly-on-a-storageclass)
- [Notes for BeeGFS Administrators](#notes-for-beegfs-administrators)
  - [General](#general)
  - [BeeGFS Mount Options](#beegfs-mount-options)
  - [Memory Consumption with RDMA](#memory-consumption-with-rdma)
  - [Permissions](#permissions)
    - [fsGroup Behavior](#fsgroup-behavior)
- [Limitations and Known Issues](#limitations-and-known-issues)
  - [General](#general-1)
  - [Read Only and Access Modes in Kubernetes](#read-only-and-access-modes-in-kubernetes)
  - [Long paths may cause errors](#long-paths-may-cause-errors)

***

<a name="important-concepts"></a>
## Important Concepts

<a name="definition-of-a-volume"></a>
### Definition of a "Volume"

Within the context of this driver, a "volume" is simply a directory within a
BeeGFS filesystem. When a volume is mounted by a Kubernetes Pod, only files
within this directory and its children are accessible by the Pod. An entire
BeeGFS filesystem can be a volume (e.g. by specifying */* as the */path/to/dir*
in the static provisioning workflow) or a single subdirectory many levels deep
can be a volume (e.g. by specifying */a/very/deep/directory* as the
*volDirBasePath* in the dynamic provisioning workflow).

<a name="capacity"></a>
### Capacity

In this version, the driver ignores the capacity requested for a Kubernetes
Persistent Volume. Consider the definition of a "volume" above. While an entire
BeeGFS filesystem may have a usable capacity of 100GiB, there is very little
meaning associated with the "usable capacity" of a directory within a BeeGFS (or
any POSIX) filesystem. The driver does provide integration with BeeGFS permissions 
and quotas which provides ways to limit the capacity consumed by containers. For
more details refer to the documentation on [Quotas](quotas.md).

Starting with v1.7.0 the driver also supports [volume
expansion](https://kubernetes-csi.github.io/docs/volume-expansion.html), which is useful for
instances where the persistent volume claim request size has meaning for the application. As with
the initial capacity request, the size of the PVC and PV are simply updated in the Kubernetes API to
reflect the requested new capacity, and there are no checks there is actually sufficient space
available to satisfy the requested capacity.

<a name="static-vs-dynamic-provisioning"></a>
### Static vs Dynamic Provisioning

<a name="dynamic-provisioning-use-case"></a>
#### Dynamic Provisioning Use Case

As a user, I want a volume to use as high-performance scratch space or
semi-temporary storage for my workload. I want the volume to be empty when my
workload starts. I may keep my volume around for other stages in my data
pipeline, or I may provide access to other users or workloads. Eventually, I'll
no longer need the volume and I expect it to clean up automatically.

In the Kubernetes dynamic provisioning workflow, an administrator identifies an
existing parent directory within a BeeGFS filesystem. When a user creates a PVC,
the driver automatically creates a new subdirectory underneath that parent
directory and binds it to the PVC. To the user and/or workload, the subdirectory
is the entire volume. It exists as long as the PVC exists.

<a name="static-provisioning-use-case"></a>
#### Static Provisioning Use Case

As an administrator, I want to make a directory within an existing BeeGFS file
system available to be mounted by multiple users and/or workloads. This
directory probably contains a large, commonly used dataset that I don't want to
see copied to multiple locations within my file system. I plan to manage the
volume's lifecycle and I don't want it cleaned up automatically.

As a user, I want to consume an existing dataset in my workload.

In the Kubernetes static provisioning workflow, an administrator manually
creates a PV and PVC representing an existing BeeGFS file system directory.
Multiple users and/or workloads can mount that PVC and consume the data the
directory contains.

<a name="client-configuration-and-tuning"></a>
### Client Configuration and Tuning

Depending on your topology, different nodes within your cluster or different
BeeGFS file systems accessible by your cluster may need different client
configuration parameters. This configuration is NOT handled at the volume level
(e.g. in a Kubernetes Storage Class or Kubernetes Persistent Volume). See
Managing BeeGFS Client Configuration in the [deployment guide](deployment.md)
for detailed instructions on how to prepare your cluster to mount various BeeGFS
file systems.

***

<a name="dynamic-provisioning-workflow"></a>
## Dynamic Provisioning Workflow

### Assumptions

1. A BeeGFS filesystem with its management service listening at `sysMgmtdHost`
   already exists and is accessible from all Kubernetes worker nodes.
1. A directory that can serve as the parent to all dynamically allocated
   subdirectories already exists within the BeeGFS filesystem at
   */path/to/parent/dir* OR it is fine for the driver to create one at
   */path/to/parent/dir*.

### High Level

1. An administrator creates a Kubernetes Storage Class describing a particular
   directory on a particular BeeGFS filesystem under which dynamically
   provisioned subdirectories should be created.
1. A user creates a Kubernetes Persistent Volume Claim requesting access to a
   newly provisioned subdirectory.
1. A user creates a Kubernetes Pod, Deployment, Stateful Set, etc. that
   references the Persistent Volume Claim.

Under the hood, the driver creates a new BeeGFS subdirectory. This subdirectory
is tied to a new Kubernetes Persistent Volume, which is bound to the
user-created Kubernetes Persistent Volume Claim. When a Pod is scheduled to a
Node, the driver uses information supplied by the Persistent Volume to mount the
subdirectory into the Pod's namespace.

### Create a Storage Class

Who: A Kubernetes administrator working closely with a BeeGFS administrator

Specify the filesystem and parent directory using the `sysMgmtdHost` and
`volDirBasePath` parameters respectively.

Striping parameters that can be specified using the beegfs-ctl command line
utility in the `--setpattern` mode can be passed with the prefix
`stripePattern/` in the `parameters` map. If no striping parameters are passed,
the newly created subdirectory has the same striping configuration as its
parent. The following `stripePattern/` parameters work with the driver:

| Prefix         | Parameter     | Required | Accepted patterns                       | Example    | Default             |
| -------------- | ------------- | -------- | --------------------------------------- | ---------- | ------------------- |
| stripePattern/ | storagePoolID | no       | unsigned integer                        | 1          | file system default |
| stripePattern/ | chunkSize     | no       | unsigned integer + k (kilo) or m (mega) | 512k<br>1m | file system default |
| stripePattern/ | numTargets    | no       | unsigned integer                        | 4          | file system default |

NOTE: While the driver expects values with certain patterns (e.g. unsigned
integer), Kubernetes only accepts string values in Storage Classes. These
values must be quoted in the Storage Class .yaml (as in the example below).

NOTE: The effects of unlisted configuration options are NOT tested with the
driver. Contact your BeeGFS support representative for recommendations on
appropriate settings. See the [BeeGFS documentation on
striping](https://doc.beegfs.io/latest/advanced_topics/striping.html) for
additional details.

By default, the driver creates all new subdirectories with root:root ownership
and globally read/write/executable 0777 access permissions. This makes it easy
for an arbitrary Pod to consume a dynamically provisioned volume. However,
administrators may want to [change the default permissions](#permissions) on a 
per-Storage-Class basis, in particular if [integration with BeeGFS quotas is desired](quotas.md). 
The following `permissions/` parameters allow this fine-grained control:

| Prefix       | Parameter | Required | Accepted patterns                  | Example     | Default  |
| ------------ | --------- | -------- | ---------------------------------- | ----------- | -------- |
| permissions/ | uid       | no       | unsigned integer                   | 1000        | 0 (root) |
| permissions/ | gid       | no       | unsigned integer                   | 1000        | 0 (root) |
| permissions/ | mode      | no       | three or four digit octal notation | 755<br>0755 | 0777     |

NOTE: While the driver expects values with certain patterns (e.g. unsigned 
integer), Kubernetes only accepts string values in Storage Classes. These 
values must be quoted in the Storage Class .yaml (as in the example below).

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: my-storage-class
provisioner: beegfs.csi.netapp.com
parameters:
  # All Storage Class values must be strings. Quotes are required on integers.
  sysMgmtdHost: 10.113.72.217
  volDirBasePath: /path/to/parent/dir 
  stripePattern/storagePoolID: "1"
  stripePattern/chunkSize: 512k
  stripePattern/numTargets: "4"
  permissions/uid: "1000"
  permissions/gid: "1000"
  permissions/mode: "0644"
reclaimPolicy: Delete
volumeBindingMode: Immediate
allowVolumeExpansion: false
```

### Create a Persistent Volume Claim

Who: A Kubernetes user

Specify the Kubernetes Storage Class using the `storageClassName` field in the
Kubernetes Persistent Volume Claim `spec` block.

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-pvc
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests: 
      storage: 100Gi
  storageClassName: my-storage-class
```

### Create a Pod, Deployment, Stateful Set, etc.

Who: A Kubernetes user

Follow standard Kubernetes practices to deploy a Pod that consumes the newly
created Kubernetes Persistent Volume Claim.

***

<a name="static-provisioning-workflow"></a>
## Static Provisioning Workflow

### Assumptions

1. A BeeGFS filesystem with its management service listening at `sysMgmtdHost`
   already exists and is accessible from all Kubernetes worker nodes.
1. A directory of interest already exists within the BeeGFS filesystem at
   */path/to/dir*. If this whole BeeGFS filesystem is to be consumed,
   */path/to/dir* is */*.

### High Level

1. An administrator creates a Kubernetes Persistent Volume referencing a
   particular directory on a particular BeeGFS filesystem.
1. An administrator or a user creates a Kubernetes Persistent Volume Claim that
   binds to the Persistent Volume.
1. A user creates a Kubernetes Pod, Deployment, Stateful Set, etc. that
   references the Persistent Volume Claim.

When a Pod is scheduled to a Node, the driver uses information supplied by the
Persistent Volume to mount the subdirectory into the Pod's namespace.

### Create a Persistent Volume

Who: A Kubernetes administrator working closely with a BeeGFS administrator

The driver receives all the information it requires to mount the directory of
interest into a Pod from the `volumeHandle` field in the `csi` block of the
Persistent Volume `spec` block. It MUST be formatted as modeled in the example.

NOTE: The driver does NOT provide a way to modify the stripe settings of a
directory in the static provisioning workflow.

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: my-pv
spec:
  accessModes:
    - ReadWriteMany
  capaciy:
    storage: 100Gi
  persistentVolumeReclaimPolicy: Retain
  csi:
    driver: beegfs.csi.netapp.com
    volumeHandle: beegfs://sysMgmtdHost/path/to/dir
```

### Create a Persistent Volume Claim

Who: A Kubernetes administrator or user

Each Persistent Volume Claim participates in a 1:1 mapping with a Persistent
Volume. Create a Persistent Volume Claim and set the `volumeName` field to
ensure it maps to the correct Persistent Volume.

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-pvc
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests: 
      storage: 100Gi
  storageClassName: ""
  volumeName: my-pv
```

### Create a Pod, Deployment, Stateful Set, etc.

Who: A Kubernetes user

Follow standard Kubernetes practices to deploy a Pod that consumes the newly
created Kubernetes Persistent Volume Claim.

***

<a name="best-practices"></a>
## Best Practices

* While multiple Kubernetes clusters can use the same BeeGFS file system, it is
  not recommended to have more than one cluster use the same `volDirBasePath`
  within the same file system.
* Do not rely on Kubernetes [access
  modes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes)
  to prevent directory contents from being overwritten. Instead set sensible
  permissions, especially on static directories containing shared datasets (more
  details [below](#read-only-and-access-modes-in-kubernetes)). 

***

<a name="managing-readonly-volumes"></a>
## Managing ReadOnly Volumes

In some cases an administrator or a user may wish to have a BeeGFS volume
mounted in ReadOnly mode within a container. There are several mechanisms to
accomplish this goal. The following describes some of the approaches to
configuring ReadOnly volumes and relevant information on how to choose the right
approach for your situation.

### Configuring ReadOnly Volumes Within a Pod Specification

Who: A kubernetes user or administrator

Within a Pod specification there are two options that currently apply to the
BeeGFS CSI driver for configuring a volume to be mounted as ReadOnly.
* There is a readOnly attribute of the [VolumeMounts for a
  container](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#volumes-1).
* There is a readOnly attribute of the [PersistentVolumeClaim source in the
  Volume configuration of a
  Pod](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/volume/#exposed-persistent-volumes).

#### Using The Container VolumeMount Method

This method can be used for any volume type, including BeeGFS volumes configured
as PersistentVolumeClaims. For other volume types that aren't configured as
PersistentVolumeClaims this might be the only option to specify the ReadOnly
mode in the Pod configuration. The following is an example of a pod
specification that uses this method to set the readOnly attribute.

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: pod-sample
spec:
  containers:
    - name: sample
      volumeMounts:
        - mountPath: /mnt/static
          name: csi-beegfs-static-volume
          readOnly: true
  volumes:
    - name: csi-beegfs-static-volume
      persistentVolumeClaim:
        claimName: beegfs-pvc-1
```

In this scenario the volume is staged with ReadWrite permissions and the
ReadOnly permission is applied in a subsequent bind mount specific to the
targeted container. Therefore the scope of this ReadOnly configuration is the
single container within the Pod.

#### Using the Volumes PersistentVolumeClaim Method

This method is available to any volume being presented to a pod as a
PersistentVolumeClaim. The following is an example of a pod specification that
uses this method to set the readOnly attribute.

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: pod-sample
spec:
  containers:
    - name: sample
      volumeMounts:
        - mountPath: /mnt/static
          name: csi-beegfs-static-volume
  volumes:
    - name: csi-beegfs-static-volume
      persistentVolumeClaim:
        claimName: beegfs-pvc-1
        readOnly: true
```

In this scenario the volume is staged with ReadWrite permissions and the
ReadOnly permission is applied in a subsequent bind mount where the volume is
made available to the Pod. Therefore the scope of this ReadOnly configuration is
all containers within the Pod.

#### Additional Considerations

You may not want to use one of the Pod configuration methods for configuring the
ReadOnly volume under these circumstances.
* The targeted volume may be used by multiple pods and you may not control the
  Pod configuration for all Pods that use the volume.
* You are an administrator and you want to control the ReadOnly attributes for a
  volume instead of letting users managing the Pods control the access for that
  volume.

### Configuring ReadOnly Volumes With MountOptions

Who: A Kubernetes administrator working closely with a BeeGFS administrator

The ReadOnly attribute for a volume can also be configured through the
mountOptions property of a persistent volume or a storage class object.

#### Configuring ReadOnly on a PersistentVolume

When defining the [PersistentVolume
spec](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/persistent-volume-v1/#PersistentVolumeSpec)
you can use the mountOptions property to define the mount options to use for
that particular volume. This can include configuring the volume to be ReadOnly
with the `ro` mount option. The following is an example.

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-sample
spec:
  accessModes:
    - ReadOnlyMany
  capacity:
    storage: 100Gi
  mountOptions:
    - ro
    - nosuid
    - relatime
  persistentVolumeReclaimPolicy: Retain
  storageClassName: "csi-beegfs-dyn-sc"
  csi:
    driver: beegfs.csi.netapp.com
    # Replace "localhost" with the IP address or hostname of the BeeGFS management daemon.
    # "all" k8s clusters may share access to statically provisioned volumes.
    # Ensure that the directory, e.g. "k8s/all/static", exists on BeeGFS.  The driver will not create the directory.
    volumeHandle: beegfs://localhost/k8s/all/static
```

In this scenario the volume will be staged as ReadOnly and all bind mounts will
be ReadOnly. Also, all uses of the volume will be ReadOnly regardless of the Pod
configuration for any Pod using this volume.

NOTE: If you specify any mount options with the mountOptions property, then you
need to specify all of your desired mount options here. See the [BeeGFS Mount
Options](#beegfs-mount-options) section for information on the default mount
options used.

#### Configuring ReadOnly on a StorageClass

A StorageClass object can be configured with a mountOptions parameter similar to
how a PersistentVolume object can be configured with mountOptions. However, the
mountOptions for a StorageClass are only applied to dynamically provisioned
volumes. Any mountOptions configured on a StorageClass do not apply to a
statically defined PersistentVolume that references that StorageClass object.

***

<a name="notes-for-beegfs-administrators"></a>
## Notes for BeeGFS Administrators

### General

* By default the driver uses the beegfs-client.conf file at
  */etc/beegfs/beegfs-client.conf* for base configuration. Modifying the
  location of this file is not currently supported without changing
  kustomization files. 
* When using dynamic provisioning, if `--node-unstage-timeout` is set to a nonzero value
  (default: 60) the driver will create a directory structure at `volDirBasePath/.csi/` 
  (in the BeeGFS filesystem) and use it to persist any state used by the driver, 
  [for example to prevent orphaned mounts](troubleshooting.md#orphan-mounts). 
  This behavior can optionally be disabled, but is strongly recommended for the driver 
  to function optimally.


<a name="beegfs-mount-options"></a>
### BeeGFS Mount Options

Except for `cfgFile` (which has to be set by the driver) mount options supported
by BeeGFS can be specified on a [persistent
volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#mount-options)
or [storage class](https://kubernetes.io/docs/concepts/storage/storage-classes/#mount-options).
Please note the driver DOES NOT validate provided mount options and use of
options not supported by BeeGFS may cause unpredictable behavior. 

By default the driver mounts BeeGFS with the following mount options: 
rw, relatime, and nosuid.

* The cfgFile option is also used, but it is handled entirely by the driver and
  ignored if specified.
* The nosuid mount option is used to adhere to [BeeGFS security
  recommendations](https://www.beegfs.io/c/the-importance-of-using-connauthfile-in-beegfs/).

### Memory Consumption with RDMA
For performance (and other) reasons each Persistent Volume used on a given
Kubernetes node has a separate mount point. When using remote direct memory
access (RDMA) this will increase the amount of memory used for RDMA queue pairs
between BeeGFS clients (K8s nodes) and BeeGFS servers. As of BeeGFS 7.2 this is
around 12-13MB per mount for each client connection to a BeeGFS storage/metadata
service. 

Since clients only open connections when needed this is unlikely to be an issue,
but in some large environments may result in unexpected memory utilization. This
is much more likely to be an issue on BeeGFS storage and metadata servers than
the Kubernetes nodes themselves (since multiple clients connect to each server).
Administrators are advised to spec out BeeGFS servers accordingly.

<a name="permissions"></a>
### Permissions

Note: See the section on [Creating a Storage Class](#create-a-storage-class) for
how to set permissions using the CSI driver.

By default, the driver creates all new subdirectories with root:root ownership
and globally read/write/executable 0777 access permissions. This works well in a
Kubernetes environment where Pods may run as an arbitrary user or group but
still expect to access provisioned volumes.

NOTE: Permissions on the `volDirBasePath` are not modified by the 
driver. These permissions can be used to limit external access to dynamically 
provisioned subdirectories even when these subdirectories themselves have 0777 
access permissions.
   
In certain situations, it makes sense to override the default behavior and 
instruct the driver to create directories owned by some other user/group or 
with a different mode. This can be done on a per-Storage-Class basis. Some 
example scenarios include:
* [BeeGFS quotas](https://doc.beegfs.io/latest/advanced_topics/quota.html) are 
  in use and all files and directories in provisioned volumes must be 
  associated with a single appropriate GID (as in the 
  [project directory quota tracking](https://doc.beegfs.io/latest/advanced_topics/quota.html#project-directory-quota-tracking))
  example in the BeeGFS documentation.
  * See the driver documentation on [Quotas](quotas.md) for further guidance 
  on how to use BeeGFS quotas with the CSI driver.
* It is important to limit the ability of arbitrary BeeGFS file system users 
  to access dynamically provisioned volumes and the volumes will be accessed 
  by Pods running as a known user or group anyway (see the above note for 
  an alternate potential mitigation).

NOTE: The above BeeGFS quotas documentation suggests using `chmod g+s` on a
directory to enable the setgid bit. The exact same behavior can be obtained
using four digit octal permissions in the `parameters.permissions/mode` field of
a BeeGFS Storage Class. For example, 2755 represents the common 755 directory
access mode with setgid enabled. See the 
[chmod man page](https://linux.die.net/man/1/chmod) for more details.

Under the hood, the driver uses a combination of beegfs-ctl and 
chown/chmod-like functionality to set the owner, group, and access mode of a 
new BeeGFS subdirectory. These properties limit access to the subdirectory both 
outside of (as expected) and inside of Kubernetes. If permissions are set in a 
Storage Class, Kubernetes Pods likely need to specify one of the following 
parameters to allow access:
* spec.securityContext.runAsUser
* spec.securityContext.runAsGroup
* spec.securityContext.fsGroup
* spec.container.securityContext.runAsUser
* spec.container.securityContext.runAsGroup

<a name="fsgroup-behavior"></a>
#### fsGroup Behavior

Some CSI drivers support a recursive operation in which the permissions and
ownership of all files and directories in a provisioned volume are changed to
match the fsGroup parameter of a Security Context on Pod startup. This 
behavior is generally undesirable with BeeGFS for the following reasons:
* Unexpected permissions changes within a BeeGFS file system may be
  confusing to administrators and detrimental to security (especially in the
  static provisioning workflow).
* Competing operations executed by multiple Pods against large file systems
  may be time-consuming and affect overall system performance.

For clusters running most versions, Kubernetes heuristics enable this behavior 
on ReadWriteOnce volumes and do NOT enable this behavior on ReadWriteMany 
volumes. Create only ReadWriteMany volumes to ensure no unexpected permissions 
updates occur.

For clusters running v1.20 or v1.21 WITH the optional CSIVolumeFSGroupPolicy 
feature gate (in an eventual future version the feature gate will not be 
required), the `csiDriver.spec.fsGroupPolicy` parameter can be used to disable 
this behavior for all volumes. The beegfs-csi-driver deploys with this parameter 
set to "None" in case it is deployed to a cluster that supports it.

***

<a name="limitations-and-known-issues"></a>
## Limitations and Known Issues

### General 

* Each BeeGFS instance used with the driver must have a unique BeeGFS management
  IP address.

### Read Only and Access Modes in Kubernetes

Access modes in Kubernetes are how a driver understands what K8s wants to do
with a volume, but do not strictly enforce behavior. This may result in
unexpected behavior if administrators expect creating a Persistent Volume with
(for example) `ReadOnlyMany` access will enforce read only access across all
nodes accessing the volume. This is a larger issue with Kubernetes/CSI ecosystem
and not specific to the BeeGFS driver. Some relevant discussion can be found in
this [GitHub issue](https://github.com/kubernetes/kubernetes/issues/70505).

If the `pod.spec.volumes.persistentVolumeClaim.readOnly` flag or the
`pod.spec.containers.volumeMounts.readOnly` flag is set, volumes are mounted
read-only as expected. However, this workflow leaves the read-only vs read-write
decision up to the user requesting storage.

While moving forward we plan to look at ways the driver could better enforce
read only capabilities when access modes are specified, doing so will likely
require us to deviate slightly from the CSI spec. In the meantime one workaround
is to set permissions on static BeeGFS directories so they cannot be
overwritten. Note pods running with root permissions could ignore this.

### Long paths may cause errors 

The `volume_id` used by this CSI is in the format of a Uniform Resource
Identifier (URI) generated by aggregating several fields' values including a
path within a BeeGFS file system.
- In the case of dynamic provisioning, the fields within the StorageClass object
  (`sc`) and CreateVolumeRequest message (`cvr`) combine to yield the
  `volume_id`:
  `beegfs://{sc.parameters.sysMgmtdHost}/{sc.parameters.volDirBasePath}/{cvr.name}`
- In the case of static provisioning, the `volume_id` is written directly by the
  administrator into the Persistent Volume object (`pv`) as the
  `pv.spec.volumeHandle`. 

In either case the resulting `volume_id` URI is generally of the format
`beegfs://ip-or-domain-name/path/to/sub/directory/volume_name`.

The `volume_id`, like all string field values, is subject to a 128 byte limit
unless overridden in the CSI spec: 

> CSI defines general size limits for fields of various types (see table below).
> The general size limit for a particular field MAY be overridden by specifying
> a different size limit in said field's description. Unless otherwise
> specified, fields SHALL NOT exceed the limits documented here. These limits
> apply for messages generated by both COs and plugins.
>
> | Size       | Field Type          |
> |------------|---------------------|
> | 128 bytes  | string              |
> | 4 KiB      | map<string, string> |

Source: [CSI Specification v1.5.0 Size
Limits](https://github.com/container-storage-interface/spec/blob/v1.5.0/spec.md#size-limits)

[CSI specification
v1.4.0](https://github.com/container-storage-interface/spec/releases/tag/v1.4.0)
relaxed the size limit for some file paths and increased the limit for the
`node_id` field specifically to 192 bytes. [CSI specification
v1.5.0](https://github.com/container-storage-interface/spec/releases/tag/v1.5.0)
further increased the size limit for the `node_id` field to 256 bytes. However,
the `volume_id` size limit is unchanged.

Some cursory testing of a few CO and CSI deployments suggest that the limits are
not strictly enforced.  So, rather than impose strict failures or warnings in
the event that CSI spec field limits are exceeded, we have elected to only
document the possibility that long paths may cause errors.
