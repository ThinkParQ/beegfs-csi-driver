### Test Commands

Template for `ginkgo` command:

```bash
KUBECONFIG=/absolute/path/to/kubeconfig \
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