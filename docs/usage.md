# BeeGFS CSI Driver Usage

## Important Concepts

### Definition of a "Volume"

Within the context of this driver, a "volume" is simply a directory within a
BeeGFS filesystem. When a volume is mounted by a Kubernetes Pod, only files
within this directory and its children are accessible by the Pod. An entire
BeeGFS filesystem can be a volume (e.g. by specifying */* as the */path/to/dir*
in the static provisioning workflow) or a single subdirectory many levels deep
can be a volume (e.g. by specifying */a/very/deep/directory* as the
*volDirBasePath* in the dynamic provisioning workflow).

### Capacity

This version of the driver does very little regarding the capacity of its
volumes. Consider the definition of a "volume" above. While an entire BeeGFS
filesystem may have a usable capacity of 100GiB, there is very little meaning
associated with the "usable capacity" of a directory within a BeeGFS (or any
POSIX) filesystem. Future versions of this driver may use BeeGFS enterprise
features like Quota Enforcement (see https://www.beegfs.io/wiki/EnableQuota) to
guarantee that the capacity provisioned by the driver is not exceeded.

In this version, if a Kubernetes Persistent Volume Claim (in the dynamic
provisioning workflow) requests that a new volume have a certain capacity, the
driver will check to make sure there is at least that much capacity remaining on
the BeeGFS filesystem. If there is not, the driver will fail to create the
volume. If there is, the driver will create the volume and report that it has
the requested capacity. Neither the driver nor Kubernetes will do anything else
to guarantee that the volume's capacity specification is not exceeded.

In this version, if a Kubernetes Persistent Volume (in the static provisioning
workflow) is created with a certain capacity, the driver will do absolutely
nothing with that information.

NOTE: None of the examples in this document specify a capacity or capacity
request.

### Static vs Dynamic Provisioning

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

#### Static Provisioning Use Case

As an administrator, I want to make a directory within an existing BeeGFS file
system available to be mounted by multiple users and/or workloads. This
directory probably contains a large, commonly used data set that I don't want to
see copied to multiple locations within my file system. I plan to manage the
volume's lifecycle and I don't want it cleaned up automatically.

As a user, I want to consume an existing data set in my workload.

In the Kubernetes static provisioning workflow, an administrator manually
creates a PV and PVC representing an existing BeeGFS file system directory.
Multiple users and/or workloads can mount that PVC and consume the data the
directory contains.

### BeeGFS Version Compatibility

This version of the driver is ONLY tested for compatibility with BeeGFS
v7.1(.4+) and v7.2. The BeeGFS filesystem services and the BeeGFS clients
running on the Kubernetes nodes MUST be the same major.minor version, and
[beegfsClientConf parameters](deployment.md) passed in the configuration file
MUST apply to the version in use. The driver will log an error and refuse to
start if incompatible configuration is specified.

Future versions of the driver will support future versions of BeeGFS, but no
backwards compatibility with previous versions of BeeGFS is planned. BeeGFS
versions before v7.1.4 do not include the beegfs-dkms package, which the driver
uses to build the BeeGFS client kernel module and mount BeeGFS filesystems. 

### Client Configuration and Tuning

Depending on your topology, different nodes within your cluster or different
BeeGFS file systems accessible by your cluster may need different client
configuration parameters. This configuration is NOT handled at the volume level
(e.g. in a Kubernetes Storage Class or Kubernetes Persistent Volume). See
[deployment.md](deployment.md) for detailed instructions on how to prepare your
cluster for various BeeGFS file system backends.

## Dynamic Provisioning Workflow

### Assumptions

1. A BeeGFS filesystem with its management service listening at *sysMgmtdHost*
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

Specify the filesystem and parent directory using the *sysMgmtdHost* and
*volDirBasePath* parameters respectively.

Striping parameters that can be specified using the beegfs-ctl command line
utility in the *--setpattern* mode can be passed with the prefix 
*stripePattern/* in the *parameters* map as in the example. If no striping 
parameters are passed, the newly created subdirectory will have the same 
striping configuration as its parent. The following parameters have been tested 
with the driver:

* *storagepoolid*
* *chunksize*
* *numtargets*

NOTE: The effects of unlisted configuration options are NOT tested with the
driver. Contact your BeeGFS support representative for recommendations on
appropriate settings. See https://www.beegfs.io/wiki/Striping for additional
details.

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: my-storage-class
provisioner: beegfs.csi.netapp.com
parameters:
  sysMgmtdHost: 10.113.72.217
  volDirBasePath: /path/to/parent/dir 
  stripePattern/storagePoolID: "1"
  stripePattern/chunkSize: 512k
  stripePattern/numTargets: "4"
reclaimPolicy: Delete
volumeBindingMode: Immediate
allowVolumeExpansion: false
```

### Create a Persistent Volume Claim

Who: A Kubernetes user

Specify the Kubernetes Storage Class using the *storageClassName* field in the
Kubernetes Persistent Volume Claim *spec* block.

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

## Static Provisioning Workflow

### Assumptions

1. A BeeGFS filesystem with its management service listening at *sysMgmtdHost*
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
interest into a Pod from the *volumeHandle* field in the *csi* block of the
Persistent Volume *spec* block. It MUST be formatted as modeled in the example.

NOTE: The driver does NOT provide a way to modify the stripe settings of a
directory in the static provisioning workflow. Any *beegfsStripe/* prefixed
parameters set here will be ignored.

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: my-pv
spec:
  accessModes:
    - ReadWriteMany
  persistentVolumeReclaimPolicy: Retain
  csi:
    driver: beegfs.csi.netapp.com
    volumeHandle: beegfs://sysMgmtdHost/path/to/dir
```

### Create a Persistent Volume Claim

Who: A Kubernetes administrator or user

Each Persistent Volume Claim participates in a 1:1 mapping with a Persistent
Volume. Create a Persistent Volume Claim and set the *volumeName* field to
ensure it maps to the correct Persistent Volume.

```yaml
piVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-pvc
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: ""
  volumeName: my-pv
```

### Create a Pod, Deployment, Stateful Set, etc.

Who: A Kubernetes user

Follow standard Kubernetes practices to deploy a Pod that consumes the newly
created Kubernetes Persistent Volume Claim.

## Limitations and Known Issues

### 0777 mode BeeGFS directories created during provisioning

BeeGFS directories created by this driver during provisioning have mode 0777.

### Long paths may cause errors 

The `volume_id` used by this CSI is in the format of a Uniform Resource Identifier (URI) generated by aggregating several fields' values incuding a path within a BeeGFS file system.
- In the case of dynamic provisioning, the fields within the StorageClass object (`sc`) and CreateVolumeRequest message (`cvr`) combine to yield the `volume_id`: `beegfs://{sc.parameters.sysMgmtdHost}/{sc.parameters.volDirBasePath}/{cvr.name}`
- In the case of static provisioning, the fields within the PersistentVolume object (`pv`) and CreateVolumeRequest message (`cvr`) combine to yield the `volume_id`: `{pv.spec.csi.volumeHandle}/{cvr.name}`

In either case the resulting `volume_id` URI is generally of the format `beegfs://ip-or-domain-name/path/to/sub/directory/volume_name`.

The `volume_id`, like all string field values, is subject to a 128 byte limit unless overridden in the CSI spec: 

> CSI defines general size limits for fields of various types (see table below).
> The general size limit for a particular field MAY be overridden by specifying a different size limit in said field's description.
> Unless otherwise specified, fields SHALL NOT exceed the limits documented here.
> These limits apply for messages generated by both COs and plugins.
> 
> | Size       | Field Type          |
> |------------|---------------------|
> | 128 bytes  | string              |
> | 4 KiB      | map<string, string> |

Source: [CSI Specification v1.3.0 Size Limits](https://github.com/container-storage-interface/spec/blob/release-1.3/spec.md#size-limits)

As of Jan. 6, 2021 there is an open pull request ([PR 464](https://github.com/container-storage-interface/spec/pull/464)) to the master branch of the CSI spec that addresses the size limit for some file paths and the `node_id`.  However, the `volume_id` size limit is unchanged.  PR 464 was discussed during the 11/11/2020 CSI Community Meeting.  The 
[agenda, notes](https://docs.google.com/document/d/1-oiNg5V_GtS_JBAEViVBhZ3BYVFlbSz70hreyaD7c5Y/edit#heading=h.9pryrcuoevnn), and [recording](https://youtu.be/Nkgw6aCOQqk) are available online.  Relevant discussion is recorded between timestamps 0:00 and 20:25.

Some cursory testing of a few CO and CSI deployments suggest that the limits are not strictly enforced.  So, rather than impose strict failures or warnings in the event that CSI spec field limits are exceeded, we have elected to only document the possibility that long paths may cause errors.
