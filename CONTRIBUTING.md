# Contributing
Thank you for your interest in contributing to the BeeGFS CSI driver project! 🎉

We appreciate that you want to take the time to contribute! Please follow these
steps before submitting your PR.

# Creating a Pull Request

1. Please search [existing
   issues](https://github.com/NetApp/beegfs-csi-driver/issues) to determine if
   an issue already exists for what you intend to contribute.
2. If the issue does not exist, [create a new
   one](https://github.com/NetApp/beegfs-csi-driver/issues/new) that explains
   the bug or feature request.
3. Let us know in the issue that you plan on creating a pull request for it.
   This helps us to keep track of the pull request and avoid any duplicate
   efforts.
4. Before creating a pull request, write up a brief proposal in the issue
   describing what your change would be and how it would work so that others can
   comment.
5. It's better to wait for feedback from someone on NetApp's BeeGFS CSI driver
   development team before writing code. We don't have an SLA for our feedback,
   but we will do our best to respond in a timely manner (at a minimum, to give
   you an idea if you're on the right track and that you should proceed, or
   not).
6. Sign and submit NetApp's Contributor License Agreement. You must sign and
   submit the [Corporate Contributor License Agreement
   (CCLA)](https://netapp.na1.echosign.com/public/esignWidget?wid=CBFCIBAA3AAABLblqZhBN1IcwqxqFOpZRSYcKO8V4nppnNJaH2QnbKIaJXBcoBFoGiQXAUjSamaZAvzBJOUM*)
   in order to contribute. 
7. Make sure you specify `NetApp/BeeGFS CSI Driver` or
   `https://github.com/NetApp/beegfs-csi-driver` for the **Project Name**.
8. If you've made it this far, have written the code that solves your issue, and
   addressed the review comments, then feel free to create your pull request.

Important: **NetApp will NOT look at the PR or any of the code submitted in the
PR if the CCLA is not on file with NetApp Legal.**

# BeeGFS CSI Driver Team's Commitment
While we truly appreciate your efforts on pull requests, we **cannot** commit to
including your PR in the BeeGFS CSI driver project. Here are a few reasons why:

* There are many factors involved in integrating new code into this project
  including:
  * Adding appropriate unit and end-to-end test coverage for new/changed
    functionality. 
  * Ensuring adherence with NetApp and industry standards around security and
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
