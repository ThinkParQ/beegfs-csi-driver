# BeeGFS CSI Driver Usage

<a name="contents"></a>
## Contents

* [Important Concepts](#important-concepts)
* [Dynamic Provisioning Workflow](#dynamic-provisioning-workflow)
* [Static Provisioning Workflow](#static-provisioning-workflow)
* [Best Practices](#best-practices)
* [Notes for BeeGFS Administrators](#notes-for-beegfs-administrators)
* [Limitations and Known Issues](#limitations-and-known-issues)

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

<a name="beegfs-version-compatibility"></a>
### BeeGFS Version Compatibility

This version of the driver is ONLY tested for compatibility with BeeGFS v7.1.5
and v7.2. The BeeGFS filesystem services and the BeeGFS clients running on the
Kubernetes nodes MUST be the same major.minor version, and [beegfsClientConf
parameters](deployment.md) passed in the configuration file MUST apply to the
version in use. The driver will log an error and refuse to start if incompatible
configuration is specified.

Future versions of the driver will support future versions of BeeGFS, but no
backwards compatibility with previous versions of BeeGFS is planned. BeeGFS
versions before v7.1.4 do not include the beegfs-client-dkms package, which the
driver uses to build the BeeGFS client kernel module and mount BeeGFS file
systems. 

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

| Prefix         | Parameter     | Required | Accepted patterns                       | Example    | Default
| ------         | ---------     | -------- | -----------------                       | -------    | -------
| stripePattern/ | storagePoolID | no       | unsigned integer                        | 1          | file system default
| stripePattern/ | chunkSize     | no       | unsigned integer + k (kilo) or m (mega) | 512k<br>1m | file system default
| stripePattern/ | numTargets    | no       | unsigned integer                        | 4          | file system default

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

| Prefix       | Parameter | Required | Accepted patterns                  | Example     | Default 
| ------       | --------- | -------- | -----------------                  | -------     | -------
| permissions/ | uid       | no       | unsigned integer                   | 1000        | 0 (root)
| permissions/ | gid       | no       | unsigned integer                   | 1000        | 0 (root)
| permissions/ | mode      | no       | three or four digit octal notation | 755<br>0755 | 0777

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
or [storage class
class](https://kubernetes.io/docs/concepts/storage/storage-classes/#mount-options).
Please note the driver DOES NOT validate provided mount options and use of
options not supported by BeeGFS may cause unpredictable behavior. 

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
