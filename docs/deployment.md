# BeeGFS CSI Driver Deployment

## General Deployment
TODO: when we know how

## Kubernetes Node Preparation
The following packages MUST be installed on any Kubernetes node (master OR 
worker) that runs a component of the driver:
* beegfs-client-dkms
* beegfs-helperd (enabled and running under systemd)
* netstat

## Kubernetes Deployment
For a completely out-of-the-box deployment, verify you have kubectl access to a 
cluster and use "kubectl apply -k" (kustomize).

```bash
-> kubectl cluster-info
Kubernetes control plane is running at https://some.fqdn.or.ip:6443

-> kubectl apply -k deploy/prod
serviceaccount/csi-beegfs-controller-sa created
clusterrole.rbac.authorization.k8s.io/csi-beegfs-provisioner-role created
clusterrolebinding.rbac.authorization.k8s.io/csi-beegfs-provisioner-binding created
statefulset.apps/csi-beegfs-controller created
daemonset.apps/csi-beegfs-node created
csidriver.storage.k8s.io/beegfs.csi.netapp.com created
```

Verify all components installed and are operational.

```bash
-> kubectl get pods -n kube-system | grep csi-beegfs
csi-beegfs-controller-0                   2/2     Running   0          2m27s
csi-beegfs-node-2h6ff                     3/3     Running   0          2m27s
csi-beegfs-node-dkcr5                     3/3     Running   0          2m27s
csi-beegfs-node-ntcpc                     3/3     Running   0          2m27s
```

## Air-Gapped Kubernetes Deployment
You must either ensure all necessary images (see 
deploy/prod/kustomization.yaml for a complete list) are available on all nodes 
or ensure they can be pulled from some internal registry.

Either modify deploy/prod/kustomization.yaml or copy deploy/prod/ to a new 
directory (e.g. deploy/internal). Adjust the images[].newTag fields as 
necessary to ensure they either match images that exist on the Kubernetes nodes 
or reference your internal registry. Then follow the above commands for 
Kubernetes deployment.

## Example Application Deployment

Verify that a BeeGFS file system is accessible from the Kubernetes nodes. 
Minimally, verify that the BeeGFS servers are up and listening from a 
workstation that can access them:

```bash
-> beegfs-check-servers  # -c /path/to/config/file/if/necessary
Management
==========
mgmtd-node [ID: 1]: reachable at mgmtd.some.fdqn.or.ip:8008 (protocol: TCP)

Metadata
==========
meta-node [ID: 1]: reachable at meta.some.fqdn.or.ip:8005 (protocol: TCP)

Storage
==========
storage-node [ID: 1]: reachable at storage.some.fdqn.or.ip:8003 (protocol: TCP)
```

Modify *examples/csi-sc.yaml* such that *sysMgmtdHost* is set to an appropriate 
value (e.g. *mgmtd.some.fqdn.or.ip* in the output above).

