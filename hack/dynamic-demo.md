## BeeGFS Dynamic Volume Provisioning Demo

### Purpose

This document and its associated script `./dynamic-demo-alias.sh` demonstrate 
the basic functionality of the driver. These commands can be run against a 
locally running driver as a quick sanity test or to demonstrate the driver's 
behavior outside of the more complex confines of a container orchestrator (i.e. 
Kubernetes). `./dynamic-demo-alias.sh` contains more commands than are 
documented here.

### Assumptions

* The Container Storage Client (csc) tool is installed.
       
        go get github.com/rexray/gocsi
        
* The CSI is running.

        # the particular node id chosen does not matter
        bin/beegfs-csi-driver --node-id node1 -v 4 --cs-data-dir /tmp/csdatadir
        
* A BeeGFS file system exists with sysMgmtdHost=10.113.72.217 (substitute your 
own value) and the SYS_MGMTD_HOST environment variable is set.
  
        export SYS_MGMTD_HOST=10.193.114.48
  
* dynamic-demo-alias.sh has been sourced.

        source dynamic-demo-alias.sh

### Demonstrations

#### CreateVolume

##### Command

    createvolume
    
##### Result

    INFO[0000] /csi.v1.Controller/CreateVolume: REQ 0001: Name=pvc-00000001, VolumeCapabilities=[mount:<> access_mode:<mode:MULTI_NODE_MULTI_WRITER > ], Parameters=map[sysMgmtdHost:10.193.114.48 volDirBasePath:kubernetes], XXX_NoUnkeyedLiteral={}, XXX_sizecache=0 
    INFO[0004] /csi.v1.Controller/CreateVolume: REP 0001: Volume=volume_id:"beegfs://10.193.114.48/kubernetes/pvc-00000001" , XXX_NoUnkeyedLiteral={}, XXX_sizecache=0 
    "beegfs://10.193.114.48/kubernetes/pvc-00000001"        0

##### Evidence

A subdirectory named kubernetes/pvc-00000001 exists on the BeeGFS file system where one did not exist before.

#### NodeStageVolume

##### Command

    nodestagevolume

#### Result

    INFO[0000] /csi.v1.Node/NodeStageVolume: REQ 0001: VolumeId=beegfs://10.193.114.48/kubernetes/pvc-00000001, StagingTargetPath=/tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount, VolumeCapability=mount:<> access_mode:<mode:MULTI_NODE_MULTI_WRITER > , XXX_NoUnkeyedLiteral={}, XXX_sizecache=0 
    INFO[0001] /csi.v1.Node/NodeStageVolume: REP 0001: XXX_NoUnkeyedLiteral={}, XXX_sizecache=0 
    beegfs://10.193.114.48/kubernetes/pvc-00000001
    
#### Evidence

The beegfs file system is mounted at /tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/mount

    -> mount
    beegfs_nodev on /tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount/mount type beegfs (rw,relatime,cfgFile=/tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount/beegfs-client.conf)

#### NodePublishVolume

##### Command

    nodepublishvolume
    
##### Result

    INFO[0000] /csi.v1.Node/NodePublishVolume: REQ 0001: VolumeId=beegfs://10.193.114.48/kubernetes/pvc-00000001, StagingTargetPath=/tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001, TargetPath=/tmp/kubelet/pods/pod1/volumes/kubernetes.io~csi/pvc-00000001, VolumeCapability=mount:<> access_mode:<mode:MULTI_NODE_MULTI_WRITER > , Readonly=false, XXX_NoUnkeyedLiteral={}, XXX_sizecache=0 
    INFO[0000] /csi.v1.Node/NodePublishVolume: REP 0001: XXX_NoUnkeyedLiteral={}, XXX_sizecache=0 
    beegfs://10.193.114.48/kubernetes/pvc-00000001
    
##### Evidence

The BeeGFS file system is mounted again (bind mounted) at 
/tmp/kubelet/pods/pod1/volumes/kubernetes.io~csi/pvc-00000001/mount. The "pod" 
data directory is empty because the pod only has access to a subdirectory of 
the BeeGFS file system.

    -> mount
    beegfs_nodev on /tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount/mount type beegfs (rw,relatime,cfgFile=/tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount/beegfs-client.conf)
    beegfs_nodev on /tmp/kubelet/pods/pod1/volumes/kubernetes.io~csi/pvc-00000001/mount type beegfs (rw,relatime,cfgFile=/tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount/beegfs-client.conf)
    -> cd /tmp/kubelet/pods/pod1/volumes/kubernetes.io~csi/pvc-00000001/mount
    -> ls
    # no output

