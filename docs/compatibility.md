# BeeGFS CSI Driver Compatibility <!-- omit in toc -->

The goal for the BeeGFS CSI driver is to maintain compatibility with relevant
versions of Kubernetes as well as relevant version of BeeGFS. With this goal in
mind the following notes are the guidelines we use when planning for changes to
version compatibility and testing.

***

## Contents <!-- omit in toc -->

- [Kubernetes](#kubernetes)
  - [Supporting New Kubernetes Releases](#supporting-new-kubernetes-releases)
  - [Dropping Compatibility Support For Old Kubernetes Releases](#dropping-compatibility-support-for-old-kubernetes-releases)
- [BeeGFS](#beegfs)
  - [BeeGFS Version Support](#beegfs-version-support)
- [OpenShift](#openshift)
- [Nomad](#nomad)

***

## Kubernetes

The BeeGFS CSI driver will generally support the set of Kubernetes releases
supported by the Kubernetes project itself at the time of the driver release.
Information about Kubernetes release support status can be found on the
[Kubernetes releases](https://kubernetes.io/releases/) page.

Occasionally, a particular Kubernetes patch version may be required to 
guarantee smooth driver operation. See the [Troubleshooting 
Guide](docs/troubleshooting.md) for known issues.

### Supporting New Kubernetes Releases

The criteria for including a new Kubernetes release, meaning a new Kubernetes
minor version, for compatibility testing is the following: It is able to be set
up via a released version of
[Minikube](https://minikube.sigs.k8s.io/docs/) (used to maintain
BeeGFS CSI driver test environments).

### Dropping Compatibility Support For Old Kubernetes Releases

Kubernetes versions that become unsupported during the development cycle of a
new BeeGFS CSI driver version will be considered for removal from the
compatibility testing matrix. We make a "best effort" to maintain compatibility
with one out-of-support version as an acknowledgement that Kubernetes has a 
fast moving release cycle and upgrading environments can take time. However,
if any issues arise when using the driver on a Kubernetes version that is
out of support, the first recommendation is to upgrade Kubernetes.

***

## BeeGFS

The BeeGFS CSI driver primarily interacts with the BeeGFS client and so the
version of the client is the focus of our compatibility testing. Testing BeeGFS
client and server compatibility directly is outside the scope of the BeeGFS CSI
driver testing. Therefore, unless explicitly noted otherwise, the identification
of a BeeGFS version should be interpreted as an environment where all BeeGFS
components (client and server) are running the same specified version.

For guidance on compatibility between BeeGFS clients and servers refer to 
the [BeeGFS documentation](https://doc.beegfs.io/latest/index.html)

### BeeGFS Version Support

Starting with version 8, BeeGFS more strictly adheres to semantic versioning guarantees. As a result
each version of the driver is now only tested with a single version of BeeGFS 8 (usually the latest
at the time of each driver release), but all 8.x versions prior to that version should also be
compatible. This decision was made to reduce the size of the test matrix and CI runtime.

When a new version of BeeGFS is released we will add that new version to the test matrix and drop
the last version. A new version of the driver may not be released unless changes are required to
support the new version, which is generally not anticipated.

Support for at least one BeeGFS 7 release will continue at least until BeeGFS 7 reached
[EOL](https://github.com/ThinkParQ/beegfs/blob/master/SUPPORT.md).

***

## OpenShift

Starting from version v1.5.0, the BeeGFS CSI driver will no longer be tested
with RedHat OpenShift. While we expect that the driver will continue to function
with RedHat OpenShift, we have decided to suspend direct testing against this
platform due to the associated licensing and other costs of maintaining the
necessary test environments. If this change impacts your use of the driver, we
encourage you to open an issue so we can understand your specific use case and
discuss potential avenues to resume testing with OpenShift, or alternatively,
with the community version, OKD.

***

## Nomad

While we have made [initial investments](deploy/nomad/README.md) into enabling 
the use of the BeeGFS CSI driver with HashiCorp Nomad, we may not test with 
Nomad for every driver release and do not currently consider Nomad to be a 
supported container orchestrator.