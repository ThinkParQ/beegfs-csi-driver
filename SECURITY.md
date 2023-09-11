# Security Policy for the BeeGFS CSI Driver <!-- omit in toc -->

## Contents <!-- omit in toc -->

- [How to Report](#how-to-report)
- [Response and Handling](#response-and-handling)
- [Disclosure Policy](#disclosure-policy)
- [Supported Versions](#supported-versions)
- [Prevention](#prevention)
- [Acknowledgments](#acknowledgments)

## How to Report

* Please [report](https://github.com/ThinkParQ/beegfs-csi-driver/security) potential security vulnerabilities using [GitHub's private
  vulnerability
  reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability). Make sure
  to not disclose this information in public.
* Provide a detailed description of the potential vulnerability, ensuring you
  include steps that can help in reproducing the issue.

## Response and Handling

While the BeeGFS CSI Driver is an open source project and [support for the
driver](https://github.com/ThinkParQ/beegfs-csi-driver#support) is "best
effort", we will make every effort to response to and resolve security issues in
a timely manner. To that end our goals when handling security issues are:

* Acknowledge every report within three working days.
* Assess the report, evaluate its impact and severity, and determine its
  authenticity providing an new update within five working days.
* Work diligently to address any verified vulnerabilities. While the time to
  deliver a fix will vary depending on complexity, throughout this process,
  we'll provide timely updates on our progress until resolution.
* Once the vulnerability has been fixed, we will make a public announcement
  crediting you for the discovery (unless you wish to remain anonymous).

## Disclosure Policy

Upon confirmation of a security issue, our approach is:

1. Verify the vulnerability and determine affected versions.
2. Develop a fix or a workaround.
3. Upon a successful fix or workaround, inform the community through a public
   advisory.

## Supported Versions

Only the latest version of the BeeGFS CSI driver is supported with security
updates. Users are urged to always use the latest version and generally fixes are not backported to older versions per our [support policy](https://github.com/ThinkParQ/beegfs-csi-driver#support)

## Prevention

To help prevent security vulnerabilities, we:

- Regularly review and update our dependencies using Dependabot and CodeQL.
  
- Adhere to best coding practices and conduct regular code reviews.
  
- Actively seek feedback and input from our developer community on security matters.

## Acknowledgments

We're thankful to our community for their active involvement in enhancing the
safety of our project. Those who've identified vulnerabilities are recognized in
the CHANGELOG, unless they've opted for anonymity.
