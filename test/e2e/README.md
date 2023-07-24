# BeeGFS CSI Driver End-to-End Testing <!-- omit in toc -->

> **Warning**
> This document has not been updated yet to reflect the project's migration to the ThinkParQ organization.

## Contents <!-- omit in toc -->
- [Overview](#overview)
- [Requirements](#requirements)
- [CI Environments](#ci-environments)
- [Test Commands](#test-commands)
  - [Ginkgo](#ginkgo)
  - [Go Test](#go-test)

## Overview

The driver package (./driver) provides two TestDriver
(k8s.io/kubernetes/e2e/storage/testsuites) implementations that are compatible
with the Kubernetes end-to-end storage tests.

* The BeegfsDriver implementation is capable of testing both the static and 
  the dynamic provisioning workflow. [Commands given below](#test-commands) 
  test the BeegfsDriver implementation against beegfs-csi-driver specific tests 
  in the testsuites (./testsuites) package. 
* The BeegfsDynamicDriver implementation is only capable of testing the dynamic
  provisioning workflow. [Commands given below](#test-commands) test the 
  BeegfsDynamicDriver implementation against many of the Kubernetes end-to-end 
  storage tests.

The Jenkinsfile runs a subset of all possible tests. It uses the environment 
and command line arguments to ensure the tests run and known slow and/or 
broken tests are skipped appropriately.

## Requirements

Test commands must be run from a workstation or CI node in an environment 
that meets the following criteria:

* The workstation or CI node must have kubectl access to a Kubernetes cluster
  and the KUBECONFIG environment variable must be properly set.
* The accessible Kubernetes cluster must have the driver deployed.
* The csi-beegfs-config.yaml used to deploy the driver to the cluster must 
  EXPLICITLY refer to the sysMgmtdHost of at least one fileSystemSpecificConfig.

Some tests have additional requirements. If the requirements are not met, they 
may be skipped OR they may fail.

* Each BeeGFS filesystem referred to in the csi-beegfs-config.yaml used to 
  deploy the driver must have a pre-existing */e2e-test/static/static1* 
  directory. If it does not, "beegfs-suite" tests using the "Preprovisioned-PV" 
  pattern fail.
* The testing workstation or CI node must have passwordless SSH access to 
  cluster nodes. If it does not, tests marked \[Disruptive\] fail.
* The csi-beegfs-config.yaml used to deploy the driver to the accessible cluster
  must EXPLICITLY refer to the sysMgmtdHost of at least TWO 
  fileSystemSpecificConfigs. If only one is referenced, some "beegfs-suite" 
  tests are skipped.
* The csi-beegfs-config.yaml used to deploy the driver to the accessible cluster
  must contain at least ONE fileSystemSpecificConfig with 
  config.beegfsClientConf.connUseRDMA set to "true". If none are configured 
  this way, RDMA specific "beegfs-suite" tests do not run.
* The cluster must have at least TWO schedulable worker nodes. If it only has 
  one, some multivolume tests are skipped.
  
Certain additional environment attributes increase the coverage of the tests:

* Including a "no effect" beegfs-client.conf parameter (e.g. sysMgmtdHost) in 
  csi-beegfs-config.yaml confirms that "no effect" parameters don't break the 
  driver.
* Configuring a BeeGFS file system in a nonstandard way and then including that 
  configuration in csi-beegfs-config.yaml (e.g. connMgmtdPortTCP) confirms that 
  beegfs-client.conf parameters are respected.
* Configuring a BeeGFS file system so that it requires a connAuthFile and then 
  including a csi-beegfs-connauth.yaml during driver deployment confirms that 
  connAuth features work as expected.

## CI Environments

The driver is tested in a variety of environments during the development and 
release life cycle. csi-beegfs-config.yaml and csi-beegfs-connauth.yaml files 
FOR NETAPP INTERNAL TESTING are located in test/env/<beegfs-version>. For each 
test run, Jenkins automatically deploys the driver to a cluster with one set of 
these files during initialization.

To run end-to-end tests outside of NetApp, provide your own KUBECONFIG 
(referencing and existing cluster), csi-beegfs-config.yaml, and 
csi-beegfs-connauth.yaml (referencing an existing BeeGFS file system).
  
## Test Commands

The Kubernetes end-to-end tests are built on top of the 
[Ginkgo BDD Testing Framework](https://onsi.github.io/ginkgo/). They can be 
launched using the Ginkgo CLI (if it is installed) or by using the typical 
`go test` command.

### Ginkgo

Template for `ginkgo` command:

```bash
KUBECONFIG=/absolute/path/to/kubeconfig \
ginkgo \
--focus "some string|some other string" \
--skip "some string|some other string" \
./test/e2e \
-- \
--report-dir ./junit
```

NOTE: Control which specs run using --focus and --skip as described 
[here](https://onsi.github.io/ginkgo/).

Actual `ginkgo` command that works on a local machine:

```bash
KUBECONFIG=~/.kube/config ginkgo --focus beegfs-suite ./test/e2e -- --report-dir ./junit
```

Test runs can be parallelized using the `-p` flag. Each test suite will be run serially,
and within each suite a number of processes will be run in parallel. Tests will be divided between
these processes, with each process running its share of the test suite serially. These processes
will not share data, so it is acceptable to use a single BeegfsDriver for a number of tests in a suite.

### Go Test

Template for `go test` command:

```bash
KUBECONFIG=/absolute/path/to/kubeconfig \
go test ./test/e2e/ \
-ginkgo.focus "some string|some other string" \
-ginkgo.skip "some string|some other string" \
-ginkgo.v \ 
-test.v \
-report-dir ./junit
```

Actual `go test` command that works on a local machine:

```bash
KUBECONFIG=~/.kube/config go test ./test/e2e/ -ginkgo.focus beegfs-suite -ginkgo.v -test.v -report-dir ./junit
```

NOTE: Control which specs run using --focus and --skip as described
[here](https://onsi.github.io/ginkgo/).