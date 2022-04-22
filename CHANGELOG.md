# Changelog
Notable changes to the BeeGFS CSI driver will be documented in this file. 

[1.2.2] - 2022-05-09
--------------------
### Added
- Support for BeeGFS v7.2.6, BeeGFS v7.3.0, Kubernetes v1.23, and RedHat
  OpenShift v4.10.
- [Basic support](docs/deployment.md#security-considerations-selinux) for
  SELinux-enabled nodes.
- [Experimental support](deploy/openshift-beegfs-client/README.md) for deploying
  the BeeGFS client to OpenShift RHCOS nodes. The driver is still only
  officially supported in OpenShift on RHEL nodes.

### Changed
- The driver now fails in initialization if it does not detect a running BeeGFS
  client kernel module. Previously it would not fail until it served the first
  request.
- If the `client-conf-template-path` command line parameter is not specified,
  the driver now looks for a beegfs-client.conf file in multiple expected
  locations. It still looks in the previous default location
  `/etc/beegfs/beegfs-client.conf` first. 
  
### Deprecated
- Testing for BeeGFS v7.1.5 (to be removed in the next release).

### Fixed
- Slow but successful CreateVolume operations may never return an OK status
  within the time frame that the client is listening. This typically only occurs
  in environments with misconfigured BeeGFS networking.
- ValidateVolumeCapabilities returns an INTERNAL error code when an invalid
  volume ID is included in a request instead of a NOT_FOUND error code (as
  required by the CSI spec).
- DeleteVolume returns an INTERNAL error code when an invalid volume ID is
  included in a request instead of OK (as required by the CSI spec).
- Minor issues related to end-to-end testing.

### Removed
- Testing for BeeGFS v7.2.5, Kubernetes v1.19, and RedHat OpenShift v4.9

### Security
- Updated multiple vulnerable dependencies to versions not known to be
  vulnerable.
- Completed a threat model of the controller service and made minor
  documentation improvements in response.

[1.2.1] - 2021-12-20
--------------------
### Added
- Support for BeeGFS v7.2.5, Kubernetes v1.22, and RedHat OpenShift v4.9. 
- The ability to persist state in BeeGFS using a .csi/ directory structure that
  exists alongside dynamically provisioned volumes in their `volDirBasePath`.
  This is automatically enabled by default but can be [optionally
  disabled](docs/usage.md#notes-for-beegfs-administrators).

### Fixed
- Common causes of [orphaned BeeGFS
  mounts](docs/troubleshooting.md#orphan-mounts) being left on Kubernetes nodes
  (listed as a known issue in v1.2.0) by maintaining a record of nodes with
  active BeeGFS mounts for each volume in the new .csi/ directory and falling
  back on a newly added timeout (`--node-unstage-timeout`) when needed.

### Security
Note: The BeeGFS CSI driver is written in Golang and does not import or
implement any functionality that makes it susceptible to the recent Log4j
vulnerability threat. For more details please refer to [NetApp's official
response](https://www.netapp.com/newsroom/netapp-apache-log4j-response/).

[1.2.0] - 2021-10-11
--------------------
### Added
- A new [BeeGFS CSI Driver Operator](operator/README.md) as an option to deploy
  and manage the lifecycle of the driver. This allows for a more seamless
  [discovery](https://operatorhub.io/) and installation experience from clusters
  running Operator Lifecycle Manager (OLM).
- [Documentation and job specifications](deploy/nomad/README.md) showing how to
  deploy the driver to HashiCorp Nomad.
  - Note: At this time the BeeGFS CSI driver does not officially support Nomad.
    These are being provided as an example for others who might want to
    experiment with using BeeGFS and Nomad, in particular anyone interested in
    [contributing](CONTRIBUTING.md) to any future efforts around Nomad.
- [Documentation](docs/deployment.md#mixed-kubernetes-deployment) on how to
  deploy the driver to Kubernetes clusters where some nodes can access BeeGFS
  volumes, and some cannot.
- Support for BeeGFS v7.2.4, Kubernetes v1.21, and RedHat OpenShift v4.8. 
- Support for specifying [BeeGFS mount
  options](docs/usage.md#beegfs-mount-options) on a persistent volume or storage
  class.
- Information on how to [contribute](CONTRIBUTING.md) to the project. 

### Changed
- Greatly improved performance of end-to-end testing by parallelizing many tests
  and being more selective about when certain tests run.
- Updated the project to adhere to v1.5.0 of the CSI specification.

### Fixed
- Automated tests failing in a confusing manner when `csi-beegfs-config.yaml` is
  empty. 

### Known Issues
- In some instances Kubernetes has been observed to call `DeleteVolume` prior to
  `NodeUnpublishVolume` and `NodeUnstageVolume`. This has the effect of leaving
  behind BeeGFS mount points on Kubernetes nodes for volumes that no longer
  exist in the Kubernetes API or BeeGFS. Over time if enough "orphaned" mounts
  accrue, the Kubernetes node may become unstable. To date this has only been
  observed as part of end-to-end testing, and is suspected to be either a side
  effect of how the E2E test framework interacts with Kubernetes, or a bug
  within Kubernetes itself.

[1.1.0] - 2021-05-10
--------------------
### Added 
- Automated [end-to-end (E2E) testing](test/e2e/README.md) leveraging the
  Kubernetes E2E framework.
- Support for [BeeGFS Connection Based
  Authentication](https://doc.beegfs.io/latest/advanced_topics/authentication.html).
- Support for BeeGFS v7.2.1 and Kubernetes v1.18 and v1.20. 
- The ability to specify [permissions](docs/usage.md#permissions) in BeeGFS from
  Storage Classes in Kubernetes. This simplifies integration with [BeeGFS
  quotas](docs/quotas.md). 

### Changed
- Explicitly set the CSI driver's `fsGroupPolicy` to `None` disabling
  [fsGroup](docs/usage.md#fsgroup-behavior) support to prevent time consuming
  and/or unintended permissions and ownership changes on Kubernetes clusters
  that [support this
  parameter](https://kubernetes-csi.github.io/docs/support-fsgroup.html).
- Improved logging, in particular simplifying identification of logs associated
  with a particular request.
- Updated the project to use Golang 1.16. 

### Fixed
- A race condition when creating volumes where slow running beegfs-ctl commands
  could prevent volume creation.
- The error returned on NodeStageVolume to align with the CSI spec when we fail
  to stage a volume because the driver can't find it.

### Security 
NOTE: The BeeGFS CSI Driver undergoes extensive security scanning before each
release, and third party components with identified security issues will be
updated before each release regardless if they are exploitable in the driver.
Going forward only security issues deemed to be exploitable will be noted in the
changelog. 

[1.0.1] - 2021-03-10
--------------------
### Changed
- Updated default deployment manifests to account for environments where the
  SYS_ADMIN capability is insufficient for the controller pod to mount BeeGFS
  when cleaning up deleted volumes (for example when AppArmor is in use).

### Security
- Updated Dockerfile to use Alpine 3.13.2 which mitigates an OpenSSL
  vulnerability
  ([CVE-2021-23840](https://nvd.nist.gov/vuln/detail/CVE-2021-23840)).

[1.0.0] - 2021-02-10
--------------------
- Initial Release
