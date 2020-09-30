# Source this file to execute the complex commands in ../docs/beegfs/dynamic-demo.md without having to list out (or
# copy and paste) all the parameters.

alias createvolume='csc controller create-volume volume1 --cap "MULTI_NODE_MULTI_WRITER,mount," --params "volDirBasePath=scratch,beegfsConf/sysMgmtdHost=10.113.72.217"'
alias nodestagevolume='csc node stage --staging-target-path="/tmp/csi_data_dir/volume1" --cap "MULTI_NODE_MULTI_WRITER,mount," --vol-context="beegfsConf/sysMgmtdHost=10.113.72.217,volDirBasePath=scratch" beegfs://10.113.72.217/scratch/volume1'
alias nodepublishvolume='csc node publish --staging-target-path="/tmp/csi_data_dir/volume1" --target-path="/tmp/fake_kubelet_dir/pods/pod1/volumes/volume1" --cap "MULTI_NODE_MULTI_WRITER,mount," --vol-context="beegfsConf/sysMgmtdHost=10.113.72.217,volDirBasePath=scratch" beegfs://10.113.72.217/scratch/volume1'
alias nodepublishvolumepod2='csc node publish --staging-target-path="/tmp/csi_data_dir/volume1" --target-path="/tmp/fake_kubelet_dir/pods/pod2/volumes/volume1" --cap "MULTI_NODE_MULTI_WRITER,mount," --vol-context="beegfsConf/sysMgmtdHost=10.113.72.217,volDirBasePath=scratch" beegfs://10.113.72.217/scratch/volume1'
alias nodeunpublishvolume='csc node unpublish --target-path="/tmp/fake_kubelet_dir/pods/pod1/volumes/volume1" beegfs://10.113.72.217/scratch/volume1'
alias nodeunpublishvolumepod2='csc node unpublish --target-path="/tmp/fake_kubelet_dir/pods/pod2/volumes/volume1" beegfs://10.113.72.217/scratch/volume1'
alias nodeunstagevolume='csc node unstage --staging-target-path="/tmp/csi_data_dir/volume1" beegfs://10.113.72.217/scratch/volume1'
alias deletevolume='csc controller delete-volume beegfs://10.113.72.217/scratch/volume1'