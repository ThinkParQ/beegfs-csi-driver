# Copyright 2021 NetApp, Inc. All Rights Reserved.
# Licensed under the Apache License, Version 2.0.

# Source this file to simplify the execution of some GRPC calls against a locally running driver.

# NOTE: ./dynamic-demo.md explains the operation of some of these commands and documents ways to verify they have run
# successfully. This file contains more commands than are documented in ./dynamic-demo.md.

# Run the plugin like "sudo bin/beegfs-csi-driver --node-id node1 -v 4 --cs-data-dir /tmp/csdatadir".

# Set SYS_MGMTD_HOST in the environment or default to a BeeGFS file system running on localhost. This change applies
# at alias time (not at run time).
SYS_MGMTD_HOST="${SYS_MGMTD_HOST:-localhost}"

mkdir -p /tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount /tmp/kubelet/pods/pod1/volumes/kubernetes.io~csi/pvc-00000001 /tmp/kubelet/pods/pod2/volume/kubernetes.io~csi/pvc-00000001 /tmp/csdatadir

alias csc='sudo -i csc -e /tmp/csi.sock -l info --with-request-logging --with-response-logging'
alias createvolume="csc controller create-volume pvc-00000001 --cap MULTI_NODE_MULTI_WRITER,mount, --params sysMgmtdHost=${SYS_MGMTD_HOST},volDirBasePath=kubernetes"
alias createvolumestripepattern="csc controller create-volume pvc-00000001 --cap MULTI_NODE_MULTI_WRITER,mount, --params sysMgmtdHost=${SYS_MGMTD_HOST},volDirBasePath=kubernetes,stripePattern/storagePoolID=2,stripePattern/chunkSize=1m,stripePattern/numTargets=2"
alias createvolumepermissions="csc controller create-volume pvc-00000001 --cap MULTI_NODE_MULTI_WRITER,mount, --params sysMgmtdHost=${SYS_MGMTD_HOST},volDirBasePath=kubernetes,permissions/uid=1000,permissions/gid=1000,permissions/mode=0755"
alias createvolumespecialpermissions="csc controller create-volume pvc-00000001 --cap MULTI_NODE_MULTI_WRITER,mount, --params sysMgmtdHost=${SYS_MGMTD_HOST},volDirBasePath=kubernetes,permissions/uid=1000,permissions/gid=1000,permissions/mode=2755"
alias nodestagevolume="csc node stage --staging-target-path=/tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount --cap MULTI_NODE_MULTI_WRITER,mount, beegfs://${SYS_MGMTD_HOST}/kubernetes/pvc-00000001"
alias nodepublishvolume="csc node publish --staging-target-path=/tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount --target-path=/tmp/kubelet/pods/pod1/volumes/kubernetes.io~csi/pvc-00000001/mount --cap MULTI_NODE_MULTI_WRITER,mount, beegfs://${SYS_MGMTD_HOST}/kubernetes/pvc-00000001"
alias nodepublishvolumereadonly="csc node publish --staging-target-path=/tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount --target-path=/tmp/kubelet/pods/pod1/volumes/kubernetes.io~csi/pvc-00000001/mount --cap MULTI_NODE_MULTI_WRITER,mount, --read-only beegfs://${SYS_MGMTD_HOST}/kubernetes/pvc-00000001"
alias nodepublishvolumepod2="csc node publish --staging-target-path=/tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount --target-path=/tmp/kubelet/pods/pod2/volumes/kubernetes.io~csi/pvc-00000001/mount --cap MULTI_NODE_MULTI_WRITER,mount, beegfs://${SYS_MGMTD_HOST}/kubernetes/pvc-00000001"
alias nodepublishvolumepodreadonly2="csc node publish --staging-target-path=/tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount --target-path=/tmp/kubelet/pods/pod2/volumes/kubernetes.io~csi/pvc-00000001/mount --cap MULTI_NODE_MULTI_WRITER,mount, --read-only beegfs://${SYS_MGMTD_HOST}/kubernetes/pvc-00000001"
alias nodeunpublishvolume="csc node unpublish --target-path=/tmp/kubelet/pods/pod1/volumes/kubernetes.io~csi/pvc-00000001/mount beegfs://${SYS_MGMTD_HOST}/kubernetes/pvc-00000001"
alias nodeunpublishvolumepod2="csc node unpublish --target-path=/tmp/kubelet/pods/pod2/volumes/kubernetes.io~csi/pvc-00000001/mount beegfs://${SYS_MGMTD_HOST}/kubernetes/pvc-00000001"
alias nodeunstagevolume="csc node unstage --staging-target-path=/tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount beegfs://${SYS_MGMTD_HOST}/kubernetes/pvc-00000001"
alias deletevolume="csc controller delete-volume beegfs://${SYS_MGMTD_HOST}/kubernetes/pvc-00000001"
alias validatevolumecapabilities="csc controller validate-volume-capabilities --cap MULTI_NODE_MULTI_WRITER,mount, beegfs://${SYS_MGMTD_HOST}/kubernetes/pvc-00000001"
alias validatevolumecapabilitiesfail="csc controller validate-volume-capabilities --cap MULTI_NODE_MULTI_WRITER,block, beegfs://${SYS_MGMTD_HOST}/kubernetes/pvc-00000001"