##### Additional "Tricks"

Publish the volume again to another "pod". Confirm that it is mounted yet again (bind mounted).

    -> nodepublishvolumepod2
    INFO[0000] /csi.v1.Node/NodePublishVolume: REQ 0001: VolumeId=beegfs://10.193.114.48/kubernetes/pvc-00000001, StagingTargetPath=/tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount, TargetPath=/tmp/kubelet/pods/pod2/volumes/kubernetes.io~csi/pvc-00000001/mount, VolumeCapability=mount:<> access_mode:<mode:MULTI_NODE_MULTI_WRITER > , Readonly=false, XXX_NoUnkeyedLiteral={}, XXX_sizecache=0 
    INFO[0000] /csi.v1.Node/NodePublishVolume: REP 0001: XXX_NoUnkeyedLiteral={}, XXX_sizecache=0 
    beegfs://10.193.114.48/kubernetes/pvc-00000001
    -> mount
    beegfs_nodev on /tmp/csi_data_dir/volume1/beegfs type beegfs (rw,relatime,cfgFile=/tmp/csi_data_dir/volume1/10_113_72_217_beegfs-client.conf)
    beegfs_nodev on /tmp/fake_kubelet_dir/pods/pod1/volumes/volume1 type beegfs (rw,relatime,cfgFile=/tmp/csi_data_dir/volume1/10_113_72_217_beegfs-client.conf)
    beegfs_nodev on /tmp/fake_kubelet_dir/pods/pod2/volumes/volume1 type beegfs (rw,relatime,cfgFile=/tmp/csi_data_dir/volume1/10_113_72_217_beegfs-client.conf)
       
Write data to one "pod" data directory and confirm that it can be read from the other.

    -> cd /tmp/kubelet/pods/pod1/volumes/kubernetes.io~csi/pvc-00000001/mount/
    -> touch written_from_pod1.txt
    -> ls /tmp/kubelet/pods/pod2/volumes/kubernetes.io~csi/pvc-00000001/mount
    written_from_pod1.txt

#### NodeUnpublishVolume

##### Command

If you executed NodePublishVolume for more than one "pod", repeat this command for each "pod".

    nodeunpublishvolume
    
##### Result

    INFO[0000] /csi.v1.Node/NodeUnpublishVolume: REQ 0001: VolumeId=beegfs://10.193.114.48/kubernetes/pvc-00000001, TargetPath=/tmp/kubelet/pods/pod1/volumes/kubernetes.io~csi/pvc-00000001/mount, XXX_NoUnkeyedLiteral={}, XXX_sizecache=0 
    INFO[0000] /csi.v1.Node/NodeUnpublishVolume: REP 0001: XXX_NoUnkeyedLiteral={}, XXX_sizecache=0 
    beegfs://10.193.114.48/kubernetes/pvc-00000001
    
##### Evidence

The BeeGFS file system is still mounted at /tmp/csi_data_dir/volume1/beegfs, but it is no longer mounted in any "pod" 
data directories.

    -> mount
    beegfs_nodev on /tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount/mount type beegfs (rw,relatime,cfgFile=/tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount/beegfs-client.conf)
    
#### NodeUnstageVolume

##### Command

    nodeunstagevolume
    
##### Result

    INFO[0000] /csi.v1.Node/NodeUnstageVolume: REQ 0001: VolumeId=beegfs://10.193.114.48/kubernetes/pvc-00000001, StagingTargetPath=/tmp/kubelet/plugins/kubernetes.io/csi/pv/pvc-00000001/globalmount, XXX_NoUnkeyedLiteral={}, XXX_sizecache=0 
    INFO[0002] /csi.v1.Node/NodeUnstageVolume: REP 0001: XXX_NoUnkeyedLiteral={}, XXX_sizecache=0 
    beegfs://10.193.114.48/kubernetes/pvc-00000001

##### Evidence

The BeeGFS file system is no longer mounted anywhere.
    
    -> mount
    # no output
    
#### DeleteVolume

##### Command

    deletevolume
    
##### Result

    INFO[0000] /csi.v1.Controller/DeleteVolume: REQ 0001: VolumeId=beegfs://10.193.114.48/kubernetes/pvc-00000001, XXX_NoUnkeyedLiteral={}, XXX_sizecache=0 
    INFO[0004] /csi.v1.Controller/DeleteVolume: REP 0001: XXX_NoUnkeyedLiteral={}, XXX_sizecache=0 
    beegfs://10.193.114.48/kubernetes/pvc-00000001
    
##### Evidence

The subdirectory kubernetes/pvc-00000001 no longer exists on the BeeGFS 
file system.
