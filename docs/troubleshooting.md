# Troubleshooting Guide

This section provides guidance and tips around troubleshooting issues that
come up using the driver. For anything not covered here, please [submit an
issue](https://github.com/NetApp/beegfs-csi-driver/issues) using the label
"question". Suspected bugs should be submitted with the label "bug". 

## Kubernetes

### Determining the BeeGFS Client Configuration for a PVC
<a name="determining-the-beegfs-client-configuration-for-a-pvc"></a>

BeeGFS Client configuration is specified in a Kubernetes ConfigMap, that is
parsed out to generate the client configuration that applies to a PVC for a
particular BeeGFS file system mounted to a particular Kubernetes node. How to
define this configuration is described in depth in the
[Deployment](deployment.md#general-configuration) documentation. You can
validate the deployed ConfigMap as follows: 

1. Run the following to confirm the name of the ConfigMap currently being used
   by the driver: 
```
joe-mac-0:Desktop joe$ kubectl get pod -n kube-system csi-beegfs-controller-0 -o=custom-columns="CURRENT-BEEGFS-CONFIGMAP:spec.volumes[?(@.name=='config-dir')].configMap.name"
CURRENT-BEEGFS-CONFIGMAP
csi-beegfs-config-9bk9mkb5k7
```
2. Use the resulting ConfigMap name to describe the contents as follows:
```
joe-mac-0:Desktop joe$ kubectl describe ConfigMap -n kube-system csi-beegfs-config-9bk9mkb5k7
Name:         csi-beegfs-config-9bk9mkb5k7
Namespace:    kube-system
Labels:       <none>
```

In some cases administrators may wish to validate the final configuration the
driver parsed out for a particular PVC. For the following steps to work the PVC
must have been bound to a PV, and that PVC must be in use by a running pod.

1. Determine the name of the volume that corresponds with the PVC you want to
   investigate with `kubectl get pvc`. In this example it is `pvc-3ad5dffc`.
```
joe-mac-0:Desktop joe$ kubectl get pvc -A
NAMESPACE   NAME                 STATUS   VOLUME         CAPACITY   ACCESS MODES   STORAGECLASS        AGE
default     csi-beegfs-dyn-pvc   Bound    pvc-3ad5dffc   1Gi        RWX            csi-beegfs-dyn-sc   22h
```
2. Determine the node a Pod consuming the PVC is running on with `kubectl get
   pod -o wide`. In this example it is `ictm1625h12`.
```
joe-mac-0:Desktop joe$ kubectl get pod -o wide
NAME                 READY   STATUS    RESTARTS   AGE   IP               NODE          NOMINATED NODE   READINESS GATES
csi-beegfs-dyn-app   1/1     Running   0          22h   10.233.124.237   ictm1625h12   <none>           <none>
```
3. SSH to the node in question then run `mount | grep <PV_NAME>`, for example: 
```
user@ictm1625h12:~$ mount | grep pvc-3ad5dffc
beegfs_nodev on /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-3ad5dffc/globalmount/mount type beegfs (rw,relatime,cfgFile=/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-3ad5dffc/globalmount/beegfs-client.conf)
beegfs_nodev on /var/lib/kubelet/pods/aa002809-c443-42f1-970c-e2e9c7ca14ad/volumes/kubernetes.io~csi/pvc-3ad5dffc/mount type beegfs (rw,relatime,cfgFile=/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-3ad5dffc/globalmount/beegfs-client.conf)
```
4. The file referenced by cfgFile (both mounts reference the same file) is the
   actual BeeGFS client file used to mount the PVC: 
```
user@ictm1625h12:~$ sudo cat /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-3ad5dffc/globalmount/beegfs-client.conf | grep quotaEnabled
quotaEnabled                 = true
```