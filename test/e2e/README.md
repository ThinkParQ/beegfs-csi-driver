### Basic Information

The driver package (./driver) provides multiple TestDriver 
(k8s.io/kubernetes/e2e/storage/testsuites) implementations that are compatible 
with the Kubernetes end-to-end storage tests. Running one of the test commands 
below tests these implementations against many of the Kubernetes end-to-end 
storage test, as well as beegfs-csi-driver specific tests in the testsuites 
(./testsuites) package.

### Requirements

The below commands must be run from a workstation or CI node in an environment 
that meets the following criteria:

* At least one file system must be appropriately described in a 
  test_config.yaml file at --per-fs-configs-path on the workstation or CI node.
* The workstation or CI node must have kubectl access to a Kubernetes cluster 
  and the KUBECONFIG environment variable must be properly set.
* The accessible Kubernetes cluster must have access to all file systems 
  described in the test_config.yaml.
* The accessible Kubernetes cluster must have beegfs-csi-driver deployed.
* The deployed beegfs-csi-driver must be configured so that it can interact 
  with all file systems in the test_config.yaml.
* The workstation or CI node must have all file systems described in the 
  test_config.yaml mounted at the locations described in the test_config.yaml.
* The workstation or CI node must have passwordless SSH access to all worker 
  nodes in the accessible Kubernetes cluster. This access is configurable using 
  environment variables. The SSH user must have sudo capabilities.
  
### Test Commands

Template for `ginkgo` command:

```bash
KUBECONFIG=/absolute/path/to/kubeconfig \
KUBE_SSH_KEY=/absolute/path/to/ssh/key \
KUBE_SSH_USER= root \ # or some user with sudo capabilities
ginkgo \
--focus beegfs-suite  # or no focus for all K8s tests as well
./test/e2e \
-- \
--report-dir ./junit \  # or wherever you want the files (one per parallel node)
--per-fs-configs-path=/absolute/path/to/test_config.yml
```

Actual `ginkgo` command that works on my local machine:

```bash
KUBECONFIG=/home/eric/.kube/config ginkgo --focus beegfs-suite ./test/e2e -- --report-dir ./junit --per-fs-configs-path=/home/eric/beegfs-csi-driver/test/e2e/example_test_config.yml
```

Template for `go test` command:

```bash
KUBECONFIG=/absolute/path/to/kubeconfig \
KUBE_SSH_KEY=/absolute/path/to/ssh/key \
KUBE_SSH_USER= root \ # or some user with sudo capabilities
go test ./test/e2e/ \
-ginkgo.focus beegfs-suite \  # or no focus for all K8s tests as well
-ginkgo.v \ 
-test.v \
-report-dir ./junit \  # or wherever you want the files (one per parallel node)
-per-fs-configs-path=/absolute/path/to/test_config.yml
```

Actual `go test` command that works on my local machine:

```bash
KUBECONFIG=/home/eric/.kube/config go test ./test/e2e/ -ginkgo.focus beegfs-suite -ginkgo.v -test.v -report-dir ./junit -per-fs-configs-path=/home/eric/beegfs-csi-driver/test/e2e/example_test_config.yml
```