From the project directory, deploy the application files found in the 
*examples/* directory, including a storage class, a PVC, and a pod which mounts 
a volume using the BeeGFS driver:

```bash
-> kubectl apply -f examples/csi-sc.yaml -f examples/csi-pvc.yaml -f examples/csi-app.yaml
storageclass.storage.k8s.io/csi-beegfs-sc created
persistentvolumeclaim/csi-beegfs-pvc created
pod/csi-beegfs-app created
```

Validate that the components deployed successfully:

```shell
-> kubectl get sc csi-beegfs-sc
NAME            PROVISIONER             RECLAIMPOLICY   VOLUMEBINDINGMODE   ALLOWVOLUMEEXPANSION   AGE
csi-beegfs-sc   beegfs.csi.netapp.com   Delete          Immediate           false                  61s


-> kubectl get pvc csi-beegfs-pvc
NAME             STATUS   VOLUME         CAPACITY   ACCESS MODES   STORAGECLASS    AGE
csi-beegfs-pvc   Bound    pvc-7621fab2   100Gi      RWX            csi-beegfs-sc   84s

-> kubectl get pod csi-beegfs-app
NAME             READY   STATUS    RESTARTS   AGE
csi-beegfs-app   1/1     Running   0          93s
```

Finally, inspect the application pod *csi-beegfs-app* which mounts a BeeGFS 
volume:

```bash
-> kubectl describe pod csi-beegfs-app
Name:         csi-beegfs-app
Namespace:    default
Priority:     0
Node:         node3/10.193.161.165
Start Time:   Tue, 05 Jan 2021 09:29:57 -0600
Labels:       <none>
Annotations:  cni.projectcalico.org/podIP: 10.233.92.101/32
              cni.projectcalico.org/podIPs: 10.233.92.101/32
Status:       Running
IP:           10.233.92.101
IPs:
  IP:  10.233.92.101
Containers:
  csi-beegfs-app:
    Container ID:  docker://21b4404e9791de597ac60043ebf7ddd24d3f684e4228874b81c4e307e017f862
    Image:         alpine:latest
    Image ID:      docker-pullable://alpine@sha256:3c7497bf0c7af93428242d6176e8f7905f2201d8fc5861f45be7a346b5f23436
    Port:          <none>
    Host Port:     <none>
    Command:
      ash
      -c
      touch "/data/touched-by-${POD_UUID}" && sleep 10000
    State:          Running
      Started:      Tue, 05 Jan 2021 09:30:00 -0600
    Ready:          True
    Restart Count:  0
    Environment:
      POD_UUID:   (v1:metadata.uid)
    Mounts:
      /data from csi-beegfs-volume (rw)
      /var/run/secrets/kubernetes.io/serviceaccount from default-token-bsr5d (ro)
Conditions:
  Type              Status
  Initialized       True 
  Ready             True 
  ContainersReady   True 
  PodScheduled      True 
Volumes:
  csi-beegfs-volume:
    Type:       PersistentVolumeClaim (a reference to a PersistentVolumeClaim in the same namespace)
    ClaimName:  csi-beegfs-pvc
    ReadOnly:   false
  default-token-bsr5d:
    Type:        Secret (a volume populated by a Secret)
    SecretName:  default-token-bsr5d
    Optional:    false
QoS Class:       BestEffort
Node-Selectors:  <none>
Tolerations:     node.kubernetes.io/not-ready:NoExecute op=Exists for 300s
                 node.kubernetes.io/unreachable:NoExecute op=Exists for 300s
Events:
  Type    Reason     Age   From               Message
  ----    ------     ----  ----               -------
  Normal  Scheduled  20s   default-scheduler  Successfully assigned default/csi-beegfs-app to node3
  Normal  Pulling    18s   kubelet            Pulling image "alpine:latest"
  Normal  Pulled     17s   kubelet            Successfully pulled image "alpine:latest" in 711.793082ms
  Normal  Created    17s   kubelet            Created container csi-beegfs-app
  Normal  Started    17s   kubelet            Started container csi-beegfs-app
```

*csi-beegfs-app* is configured to create a file inside its own */data* 
directory called *touched-by-<pod_uuid>*. Confirm that this file exists within 
the pod:

```bash
-> kubectl exec csi-beegfs-app -- ls /data
touched-by-9154aace-65a3-495d-8266-38eb0b564ddd
``` 

Kubernetes stages the BeeGFS file system in the /var/lib/kubelet/plugins 
directory and publishes the specific directory associated with the persistent 
volume to the pod in the /var/lib/kubelet/pods directory. Verify that the 
touched file exists in these locations if you have appropriate access to the 
worker node:

```bash
-> ssh root@10.193.161.165
-> ls /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-7621fab2/globalmount/mount/k8s-volumes/pvc-7621fab2/
touched-by-9154aace-65a3-495d-8266-38eb0b564ddd
-> ls /var/lib/kubelet/pods/9154aace-65a3-495d-8266-38eb0b564ddd/volumes/kubernetes.io~csi/pvc-7621fab2/mount/
touched-by-9154aace-65a3-495d-8266-38eb0b564ddd
```

Finally, verify that the file exists on the BeeGFS file system from a 
workstation that can access it.

```bash
ls /mnt/beegfs/k8s-volumes/pvc-7621fab2/
touched-by-9154aace-65a3-495d-8266-38eb0b564ddd
```

## General Configuration

The driver is ready to be used right out of the box, but many environments may
either require or benefit from additional configuration.

