## Deployment
The easiest way to test the BeeGFS driver is to run the `deploy.sh` script for the Kubernetes version used by
the cluster as shown below for Kubernetes 1.17. This creates the deployment that is maintained specifically for that
release of Kubernetes. However, other deployments may also work.

```
# deploy beegfs driver
# NOTE: For development purposes it may be useful to set BEEGFSPLUGIN_REGISTRY and BEEGFSPLUGIN_TAG environment variables; Otherwise 'netapp.com/k8scsi' and 'v1.4.0', respectively, are specified in deploy/kubernetes-1.17/csi-beegfs-plugin.yaml.
$ deploy/kubernetes-latest/deploy.sh
```

You should see an output similar to the following printed on the terminal showing the application of rbac rules and the
result of deploying the beegfs driver, external provisioner, and external attacher. Note that the following output is from Kubernetes 1.19:

```shell
applying RBAC rules
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-provisioner/v2.0.2/deploy/kubernetes/rbac.yaml
serviceaccount/csi-provisioner created
clusterrole.rbac.authorization.k8s.io/external-provisioner-runner created
clusterrolebinding.rbac.authorization.k8s.io/csi-provisioner-role created
role.rbac.authorization.k8s.io/external-provisioner-cfg created
rolebinding.rbac.authorization.k8s.io/csi-provisioner-role-cfg created
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-attacher/v3.0.0/deploy/kubernetes/rbac.yaml
serviceaccount/csi-attacher created
clusterrole.rbac.authorization.k8s.io/external-attacher-runner created
clusterrolebinding.rbac.authorization.k8s.io/csi-attacher-role created
role.rbac.authorization.k8s.io/external-attacher-cfg created
rolebinding.rbac.authorization.k8s.io/csi-attacher-role-cfg created
deploying beegfs components
   deploy/kubernetes-latest/beegfs/csi-beegfs-attacher.yaml
        using           image: k8s.gcr.io/sig-storage/csi-attacher:v3.0.0
statefulset.apps/csi-beegfs-attacher created
   deploy/kubernetes-latest/beegfs/csi-beegfs-driverinfo.yaml
Warning: storage.k8s.io/v1beta1 CSIDriver is deprecated in v1.19+, unavailable in v1.22+; use storage.k8s.io/v1 CSIDriver
csidriver.storage.k8s.io/beegfs.csi.netapp.com created
   deploy/kubernetes-latest/beegfs/csi-beegfs-plugin.yaml
        using           image: k8s.gcr.io/sig-storage/csi-node-driver-registrar:v2.0.1
        using           image: netapp.com/k8scsi/beegfsplugin:v1.4.0
        using           image: k8s.gcr.io/sig-storage/livenessprobe:v2.1.0
service/csi-beegfsplugin created
statefulset.apps/csi-beegfsplugin created
   deploy/kubernetes-latest/beegfs/csi-beegfs-provisioner.yaml
        using           image: k8s.gcr.io/sig-storage/csi-provisioner:v2.0.2
statefulset.apps/csi-beegfs-provisioner created
   deploy/kubernetes-latest/beegfs/csi-beegfs-testing.yaml
        using           image: alpine/socat:1.0.3
service/beegfs-service created
statefulset.apps/csi-beegfs-socat created
20:35:11 waiting for beegfs deployment to complete, attempt #0
20:35:21 waiting for beegfs deployment to complete, attempt #1
```

