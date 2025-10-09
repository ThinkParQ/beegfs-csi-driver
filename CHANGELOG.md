# Changelog
Notable changes to the BeeGFS CSI driver will be documented in this file.

[1.8.0] - PRERELEASE
--------------------

### Added
- Support for BeeGFS 8
  - See the [v1.8.0 upgrade notes](deploy/k8s/README.md#upgrading-to-v180-and-beegfs-8) before
    upgrading to BeeGFS 8 for details on new configuration specific to BeeGFS 8.
- Support for Kubernetes v1.32, v1.33, and v1.34.

### Changed
- Migrated from the `kube-rbac-proxy` sidecar to the built-in controller-runtime protection.
  - This change only affects the operator deployment path, and does not require user action.

### Deprecated
- Kubernetes v1.29 and v1.30 support will be dropped in the next driver release according to our
  [support
  policy](docs/compatibility.md#dropping-compatibility-support-for-old-kubernetes-releases).

### Removed 
- Support/testing for Kubernetes v1.27 and v1.28.
- Support/testing for BeeGFS 7.3.

[1.7.0] - 2024-11-04
--------------------

### Added
- Support for BeeGFS v7.4.5, Kubernetes v1.29, v1.30, and v1.31.
- Support for volume resizing.
  - Note that volume capacity still has no effect when using the driver. However, support for
    resizing can be helpful for applications that rely on the size of the Persistent Volume (PV) or
    Persistent Volume Claim (PVC) as indicated in the Kubernetes API.

### Deprecated
- Kubernetes v1.27 and v1.28 support will be dropped in the next driver release according to our
  [support
  policy](docs/compatibility.md#dropping-compatibility-support-for-old-kubernetes-releases).
- BeeGFS 7.3 support will be dropped in the next driver release. Users are advised to upgrade to
  BeeGFS 7.4.

### Removed 
- Support/testing for Kubernetes v1.25 and v1.26.

[1.6.0] - 2024-02-28
--------------------

### Added
- Support for BeeGFS v7.4.2 and Kubernetes v1.28.
- Support for arm64 and official multi-arch container images for all supported platforms
  (linux/amd64 and linux/arm64).

### Deprecated
- Kubernetes v1.25 and v1.26 support will be dropped in the next driver release according to our
  [support
  policy](docs/compatibility.md#dropping-compatibility-support-for-old-kubernetes-releases).

### Removed 
- Support/testing for Kubernetes v1.23 and v1.24.

[1.5.0] - 2023-09-11
--------------------

### Added
- Support for Kubernetes 1.26 and 1.27.
- Support for BeeGFS v7.3.4 and v7.4.0.
- Support for binary connAuthFile secrets utilizing base64 encoding.

### Changed
- Migrated project to the ThinkParQ GitHub organization.
- Updated deployment manifests to accommodate new container registries. See the
  [upgrade instructions](deploy/k8s/README.md##upgrading-to-v150) if you were
  previously overriding image names or tags with a Kustomize overlay.
  - BeeGFS CSI driver container images have been migrated from DockerHub to
  GitHub Container Registry. This changes the default driver container name from
  `docker.io/netapp/beegfs-csi-driver` to
  `ghcr.io/thinkparq/beegfs-csi-driver:v1.5.0`.
  - [Kubernetes CSI sidecar
  containers](https://kubernetes-csi.github.io/docs/sidecar-containers.html)
  have been migrated from `k8s.gcr.io` to `registry.k8s.io` since the former has
  been
  [deprecated](https://kubernetes.io/blog/2023/03/10/image-registry-redirect/).

### Deprecated
- Kubernetes v1.23 and v1.24 support will be dropped in the next driver release according
  to our [support
  policy](docs/compatibility.md#dropping-compatibility-support-for-old-kubernetes-releases).

### Removed
- Testing/support for RedHat OpenShift. 
  - [See the compatibility documentation](docs/compatibility.md#openshift) for more information about this change.
- Testing/support for Kubernetes v1.22 and BeeGFS v7.2.x. 

[1.4.0] - 2022-12-12
--------------------
### Added
- Support for RedHat OpenShift v4.11.
- Support for BeeGFS v7.3.2 and BeeGFS v7.2.8.
- Support for Kubernetes v1.25.
- Added default container resource requests and limits along with documentation
  for how to modify the resource specifications.
- Container images will now be signed with
  [Cosign](https://github.com/sigstore/cosign). Documentation on how to verify
  the signatures has been added to the [deployment guide](docs/deployment.md)
  and the [operator README](operator/README.md).
- Added [documentation](docs/usage.md#managing-readonly-volumes) for read-only
  volumes.
  
### Changed
- Updated the project to adhere to v1.7.0 of the CSI specification.
- Updated the operator-sdk used to v1.25.0
- Changed the default driver container name from `netapp/beegfs-csi-driver` to
  `docker.io/netapp/beegfs-csi-driver`. See the [upgrade
  instructions](deploy/k8s/README.md#upgrade-1.2.0-kubernetes-deployment) if you
  were previously overriding this name with a Kustomize overlay.
- Improved testing for Nomad deployments.
- Updated [Nomad documentation](docs/nomad.md) to reflect Alpha maturity level.

### Deprecated
- Kubernetes v1.22 support will be dropped in the next driver release according
  to our [support
  policy](docs/compatibility.md#dropping-compatibility-support-for-old-kubernetes-releases).

### Fixed
- Implemented verification of user provided stripePattern values.
- Default container logs will now be from the driver instead of csi-provisioner
  when executing "kubectl log" commands.

### Removed
- Support for Kubernetes v1.21.
- Support for RedHat OpenShift v4.10.

### Security
- Mitigated [CVE-2022-28948](https://nvd.nist.gov/vuln/detail/CVE-2022-28948) by
  upgrading go-yaml to v3.0.1.
- Mitigated [CVE-2022-27664](https://nvd.nist.gov/vuln/detail/CVE-2022-27664) by
  upgrading Go to v1.18.7.

[1.3.0] - 2022-08-22
--------------------
### Added
- Support for Kubernetes v1.24.
- Support for BeeGFS v7.3.1 and BeeGFS v7.2.7. See the new [Notable BeeGFS
  Client Parameters](docs/deployment.md#notable) and [BeeGFS Helperd
  Configuration](docs/deployment.md#beegfs-helperd-configuration) sections in
  the deployment guide for important notes when upgrading BeeGFS to these
  versions.
- The [Readme](README.md) now includes links to demo videos for a quick start
  guide, the dynamic provisioning workflow, and the static provisioning
  workflow.
- Generalized Nomad deployment and example manifests that work on Nomad v1.3.3
  and greater.

### Changed
- Updated the project to adhere to v1.6.0 of the CSI specification.
- Updated the operator-sdk used to v1.22.2
- Changed the [default BeeGFS mount options](docs/usage.md#beegfs-mount-options)
  to include the `nosuid` mount option.
- Refactor validation of parameters for CreateVolume and
  ValidateVolumeCapabilities.
- We are now checking for the BeeGFS client kernel module earlier in the driver
  initialization process in order to better identify potential driver
  initialization failures.
- Replaced usage of k8s.io/utils/mount to use k8s.io/mount-utils instead.

### Deprecated
- Kubernetes v1.21 support will be dropped in the next driver release according
  to our [support
  policy](docs/compatibility.md#dropping-compatibility-support-for-old-kubernetes-releases).

### Fixed
- Removed duplicate messages that were occurring in the driver logs for certain
  errors.

### Removed
- Support for BeeGFS v7.1.x.
- Support for Kubernetes v1.20.
- Single node Nomad deployment and example manifests that worked before Nomad v1.3.0.

### Security
- Mitigated [CVE-2022-1996](https://nvd.nist.gov/vuln/detail/CVE-2022-1996) by
  upgrading go-restful to v2.16.0
- Mitigated [CVE-2022-29526](https://nvd.nist.gov/vuln/detail/CVE-2022-29526),
  [CVE-2022-30629](https://nvd.nist.gov/vuln/detail/CVE-2022-30629), and
  [CVE-2022-32189](https://nvd.nist.gov/vuln/detail/CVE-2022-32189) by upgrading
  to Go v1.17.13

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
- Support (testing) for BeeGFS v7.1.5 (to be removed in the next release).

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
- Support (testing) for BeeGFS v7.2.5, Kubernetes v1.19, and RedHat OpenShift
  v4.9.

### Security
- Mitigated
  [CVE-2022-23772](https://security.netapp.com/advisory/ntap-20220225-0006/) by
  upgrading to Go v1.17.9.
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