The driver loads a configuration file on startup which it uses as a template to
create the necessary configuration files to properly mount a BeeGFS file system.
A beegfs-client.conf file does NOT ship with the driver, so it applies the
values defined in its configuration file on top of the default
beegfs-client.conf that ships with each BeeGFS distribution. Each *config*
section may optionally contain parameters that override previous sections.

Depending on the topology of your cluster, some nodes MAY need different
configuration than others, so each node maintains its own unique copy of the
configuration file. For non-Kubernetes deployments, it is the administrator's
responsibility to distribute an appropriate file to each node. See [Kubernetes
Configuration](#kubernetes-configuration) for the Kubernetes-native way to
manage configuration within a Kubernetes cluster.

The *beegfsClientConf* section contains parameters taken directly out of a
beegfs-client.conf configuration file. In particular, the beegfs-client.conf 
file contains a number of references to other files (e.g. 
*connInterfacesFile*). The CSI configuration file instead expects a YAML list,
which it uses to generate the expected file. See [beegfsClientConf
Parameters](#beegfsclientconf-parameters) for more detail about supported
beegfsClientConf parameters.

The order of precedence for configuration option overrides is described by
"PRECEDENCE" comments in the example below. In general, precedence is as
follows: 
1. *fileSystemSpecificConfigs[i].config*. (A file system specific config is 
   mapped to its respective file system by the 
   *fileSystemSpecificConfigs[i].sysMgmtdHost*.)
1. The outermost *config*.
1. Locally installed BeeGFS configuration files: *beegfs-client.conf*, 
   *connInterfacesFile*, *connNetFilterFile*, *connTcpOnlyFilterFile*.

NOTE: All configuration, and in particular *fileSystemSpecificConfigs*
configuration is OPTIONAL! In many situations, only the outermost *config* is 
required.

```yaml
# when more specific configuration is not provided; PRECEDENCE 1 (lowest)
config:
  connInterfaces:
    - <interface_name>  # e.g. ib0
    - <interface_name>
  connNetFilter:
    - <ip_subnet>  # e.g. 10.0.0.1/24
    - <ip_subnet>
  connTcpOnlyFilter:
    - <ip_subnet>  # e.g. 10.0.0.1/24
    - <ip_subnet>
  beegfsClientConf:
    <beegfs-client.conf_key>: <beegfs-client.conf_value>  
    # e.g. connMgmtdPortTCP: 9008
    # SEE BELOW FOR RESTRICTIONS

fileSystemSpecificConfigs:
    # for a specific filesystem; PRECEDENCE 0 (highest)
  - sysMgmtdHost: <sysMgmtdHost>  # e.g. 10.10.10.1
    config:  # as above

    # for a specific filesystem; PRECEDENCE 0 (highest)
  - sysmMgmtdHost: <sysMgmtdHost>  # e.g. 10.10.10.1
    config:  # as above
    
```

For security purposes, the contents of BeeGFS connAuthFiles are stored in a
separate file. This file is optional, and should only be
used if the connAuthFile configuration option is used on a file system's
other services.

```yaml
- sysMgmtdHost: <sysMgmtdHost>  # e.g. 10.10.10.1
  connAuth: <some_secret_value>
- sysMgmtdHost: <sysMgmtdHost>
  connAuth: <some_secret_value>
```

## Kubernetes Configuration

When deployed into Kubernetes, a single Kubernetes ConfigMap contains the
configuration for all Kubernetes nodes. The ConfigMap includes the same
information as the configuration file above, with the addition of
the *nodeSpecificConfigs* sections. These more specific sections can override 
values specified (or not specified) in a more general section. When the driver 
starts up on a node, it uses the node's name to filter the global ConfigMap 
down to the node-specific configuration defined in 
[General Configuration](#general-configuration). In later versions,
[matchExpressions-based node label 
matching](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
may also be available.

The order of precedence for configuration option overrides is described by
"PRECEDENCE" comments in the example below. In general, precedence is as
follows: default < file system < node < file system AND node. When conflicts
occur between configurations of equal precedence, configuration set lower in the
file takes precedence over configuration set higher in the file.

NOTE: All configuration, and in particular *fileSystemSpecificConfigs* and
*nodeSpecificConfigs* configuration is OPTIONAL! In many situations, only the 
outermost *config* is required.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: beegfs-csi-config
data:
  beegfsCSI: |
    # when more specific configuration is not provided; PRECEDENCE 3 (lowest)
    config:
      connInterfaces:
        - <interface_name>  # e.g. ib0
        - <interface_name>
      connNetFilter:
        - <ip_subnet>  # e.g. 10.0.0.1/24
        - <ip_subnet>
      connTcpOnlyFilter:
        - <ip_subnet>  # e.g. 10.0.0.1/24
        - <ip_subnet>
      beegfsClientConf:
        <beegfs-client.conf_key>: <beegfs-client.conf_value>  
        # e.g. connMgmtdPortTCP: 9008
        # SEE BELOW FOR RESTRICTIONS
    
    fileSystemSpecificConfigs:  # OPTIONAL
        # for a specific filesystem; PRECEDENCE 2
      - sysMgmtdHost: <sysMgmtdHost>  # e.g. 10.10.10.1
        config:  # as above

        # for a specific filesystem; PRECEDENCE 2
      - sysMgmtdHost: <sysMgmtdHost>  # e.g. 10.10.10.1
        config:  # as above
  
    nodeSpecificConfigs:  # OPTIONAL
      - nodeList:
          - <node_name>  # e.g. node1
          - <node_name>
        # matchExpressions:  may be supported in >v1.0
        # default for a specific set of nodes; PRECEDENCE 1
        config:  # as above:
        # for a specific node AND filesystem; PRECEDENCE 0 (highest)
        fileSystemSpecificConfigs:  # as above

      - nodeList:
          - <node_name>  # e.g. node1
          - <node_name>
        # matchExpressions:  may be supported in >v1.0
        # default for a specific set of nodes; PRECEDENCE 1
        config:  # as above:
        # for a specific node AND filesystem; PRECEDENCE 0 (highest)
        fileSystemSpecificConfigs:  # as above
```

For security purposes, the contents of BeeGFS connAuthFiles are stored in a
separate Kubernetes Secret object. This file is optional, and should only be
deployed if the connAuthFile configuration option is used when configuring a
file system's other services.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: beegfs-csi-secret
data:
  connAuths: |
    - sysMgmtdHost: <sysMgmtdHost>  # e.g. 10.10.10.1
      connAuth: <some_secret_value>
    - sysMgmtdHost: <sysMgmtdHost>
      connAuth: <some_secret_value>
```

## beeGFSClientConf Parameters

The following beegfs-client.conf parameters appear in the BeeGFS v7.2
[beegfs-client.conf
file](https://git.beegfs.io/pub/v7/-/blob/7.2/client_module/build/dist/etc/beegfs-client.conf).
Other parameters may exist for newer or older BeeGFS versions. The list a
parameter falls under determines its level of support in the driver.

### No Effect

These parameters are specified elsewhere (a Kubernetes StorageClass, etc.) and
have no effect when specified in the beeGFSClientConf configuration section.

* sysMgmtdHost (specified in a *fileSystemSpecificConfigs[i]* or by the volume
  definition itself)
* connClientPortUDP (semi-random to allow multiple filesystem mounts)

### Unsupported

These parameters are specified elsewhere and may exhibit undocumented behavior
if specified here.

* connAuthFile
* connInterfacesFile
* connNetFilterFile
* connTcpOnlyFilterFile

### Untested

These parameters SHOULD result in the desired effect but have not been tested.

* connHelperdPortTCP
* connMgmtdPortTCP
* connMgmtdPortUDP
* connPortShift
* connCommRetrySecs
* connFallbackExpirationSecs
* connMaxInternodeNum
* connMaxConcurrentAttempts
* connUseRDMA
* connRDMABufNum
* connRDMABufSize
* connRDMATypeOfService
* logClientID 
* logHelperdIP
* logLevel
* logType
* quotaEnabled
* sysCreateHardlinksAsSymlinks
* sysMountSanityCheckMS
* sysSessionCheckOnClose
* sysSyncOnClose
* sysTargetOfflineTimeoutSecs
* sysUpdateTargetStatesSecs
* sysXAttrsEnabled
* tuneFileCacheType
* tunePreferredMetaFile
* tunePreferredStorageFile
* tuneRemoteFSync
* tuneUseGlobalAppendLocks
* tuneUseGlobalFileLocks
* sysACLsEnabled
