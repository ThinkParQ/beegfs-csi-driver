# Changelog
Notable changes to the BeeGFS CSI driver will be documented in this file. 

[1.0.1] - 2021-03-10
--------------------
### Changed
- Updated default deployment manifests to account for environments where the SYS_ADMIN capability is insufficient for the controller pod to mount BeeGFS when cleaning up deleted volumes (for example when AppArmor is in use).

### Security
- Updated Dockerfile to use Alpine 3.13.2 which mitigates an OpenSSL vulnerability ([CVE-2021-23840](https://nvd.nist.gov/vuln/detail/CVE-2021-23840)).

[1.0.0] - 2021-02-10
--------------------
- Initial Release