The [livenessprobe side-container](https://github.com/kubernetes-csi/livenessprobe) provided by the CSI community is deployed with the CSI driver to provide the liveness checking of the CSI services.

## Run example application and validate

Next, validate the deployment.  First, ensure all expected pods are running properly including the external attacher, provisioner, and the actual beegfs driver plugin:

```shell
$ kubectl get pods
NAME                         READY   STATUS    RESTARTS   AGE
csi-beegfs-attacher-0        1/1     Running   0          4m21s
csi-beegfs-provisioner-0     1/1     Running   0          4m19s
csi-beegfs-socat-0           1/1     Running   0          4m18s
csi-beegfsplugin-0           3/3     Running   0          4m20s
```

From the root directory, deploy the application pods including a storage class, a PVC, and a pod which mounts a volume using the BeeGFS driver found in directory `./examples`:

```shell
$ for i in ./examples/csi-storageclass.yaml ./examples/csi-pvc.yaml ./examples/csi-app.yaml; do kubectl apply -f $i; done
storageclass.storage.k8s.io/csi-beegfs-sc created
persistentvolumeclaim/csi-pvc created
pod/my-csi-app created
```

Let's validate the components are deployed:

```shell
$ kubectl get pv
NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM             STORAGECLASS      REASON   AGE
pvc-ad827273-8d08-430b-9d5a-e60e05a2bc3e   1Gi        RWO            Delete           Bound    default/csi-pvc   csi-beegfs-sc              45s

$ kubectl get pvc
NAME      STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS      AGE
csi-pvc   Bound    pvc-ad827273-8d08-430b-9d5a-e60e05a2bc3e   1Gi        RWO            csi-beegfs-sc     94s
```

Finally, inspect the application pod `my-csi-app`  which mounts a BeeGFS volume:

```shell
$ kubectl describe pods/my-csi-app
Name:         my-csi-app
Namespace:    default
Priority:     0
Node:         csi-prow-worker/172.17.0.2
Start Time:   Mon, 09 Mar 2020 14:38:05 -0700
Labels:       <none>
Annotations:  kubectl.kubernetes.io/last-applied-configuration:
                {"apiVersion":"v1","kind":"Pod","metadata":{"annotations":{},"name":"my-csi-app","namespace":"default"},"spec":{"containers":[{"command":[...
Status:       Running
IP:           10.244.2.52
IPs:
  IP:  10.244.2.52
Containers:
  my-frontend:
    Container ID:  containerd://bf82f1a3e46a29dc6507a7217f5a5fc33b4ee471d9cc09ec1e680a1e8e2fd60a
    Image:         busybox
    Image ID:      docker.io/library/busybox@sha256:6915be4043561d64e0ab0f8f098dc2ac48e077fe23f488ac24b665166898115a
    Port:          <none>
    Host Port:     <none>
    Command:
      sleep
      1000000
    State:          Running
      Started:      Mon, 09 Mar 2020 14:38:12 -0700
    Ready:          True
    Restart Count:  0
    Environment:    <none>
    Mounts:
      /data from my-csi-volume (rw)
      /var/run/secrets/kubernetes.io/serviceaccount from default-token-46lvh (ro)
Conditions:
  Type              Status
  Initialized       True 
  Ready             True 
  ContainersReady   True 
  PodScheduled      True 
Volumes:
  my-csi-volume:
    Type:       PersistentVolumeClaim (a reference to a PersistentVolumeClaim in the same namespace)
    ClaimName:  csi-pvc
    ReadOnly:   false
  default-token-46lvh:
    Type:        Secret (a volume populated by a Secret)
    SecretName:  default-token-46lvh
    Optional:    false
QoS Class:       BestEffort
Node-Selectors:  <none>
Tolerations:     node.kubernetes.io/not-ready:NoExecute for 300s
                 node.kubernetes.io/unreachable:NoExecute for 300s
Events:
  Type    Reason                  Age   From                      Message
  ----    ------                  ----  ----                      -------
  Normal  Scheduled               106s  default-scheduler         Successfully assigned default/my-csi-app to csi-prow-worker
  Normal  SuccessfulAttachVolume  106s  attachdetach-controller   AttachVolume.Attach succeeded for volume "pvc-ad827273-8d08-430b-9d5a-e60e05a2bc3e"
  Normal  Pulling                 102s  kubelet, csi-prow-worker  Pulling image "busybox"
  Normal  Pulled                  99s   kubelet, csi-prow-worker  Successfully pulled image "busybox"
  Normal  Created                 99s   kubelet, csi-prow-worker  Created container my-frontend
  Normal  Started                 99s   kubelet, csi-prow-worker  Started container my-frontend
```

## Confirm BeeGFS driver works
The BeeGFS driver is configured to create new volumes under `/csi-data-dir` inside the beegfs container that is specified in the plugin StatefulSet found [here](../deploy/kubernetes-1.17/beegfs/csi-beegfs-plugin.yaml).  This path persist as long as the StatefulSet pod is up and running.

A file written in a properly mounted BeeGFS volume inside an application should show up inside the BeeGFS container.  The following steps confirms that BeeGFS is working properly.  First, create a file from the application pod as shown:

```shell
$ kubectl exec -it my-csi-app /bin/sh
/ # touch /data/hello-world
/ # exit
```

Next, ssh into the BeeGFS container and verify that the file shows up there:
```shell
$ kubectl exec -it $(kubectl get pods --selector app=csi-beegfsplugin -o jsonpath='{.items[*].metadata.name}') -c beegfs /bin/sh

```
Then, use the following command to locate the file. If everything works OK you should get a result similar to the following:

```shell
/ # find / -name hello-world
/var/lib/kubelet/pods/34bbb561-d240-4483-a56c-efcc6504518c/volumes/kubernetes.io~csi/pvc-ad827273-8d08-430b-9d5a-e60e05a2bc3e/mount/hello-world
/csi-data-dir/42bdc1e0-624e-11ea-beee-42d40678b2d1/hello-world
/ # exit
```

## Confirm the creation of the VolumeAttachment object
An additional way to ensure the driver is working properly is by inspecting the VolumeAttachment API object created that represents the attached volume:

```shell
$ kubectl describe volumeattachment
Name:         csi-5f182b564c52cd52e04e148a1feef00d470155e051924893d3aee8c3b26b8471
Namespace:    
Labels:       <none>
Annotations:  <none>
API Version:  storage.k8s.io/v1
Kind:         VolumeAttachment
Metadata:
  Creation Timestamp:  2020-03-09T21:38:05Z
  Resource Version:    10119
  Self Link:           /apis/storage.k8s.io/v1/volumeattachments/csi-5f182b564c52cd52e04e148a1feef00d470155e051924893d3aee8c3b26b8471
  UID:                 2d28d7e4-cda1-4ba9-a8fc-56fe081d71e9
Spec:
  Attacher:   beegfs.csi.netapp.com
  Node Name:  csi-prow-worker
  Source:
    Persistent Volume Name:  pvc-ad827273-8d08-430b-9d5a-e60e05a2bc3e
Status:
  Attached:  true
Events:      <none>
```

