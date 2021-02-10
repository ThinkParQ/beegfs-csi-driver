# BeeGFS CSI Driver

## Contents 
* [Overview](#overview)
* [Getting Started](#getting-started)
* [Basic Use](#basic-use)
  * [Examples](#examples)
* [Submitting Feedback and Reporting Issues](#submitting-feedback-and-reporting-issues)
* [License](#license)
* [Maintainers](#maintainers)

## Overview 

The BeeGFS Container Storage Interface (CSI) driver provides high performing and scalable storage for workloads running in container orchestrators like Kubernetes. This driver allows containers to access existing datasets or request on-demand ephemeral or persistent high speed storage backed by [BeeGFS parallel file systems](https://blog.netapp.com/beegfs-for-beginners/). 

### Notable Features

* Integration of storage classes in Kubernetes with [storage pools](https://doc.beegfs.io/latest/advanced_topics/storage_pools.html) in BeeGFS, allowing different tiers of storage within the same file system to be exposed to end users. 
* Management of global and node specific BeeGFS client configuration applied to Kubernetes nodes, simplifying use in large environments. 
* Set [striping parameters](https://doc.beegfs.io/latest/advanced_topics/striping.html) in BeeGFS from storage classes in Kubernetes to optimize for diverse workloads sharing the same file system.
* Support for ReadWriteOnce, ReadOnlyMany, and ReadWriteMany access modes allow workloads distributed across multiple Kubernetes nodes to share access to the same working directories and enable multi-user/application access to common datasets.

### Interoperability and CSI Feature Matrix
| beegfs.csi.netapp.com | K8s Versions  | BeeGFS Versions | CSI Versions | Persistence | Supported Access Modes   | Dynamic Provisioning |
| ----------------------| ------------- | --------------- | ------------ | ----------- | ------------------------ | -------------------- |
| v1.0.0                 | 1.19          | 7.2, 7.1.5      | 1.0+         | Persistent  | Read/Write Multiple Pods | Yes                  |  

Note: This matrix indicates tested BeeGFS and Kubernetes versions. The driver is expected to work with other versions of Kubernetes, but extensive testing has not been performed.

## Getting Started 

### Prerequisite(s) 

* Deploying the driver requires access to a terminal with `kubectl`. 
* The BeeGFS client must be preinstalled to each Kubernetes node that needs BeeGFS access. 
* One or more existing BeeGFS file systems should be available to the Kubernetes nodes over a TCP/IP and/or RDMA (InfiniBand/RoCE) capable network. 

### Quick Start
The steps in this section allow you to get the driver up and running quickly. For production use cases it is recommended to read through the full [deployment guide](docs/deployment.md).

#TODO: Outline steps. 

## Basic Use

 This section provides a quick summary of basic driver use and functionality. Please see the full [usage documentation](docs/usage.md) for a complete overview of all available functionality. The driver was designed to support both dynamic and static storage provisioning and allows directories in BeeGFS to be used as [persistent volumes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) (PVs) in Kubernetes. Pods with persistent volume claims (PVCs) are only able to see/access the specified directory (and any subdirectories), providing isolation between multiple applications and users using the same BeeGFS file system when desired. 

#### Dynamic Storage Provisioning:

Administrators create a storage class in Kubernetes referencing at minimum a specific BeeGFS file system and parent directory within that file system. Users can then submit PVCs against the storage class, and are provided isolated access to new directories under the parent specified in the storage class. 

#### Static Provisioning:

Administrators create a PV and PVC representing an existing directory in a BeeGFS file system. This is useful for exposing some existing dataset or shared directory to Kubernetes users and applications.

### Examples

[Example Kubernetes manifests](examples/README.md) of how to use the driver are provided. These are meant to be repurposed to simplify creating objects related to the driver including storage classes, persistent volumes, and persistent volumes claims in your environment.

## Submitting Feedback and Reporting Issues 

If you have any questions, feature requests, or would like to report an issue please submit them at https://github.com/NetApp/beegfs-csi-driver/issues. 

## License 

Apache License 2.0

## Maintainers 

* Austin Major (@austinmajor).
* Eric Weber (@ejweber).
* Joe McCormick (@iamjoemccormick).
* Joey Parnell (@unwieldy0). 
* Justin Bostian (@jb5n).
