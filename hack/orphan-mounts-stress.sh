#!/bin/bash

# Set KUBECONFIG.
# Deploy the driver.
# Modify DRIVER_NAMESPACE if necessary.
# Run this script.

START_TIME=${SECONDS}
DRIVER_NAMESPACE=default
OUTPUT_DIR=$(mktemp -d /tmp/e2e.XXXXXX)
export KUBE_SSH_USER=root

exec > >(tee -a ${OUTPUT_DIR}/script.log) 2>&1  # Log all script output to OUTPUT_DIR.

fail() {
  ELAPSED=$((${SECONDS} - ${START_TIME}))
  kubectl logs -n ${DRIVER_NAMESPACE} -c beegfs --since=${ELAPSED}s csi-beegfs-controller-0 >${OUTPUT_DIR}/controller.log
  kubectl get pod -n ${DRIVER_NAMESPACE} -l app=csi-beegfs-node --no-headers | awk '{print $1}' | xargs -I{} kubectl logs -n ${DRIVER_NAMESPACE} -c beegfs --since=${ELAPSED}s {} >>${OUTPUT_DIR}/nodes.log
  echo "see ${OUTPUT_DIR} for details"
  exit
}

echo "cleaning up previous test"
kubectl get ns --no-headers | awk '{print $1}' | grep -e provisioning- -e stress- -e beegfs- -e multivolume- -e ephemeral- -e volumemode- -e volumelimits- | xargs kubectl delete ns --cascade=foreground 2>/dev/null
echo

for ITERATION in {1..20}; do
  echo iteration ${ITERATION}
  date

  echo "running nondisruptive ginkgo tests"
  if ginkgo -p -nodes 8 -skip 'should be able to unmount after the subpath directory is deleted|\[Slow\]|\[Disruptive\]|\[Serial\]|should delete only the anticipated directory' -timeout 60m ./test/e2e/ >${OUTPUT_DIR}/nondisruptive.log; then
    echo "nondisruptive ginkgo tests passed"
  else
    echo "nondisruptive ginkgo tests failed"
    fail
  fi

  echo "running disruptive ginkgo tests"
  if ginkgo -v -noColor -skip '${ginkgoSkipRegex}' -focus '\\[Disruptive\\]|\\[Serial\\]' -timeout 60m ./test/e2e/ >${OUTPUT_DIR}/disruptive.log; then
    echo "disruptive ginkgo tests passed"
  else
    echo "disruptive ginkgo tests failed"
    fail
  fi

  echo
done

ELAPSED=$((${SECONDS} - ${START_TIME}))
# Print any controller logs that indicate either:
# -> The orphan mount prevention infrastructure worked, (waited for unstage) or
# -> The orphan mount infrastructure had difficulties, (failed to clean up due to a busy mount).
kubectl logs -n ${DRIVER_NAMESPACE} -c beegfs --since=${ELAPSED}s csi-beegfs-controller-0 | grep -e Waiting -e busy
rm -rf ${OUTPUT_DIR}
