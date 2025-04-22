#!/bin/bash
set -euo pipefail

# Prerequisites:
# * The BeeGFS client must be installed to the node running Minikube.
# * The /etc/beegfs directory must be mounted to all Minikube "nodes" (i.e., containers/VMs):
#   `minikube mount /etc/beegfs:/etc/beegfs &`

# IMPORTANT: This script is not idempotent, notably the step to create directories in BeeGFS.

export BEEGFS_VERSION=7.4.6
export BEEGFS_SECRET=mysecret

# Deploy BeeGFS file system:
envsubst < ../test/env/beegfs-ubuntu/beegfs-fs-1.yaml | kubectl apply -f -

MAX_ATTEMPTS=36
SLEEP_TIME=5
COUNTER=0

# If we try to expose the service to the host OS before the pod is ready we'll get an error.
# Make sure the BeeGFS FS started before we continue.
while [ $COUNTER -lt $MAX_ATTEMPTS ]; do
POD_STATUS=$(kubectl get pods beegfs-fs-1-0 -o jsonpath='{.status.phase}')
echo "Pod status: ${POD_STATUS}"
if [ "${POD_STATUS}" == "Running" ]; then
    echo "Verified BeeGFS FS pod is running."
    break
else
    echo "Pod is not running, waiting for ${SLEEP_TIME} seconds..."
    sleep ${SLEEP_TIME}
    COUNTER=$((COUNTER+1))
fi
done

if [ $COUNTER -eq $MAX_ATTEMPTS ]; then
echo "BeeGFS FS pod did not reach 'Running' status within the maximum allowed time. Outputting debug information and exiting with error..."
kubectl get pods -A
kubectl describe pod beegfs-fs-1-0
docker images
exit 1
fi

# Adapted from https://minikube.sigs.k8s.io/docs/handbook/accessing/
# This is required to mount BeeGFS since the kernel module is outside the container.
# For some reason we don't need to override the ephemeral port and can use the actual 800* ports.
minikube service beegfs-fs-1-svc

# Update client configuration so ctl can interact with the file system:
sudo echo $BEEGFS_SECRET | sudo tee /etc/beegfs/connAuth
sudo sed -i '/connAuthFile = \/etc\/beegfs\/connAuth/! {0,/connAuthFile[[:space:]]*=[[:space:]]*/s//connAuthFile = \/etc\/beegfs\/connAuth/}' /etc/beegfs/beegfs-client.conf
sudo sed -i '/sysMgmtdHost = localhost/! {0,/sysMgmtdHost[[:space:]]*=[[:space:]]*/s//sysMgmtdHost = localhost/}' /etc/beegfs/beegfs-client.conf

# Note the /etc/beegfs directory should already be mounted to the Minikube container: 
# `minikube mount /etc/beegfs:/etc/beegfs &`

# Precreate directories for static provisioning examples:
minikube ssh "sudo beegfs-ctl --cfgFile=/etc/beegfs/beegfs-client.conf --unmounted --createdir /k8s"
minikube ssh "sudo beegfs-ctl --cfgFile=/etc/beegfs/beegfs-client.conf --unmounted --createdir /k8s/all"
minikube ssh "sudo beegfs-ctl --cfgFile=/etc/beegfs/beegfs-client.conf --unmounted --createdir /k8s/all/static"
minikube ssh "sudo beegfs-ctl --cfgFile=/etc/beegfs/beegfs-client.conf --unmounted --createdir /k8s/all/static-ro"
export BEEGFS_MGMTD=$(kubectl get nodes -o=jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
for file in ../examples/k8s/all/*; do sed -i 's/localhost/'"${BEEGFS_MGMTD}"'/g' "$file"; done
kubectl apply -f ../examples/k8s/all
