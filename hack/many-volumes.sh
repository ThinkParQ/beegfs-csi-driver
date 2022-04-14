#!/bin/bash

# Copyright 2022 NetApp, Inc. All Rights Reserved.
# Licensed under the Apache License, Version 2.0.

# This script was written to enable manual tests in which a large number of Persistent Volume Claims flood Kubernetes 
# (and, by extension, the driver) at once. Use it to generate a YAML manifest containing a Storage Class and any number 
# of Persistent Volume Claims that reference it.

# Example usage:
# cd beegfs-csi-driver/hack
# Modify NUM_PVCS and SYS_MGMTD_HOST.
# ./many-volumes.sh
# kubectl apply -f many-volumes.yaml
# kubectl logs csi-beegfs-controller-0 beegfs
# kubectl delete -f many-volumes.yaml
# kubectl logs csi-beegfs-controller-0 beegfs

NUM_PVCS=100
SYS_MGMTD_HOST="localhost"
EXAMPLE_SC_PATH="../examples/k8s/dyn/dyn-sc.yaml"
EXAMPLE_PVC_PATH="../examples/k8s/dyn/dyn-pvc.yaml"

cat << EOF > many-volumes.yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: many-volumes-sc
provisioner: beegfs.csi.netapp.com
parameters:
  sysMgmtdHost: ${SYS_MGMTD_HOST}
  volDirBasePath: k8s/many-volumes
  permissions/mode: "1644"  # Slow down the driver by forcing a mount.
reclaimPolicy: Delete
volumeBindingMode: Immediate
allowVolumeExpansion: false
EOF

for i in $(seq 0 $(($NUM_PVCS - 1)))
do
# Forgo indention here for the sake of readability in the heredoc.
printf '\n---\n\n' >> many-volumes.yaml
cat << EOF >> many-volumes.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: many-volumes-pvc-${i}
spec:
  accessModes:
  - ReadWriteMany
  resources:
    requests:
      storage: 100Gi
  storageClassName: many-volumes-sc
EOF
done
