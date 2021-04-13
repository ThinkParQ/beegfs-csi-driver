TEST_ROOT="$(dirname $(realpath $0))"
KUBECONFIG="${KUBECONFIG:-$(realpath ~/.kube/config)}"
echo $KUBECONFIG

set -x

e2e.test \
  -ginkgo.focus='External.Storage' \
  -storage.testdriver="${TEST_ROOT}/basic-driver.yaml" \
  -kubeconfig="${KUBECONFIG}" \
  -repo-root="${TEST_ROOT}"
