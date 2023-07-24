# Contributing <!-- omit in toc -->
Thank you for your interest in contributing to the BeeGFS CSI driver project! 🎉

We appreciate that you want to take the time to contribute! Please follow these
steps before submitting your PR.

# Contents <!-- omit in toc -->

- [Developer Certificate of Origin (DCO)](#developer-certificate-of-origin-dco)
- [Creating a Pull Request](#creating-a-pull-request)
- [BeeGFS CSI Driver Team's Commitment](#beegfs-csi-driver-teams-commitment)

# Developer Certificate of Origin (DCO)

At this time signing a formal Contributor License Agreement is not required, but this requirement may be instated in the future. By contributing to this project, you agree to v1.1 of the [Developer Certificate of Origin (DCO)](https://developercertificate.org/), a copy of which is included before. This document was created by the Linux Kernel community and is a simple statement that you, as a contributor, have the legal right to make the contribution. 

```text
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.

Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.


Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

# Creating a Pull Request

1. Please search [existing
   issues](https://github.com/ThinkParQ/beegfs-csi-driver/issues) to determine if
   an issue already exists for what you intend to contribute.
2. If the issue does not exist, [create a new
   one](https://github.com/ThinkParQ/beegfs-csi-driver/issues/new) that explains
   the bug or feature request.
   * Let us know in the issue that you plan on creating a pull request for it.
   This helps us to keep track of the pull request and avoid any duplicate
   efforts.
3. Before creating a pull request, write up a brief proposal in the issue
   describing what your change would be and how it would work so that others can
   comment.
    * It's better to wait for feedback from the maintainers before writing code.
      We don't have an SLA for our feedback, but we will do our best to respond
      in a timely manner (at a minimum, to give you an idea if you're on the
      right track and that you should proceed, or not).

# BeeGFS CSI Driver Team's Commitment
While we truly appreciate your efforts on pull requests, we **cannot** commit to
including your PR in the BeeGFS CSI driver project. Here are a few reasons why:

* There are many factors involved in integrating new code into this project
  including:
  * Adding appropriate unit and end-to-end test coverage for new/changed
    functionality. 
  * Ensuring adherence with ThinkParQ and industry standards around security and
    licensing. 
  * Validating new functionality doesn't raise long-term maintainability and/or
    supportability concerns.    
  * Verifying changes fit with the current and/or planned architecture. 
  * etc. 

  In other words, while your bug fix or feature may be perfect as a standalone
  patch, we have to ensure the changes also work in all use cases, supported,
  configurations, and across our support matrix.

* The BeeGFS CSI driver team must plan resources to integrate your code into our
  code base and CI platform, and depending on the complexity of your PR, we may
  or may not have the resources available to make it happen in a timely fashion.
  We'll do our best, but typically the earliest changes can be merged into the
  master branch is with our next formal release, unless they resolve a critical
  bug or security vulnerability. 

* Sometimes a PR doesn't fit into our future plans or conflicts with other items
  on the roadmap. It's possible that a PR you submit doesn't align with our
  upcoming plans, thus we won't be able to use it. It's not personal and why we
  highly recommend submitting an issue with your proposed changes so we can
  provide feedback before you expend significant effort on development. 

Thank you for considering to contribute to the BeeGFS CSI driver project. 
