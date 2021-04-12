### Basic Information

The driver package (./driver) provides two TestDriver
(k8s.io/kubernetes/e2e/storage/testsuites) implementations that are compatible
with the Kubernetes end-to-end storage tests. The BeegfsDriver implementation is
capable of testing the static and dynamic provisioing workflows, while the
BeegfsDynamicDriver implementation is only capable of testing the dynamic
provisioing workflow. Running one of the commands below tests these
BeegfsDynamicDriver implementation against many of the Kubernetes end-to-end
storage tests and the BeegfsDriver implementation against beegfs-csi-driver
specific tests in the testsuites
(./testsuites) package.

### Requirements

The below commands must be run from a workstation or CI node in an environment 
that meets the following criteria:

* The workstation or CI node must have kubectl access to a Kubernetes cluster
  and the KUBECONFIG environment variable must be properly set.
* The accessible Kubernetes cluster must have beegfs-csi-driver deployed.
* The csi-beegfs-config.yaml used to deploy the driver to the accessible cluster 
  must EXPLICITLY refer to the sysMgmtdHost of at least one 
  fileSystemSpecificConfig.

Some tests have additional requirements. If the requirements are not met, they 
will be skipped. These requirements include:
* The csi-beegfs-config.yaml used to deploy the driver to the accessible cluster
  must EXPLICITLY refer to the sysMgmtdHost of at least TWO 
  fileSystemSpecificConfigs.
* The csi-beegfs-config.yaml used to deploy the driver to the accessible cluster
  must contain at least ONE fileSystemSpecificConfig with 
  config.beegfsClientConf.connUseRDMA set to "true".

### Environments

Correctly configured csi-beegfs-config.yaml files for driver deployment are 
currently located in test/e2e/manual/<beegfs-version>. Deploy the driver to a 
cluster with one of these files to start testing.
  
### Test Commands

Template for `ginkgo` command:

```bash
KUBECONFIG=/absolute/path/to/kubeconfig \
ginkgo \
--focus beegfs-suite  # or no focus for all K8s tests as well
./test/e2e \
-- \
--report-dir ./junit \  # or wherever you want the files (one per parallel node)
```

Actual `ginkgo` command that works on my local machine:

```bash
KUBECONFIG=/home/eric/.kube/config ginkgo --focus beegfs-suite ./test/e2e -- --report-dir ./junit
```

Template for `go test` command:

```bash
KUBECONFIG=/absolute/path/to/kubeconfig \
go test ./test/e2e/ \
-ginkgo.focus beegfs-suite \  # or no focus for all K8s tests as well
-ginkgo.v \ 
-test.v \
-report-dir ./junit \  # or wherever you want the files (one per parallel node)
```

Actual `go test` command that works on my local machine:

```bash
KUBECONFIG=/home/eric/.kube/config go test ./test/e2e/ -ginkgo.focus beegfs-suite -ginkgo.v -test.v -report-dir ./junit
```