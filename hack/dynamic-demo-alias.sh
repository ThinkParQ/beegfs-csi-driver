# Source this file to execute the complex commands in ../docs/beegfs/dynamic-demo.md without having to list out (or
# copy and paste) all the parameters.

# TODO(webere): Fix or remove dynamic-demo.md.
# NOTE: ../docs/beegfs/dynamic-demo.md is OUT OF DATE! These commands work, but they are not reflected correctly in
# dynamic-demo.md.

# Run the plugin like "sudo bin/beegfsplugin --nodeid node1".

# Set SYS_MGMTD_HOST in the environment or default to a BeeGFS file system running on localhost. This change applies
# at alias time (not at run time).
SYS_MGMTD_HOST="${SYS_MGMTD_HOST:-localhost}"

mkdir -p /tmp/kubelet/plugins/beegfs/volume1 /tmp/kubelet/pods/pod1 /tmp/kubelet/pods/pod2 /csi-data-dir
# Or modify dataRoot to be something more friendly, like /tmp/csi-data-dir.

alias csc='sudo -i csc -e /tmp/csi.sock'
alias createvolume="csc controller create-volume volume1 --cap MULTI_NODE_MULTI_WRITER,mount, --params sysMgmtdHost=${SYS_MGMTD_HOST},volDirBasePath=scratch"
alias nodestagevolume="csc node stage --staging-target-path=/tmp/kubelet/plugins/beegfs/volume1 --cap MULTI_NODE_MULTI_WRITER,mount, --vol-context=volDirBasePath=scratch beegfs://${SYS_MGMTD_HOST}/scratch/volume1"
alias nodepublishvolume="csc node publish --staging-target-path=/tmp/kubelet/plugins/beegfs/volume1 --target-path=/tmp/kubelet/pods/pod1/volumes/volume1 --cap MULTI_NODE_MULTI_WRITER,mount, --vol-context=volDirBasePath=scratch beegfs://${SYS_MGMTD_HOST}/scratch/volume1"
alias nodepublishvolumepod2="csc node publish --staging-target-path=/tmp/kubelet/plugins/beegfs/volume1 --target-path=/tmp/kubelet/pods/pod2/volumes/volume1 --cap MULTI_NODE_MULTI_WRITER,mount, --vol-context=volDirBasePath=scratch beegfs://${SYS_MGMTD_HOST}/scratch/volume1"
alias nodeunpublishvolume="csc node unpublish --target-path=/tmp/kubelet/pods/pod1/volumes/volume1 beegfs://${SYS_MGMTD_HOST}/scratch/volume1"
alias nodeunpublishvolumepod2="csc node unpublish --target-path=/tmp/kubelet/pods/pod2/volumes/volume1 beegfs://${SYS_MGMTD_HOST}/scratch/volume1"
alias nodeunstagevolume="csc node unstage --staging-target-path=/tmp/kubelet/plugins/beegfs/volume1 beegfs://${SYS_MGMTD_HOST}/scratch/volume1"
alias deletevolume="csc controller delete-volume beegfs://${SYS_MGMTD_HOST}/scratch/volume1"