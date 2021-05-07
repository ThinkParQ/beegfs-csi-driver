# Changelog
Notable changes to the BeeGFS CSI driver will be documented in this file. 

[1.1.0] - 2021-05-07
--------------------
### Added 
- Automated [end-to-end (E2E) testing](test/e2e/README.md) leveraging the Kubernetes E2E framework.
- Support for [BeeGFS Connection Based Authentication](https://doc.beegfs.io/latest/advanced_topics/authentication.html).
- Support for BeeGFS v7.2.1 and Kubernetes v1.18 and v1.20. 
- The ability to specify [permissions](docs/usage.md#permissions) in BeeGFS from Storage Classes in Kubernetes. This simplifies integration with [BeeGFS quotas](Quotas). 

### Changed
- Explicitly set the CSI driver's `fsGroupPolicy` to `None` disabling [fsGroup](docs/usage.md#fsgroup-behavior) support to prevent time consuming and/or unintended permissions and ownership changes on Kubernetes clusters that [support this parameter](https://kubernetes-csi.github.io/docs/support-fsgroup.html).
- Improved logging, in particular simplifying identification of logs associated with a particular request.
- Updated the project to use Golang 1.16. 

### Fixed
- A race condition when creating volumes where slow running beegfs-ctl commands could prevent volume creation.
- The error returned on NodeStageVolume to align with the CSI spec when we fail to stage a volume because the driver can't find it.

### Security 
NOTE: The BeeGFS CSI Driver undergoes extensive security scanning before each release, and third party components with identified security issues will be updated before each release regardless if they are exploitable in the driver. Going forward only security issues deemed to be exploitable will be noted in the changelog. 

[1.0.1] - 2021-03-10
--------------------
### Changed
- Updated default deployment manifests to account for environments where the SYS_ADMIN capability is insufficient for the controller pod to mount BeeGFS when cleaning up deleted volumes (for example when AppArmor is in use).

### Security
- Updated Dockerfile to use Alpine 3.13.2 which mitigates an OpenSSL vulnerability ([CVE-2021-23840](https://nvd.nist.gov/vuln/detail/CVE-2021-23840)).

[1.0.0] - 2021-02-10
--------------------
- Initial Release
