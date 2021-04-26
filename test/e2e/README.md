### Basic Information

The driver package (./driver) provides two TestDriver
(k8s.io/kubernetes/e2e/storage/testsuites) implementations that are compatible
with the Kubernetes end-to-end storage tests.

* The BeegfsDriver implementation is capable of testing both the static and 
  the dynamic provisioning workflow. The below commands test the BeegfsDriver 
  implementation against beegfs-csi-driver specific tests in the 
  testsuites(./testsuites) package. 
* The BeegfsDynamicDriver implementation is only capable of testing the dynamic
  provisioning workflow. The below commands test the BeegfsDynamicDriver 
  implementation against many of the Kubernetes end-to-end storage tests.

The Jenkinsfile runs a subset of all possible tests. It uses the environment 
and command line arguments to ensure the tests run and known slow and/or 
broken tests are skipped appropriately.

### Requirements

The below commands must be run from a workstation or CI node in an environment 
that meets the following criteria:

* The workstation or CI node must have kubectl access to a Kubernetes cluster
  and the KUBECONFIG environment variable must be properly set.
* The accessible Kubernetes cluster must have beegfs-csi-driver deployed.
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

### Environments

csi-beegfs-config.yaml and csi-beegfs-connauth.yaml files FOR NETAPP INTERNAL 
TESTING are currently located in test/manual/<beegfs-version>. Deploy the 
driver to a cluster with one of these files to get started.

NOTE: These files reference NetApp internally available test clusters and file 
systems. To run tests externally, provide your own KUBECONFIG, 
csi-beegfs-config.yaml, and csi-beegfs-connauth.yaml.
  
### Test Commands

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