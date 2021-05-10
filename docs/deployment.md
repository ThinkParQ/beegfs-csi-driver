# BeeGFS CSI Driver Deployment

## Contents
<a name="contents"></a>

* [Deploying to Kubernetes](#deploying-to-kubernetes)
  * [Kubernetes Node Preparation](#kubernetes-node-preparation)
  * [Kubernetes Deployment](#kubernetes-deployment)
  * [Air-Gapped Kubernetes Deployment](#air-gapped-kubernetes-deployment)
* [Example Application Deployment](#example-application-deployment)
* [Managing BeeGFS Client Configuration](#managing-beegfs-client-configuration)
  * [General Configuration](#general-configuration)
  * [Kubernetes Configuration](#kubernetes-configuration)
  * [BeeGFS Client Parameters](#beegfs-client-parameters) 
* [Removing the Driver from Kubernetes](#removing-the-driver-from-kubernetes)

## Deploying to Kubernetes
<a name="deploying-to-kubernetes"></a>

### Kubernetes Node Preparation
<a name="kubernetes-node-preparation"></a>
The following MUST be completed on any Kubernetes node (master OR worker) that
runs a component of the driver:
* Download the [BeeGFS repository
  file](https://doc.beegfs.io/latest/advanced_topics/manual_installation.html)
  for your Linux distribution to the appropriate directory.
* Using the package manager for your Linux distribution (i.e. yum, apt, zypper)
  install the following packages: 
  * beegfs-client-dkms
  * beegfs-helperd
  * beegfs-utils  
* Start and enable beegfs-helperd using systemd: `systemctl start beegfs-helperd
  && systemctl enable beegfs-helperd`

IMPORTANT: By default the driver uses the beegfs-client.conf file at
*/etc/beegfs/beegfs-client.conf* for base configuration. Modifying the location
of this file is not currently supported without changing kustomization files. 

### Kubernetes Deployment
<a name="kubernetes-deployment"></a>
Deployment manifests are provided in this repository under *deploy/* along with
a default BeeGFS Client ConfigMap. The driver is deployed using `kubectl apply
-k` (kustomize). 

Steps:
* On a machine with kubectl and access to the Kubernetes cluster where you want
  to deploy the BeeGFS CSI driver clone this repository: `git clone
  https://github.com/NetApp/beegfs-csi-driver.git`
* If you wish to modify the default BeeGFS client configuration fill in the
  empty ConfigMap at *deploy/prod/csi-beegfs-config.yaml*.
  * An example ConfigMap is provided at
    *deploy/prod/csi-beegfs-config-example.yaml*. Please see the section on
    [Managing BeeGFS Client
    Configuration](#managing-beegfs-client-configuration) for full details. 
* If you are using [BeeGFS Connection Based Authentication](https://doc.beegfs.io/latest/advanced_topics/authentication.html) 
  fill in the empty Secret config file at *deploy/prod/csi-beegfs-connauth.yaml*.
  * An example Secret config file is provided at 
  *deploy/prod/csi-beegfs-connauth-example.yaml*. Please see the section on 
  [ConnAuth Configuration](#connauth-configuration) for full details. 
* Change to the BeeGFS CSI driver directory (`cd beegfs-csi-driver`) and run:
  `kubectl apply -k deploy/prod`
  * Note by default the beegfs-csi-driver image will be pulled from
    [DockerHub](https://hub.docker.com/r/netapp/beegfs-csi-driver). See [this
    section](#air-gapped-kubernetes-deployment) for guidance deploying in
    offline environments.
  * Note that some supported Kubernetes versions may require modified 
    deployment manifests. Use a version specific overlay from the *deploy/prod* 
    directory as necessary (e.g. `kubectl apply -k deploy/prod-1.18`).
* Verify all components are installed and operational: `kubectl get pods -n
  kube-system | grep csi-beegfs`

Example command outputs: 

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

-> kubectl get pods -n kube-system | grep csi-beegfs
csi-beegfs-controller-0                   2/2     Running   0          2m27s
csi-beegfs-node-2h6ff                     3/3     Running   0          2m27s
csi-beegfs-node-dkcr5                     3/3     Running   0          2m27s
csi-beegfs-node-ntcpc                     3/3     Running   0          2m27s
```
Note the `csi-beegfs-node-#####` pods are part of a DaemonSet, so the number you
see will correspond to the number of Kubernetes worker nodes in your
environment.

Next Steps:
* If you want a quick example on how to get started with the driver see the
  [Example Application Deployment](#example-application-deployment) section. 
* For a comprehensive introduction see the [BeeGFS CSI Driver Usage](usage.md)
  documentation.

### Air-Gapped Kubernetes Deployment
<a name="air-gapped-kubernetes-deployment"></a>
This section provides guidance on deploying the BeeGFS CSI driver in
environments where Kubernetes nodes do not have internet access. 

Deploying the CSI driver involves pulling multiple Docker images from various
Docker registries. You must either ensure all necessary images (see
*deploy/prod/kustomization.yaml* for a complete list) are available on all
nodes, or ensure they can be pulled from some internal registry.

If your air-gapped environment does not have a DockerHub mirror, one option is
pulling the necessary images from a machine with access to the internet
(example: `docker pull netapp/beegfs-csi-driver`) then save them as tar files
with [docker save](https://docs.docker.com/engine/reference/commandline/save/)
so they can be copied to the air-gapped Kubernetes nodes and loaded with [docker
load](https://docs.docker.com/engine/reference/commandline/load/).

Once the images are available, either modify *deploy/prod/kustomization.yaml* or
copy *deploy/prod/* to a new directory (e.g. *deploy/internal*). Adjust the
`images[].newTag` fields as necessary to ensure they either match images that
exist on the Kubernetes nodes or reference your internal registry. Then follow
the above commands for Kubernetes deployment.

## Example Application Deployment
<a name="example-application-deployment"></a>
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

Modify *examples/dyn/dyn-sc.yaml* such that `parameters`:
- `sysMgmtdHost` is set to an appropriate value (e.g. `mgmtd.some.fqdn.or.ip` in
  the output above)
- `volDirBasePath` contains a k8s cluster `name` that is unique to this k8s
  cluster among all k8s clusters accessing this BeeGFS file system.

From the project directory, deploy the application files found in the
*examples/dyn/* directory, including a Storage Class, a PVC, and a pod which
mounts a volume using the BeeGFS CSI driver:

```bash
-> kubectl apply -f examples/dyn
storageclass.storage.k8s.io/csi-beegfs-dyn-sc created
persistentvolumeclaim/csi-beegfs-dyn-pvc created
pod/csi-beegfs-dyn-app created
```

Validate that the components deployed successfully:

```shell
-> kubectl get sc csi-beegfs-dyn-sc
NAME                PROVISIONER             RECLAIMPOLICY   VOLUMEBINDINGMODE   ALLOWVOLUMEEXPANSION   AGE
csi-beegfs-dyn-sc   beegfs.csi.netapp.com   Delete          Immediate           false                  61s


-> kubectl get pvc csi-beegfs-dyn-pvc
NAME                 STATUS   VOLUME         CAPACITY   ACCESS MODES   STORAGECLASS        AGE
csi-beegfs-dyn-pvc   Bound    pvc-7621fab2   100Gi      RWX            csi-beegfs-dyn-sc   84s

-> kubectl get pod csi-beegfs-dyn-app
NAME                 READY   STATUS    RESTARTS   AGE
csi-beegfs-dyn-app   1/1     Running   0          93s
```

Finally, inspect the application pod `csi-beegfs-dyn-app` which mounts a BeeGFS
volume:

```bash
-> kubectl describe pod csi-beegfs-dyn-app
Name:         csi-beegfs-dyn-app
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
      touch "/mnt/dyn/touched-by-${POD_UUID}" && sleep 7d
    State:          Running
      Started:      Tue, 05 Jan 2021 09:30:00 -0600
    Ready:          True
    Restart Count:  0
    Environment:
      POD_UUID:   (v1:metadata.uid)
    Mounts:
      /mnt/dyn from csi-beegfs-dyn-volume (rw)
      /var/run/secrets/kubernetes.io/serviceaccount from default-token-bsr5d (ro)
Conditions:
  Type              Status
  Initialized       True 
  Ready             True 
  ContainersReady   True 
  PodScheduled      True 
Volumes:
  csi-beegfs-dyn-volume:
    Type:       PersistentVolumeClaim (a reference to a PersistentVolumeClaim in the same namespace)
    ClaimName:  csi-beegfs-dyn-pvc
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
  Normal  Scheduled  20s   default-scheduler  Successfully assigned default/csi-beegfs-dyn-app to node3
  Normal  Pulling    18s   kubelet            Pulling image "alpine:latest"
  Normal  Pulled     17s   kubelet            Successfully pulled image "alpine:latest" in 711.793082ms
  Normal  Created    17s   kubelet            Created container csi-beegfs-dyn-app
  Normal  Started    17s   kubelet            Started container csi-beegfs-dyn-app
```

`csi-beegfs-dyn-app` is configured to create a file inside its own */mnt/dyn*
directory called *touched-by-<pod_uuid>*. Confirm that this file exists within
the pod:

```bash
-> kubectl exec csi-beegfs-dyn-app -- ls /mnt/dyn
touched-by-9154aace-65a3-495d-8266-38eb0b564ddd
``` 

Kubernetes stages the BeeGFS file system in the */var/lib/kubelet/plugins*
directory and publishes the specific directory associated with the persistent
volume to the pod in the */var/lib/kubelet/pods* directory. Verify that the
touched file exists in these locations if you have appropriate access to the
worker node:

```bash
-> ssh root@10.193.161.165
-> ls /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-7621fab2/globalmount/mount/k8s/my-cluster-name/pvc-7621fab2/
touched-by-9154aace-65a3-495d-8266-38eb0b564ddd
-> ls /var/lib/kubelet/pods/9154aace-65a3-495d-8266-38eb0b564ddd/volumes/kubernetes.io~csi/pvc-7621fab2/mount/
touched-by-9154aace-65a3-495d-8266-38eb0b564ddd
```

Finally, verify that the file exists on the BeeGFS file system from a
workstation that can access it.

```bash
ls /mnt/beegfs/k8s/my-cluster-name/pvc-7621fab2/
touched-by-9154aace-65a3-495d-8266-38eb0b564ddd
```

Next Steps:
* For a comprehensive introduction see the [BeeGFS CSI Driver Usage](usage.md)
  documentation.
* For additional examples see *examples/README.md*. 

## Managing BeeGFS Client Configuration
<a name="managing-beegfs-client-configuration"></a>

Currently the only tested and supported container orchestrator (CO) for the
BeeGFS CSI driver is Kubernetes. Notes in the General Configuration section
below would apply to other COs if supported. For Kubernetes the preferred method
to apply desired BeeGFS Client configuration is using a Kubernetes ConfigMap and
Secret, as described in [Kubernetes Configuration](#kubernetes-configuration).

### General Configuration
<a name="general-configuration"></a>

The driver is ready to be used right out of the box, but many environments may
either require or benefit from additional configuration.

The driver loads a configuration file on startup which it uses as a template to
create the necessary configuration files to properly mount a BeeGFS file system.
A beegfs-client.conf file does NOT ship with the driver, so it applies the
values defined in its configuration file on top of the default
beegfs-client.conf that ships with each BeeGFS distribution. Each `config`
section may optionally contain parameters that override previous sections.

NOTE: The configuration file is specified by the `--config-path` command line 
argument. For Kubernetes, the deployment manifests handle this automatically.

Depending on the topology of your cluster, some nodes MAY need different
configuration than others. This requirement can be handled in one of two ways:
1. The administrator creates unique configuration files and deploys each to the
   proper node.
2. The administrator creates one global configuration file with a
   `nodeSpecificConfigs` section and specifies the `--node-id` CLI flag on each
   node when starting the driver.
     * Note: Deploying to Kubernetes using the provided manifests in deploy/
       handles setting this flag and deploying a ConfigMap representing this
       global configuration file.  

Kubernetes deployment greatly simplifies the distribution of a global
configuration file using the second approach and is the de-facto standard way to
deploy the driver. See [Kubernetes Configuration](#kubernetes-configuration) for
details.

In the example below, the `beegfsClientConf` section contains parameters taken
directly out of a beegfs-client.conf configuration file. In particular, the
beegfs-client.conf file contains a number of references to other files (e.g.
`connInterfacesFile`). The CSI configuration file instead expects a YAML list,
which it uses to generate the expected file. See [BeeGFS Client
Parameters](#beegfs-client-parameters) for more detail about
supported `beegfsClientConf` parameters.

The order of precedence for configuration option overrides is described by
"PRECEDENCE" comments in the example below. In general, precedence is as
follows: 

>config < fileSystemSpecificConfig < nodeSpecificConfig.config <
nodeSpecificConfig.fileSystemSpecificConfig 

When conflicts occur between configurations of equal precedence, configuration
set lower in the file takes precedence over configuration set higher in the
file.

NOTE: All configuration, and in particular `fileSystemSpecificConfigs` and
`nodeSpecificConfigs` configuration is OPTIONAL! In many situations, only the
outermost `config` is required.

```yaml
# when more specific configuration is not provided; PRECEDENCE 3 (lowest)
config:  # OPTIONAL
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
  - sysMgmtdHost: <sysMgmtdHost>  # e.g. 10.10.10.100
    config:  # as above

nodeSpecificConfigs:  # OPTIONAL
  - nodeList:
      - <node_name>  # e.g. node1
      - <node_name>
    # default for a specific set of nodes; PRECEDENCE 1
    config:  # as above:
    # for a specific node AND filesystem; PRECEDENCE 0 (highest)
    fileSystemSpecificConfigs:  # as above

  - nodeList:
      - <node_name>  # e.g. node1
      - <node_name>
    # default for a specific set of nodes; PRECEDENCE 1
    config:  # as above:
    # for a specific node AND filesystem; PRECEDENCE 0 (highest)
    fileSystemSpecificConfigs:  # as above
```

#### ConnAuth Configuration
<a name="connauth-configuration"></a>
For security purposes, the contents of BeeGFS connAuthFiles are stored in a
separate file. This file is optional, and should only be used if the
connAuthFile configuration option is used on a file system's other services.

NOTE: The connAuth configuration file is specified by the `--connauth-path`
command line argument. For Kubernetes, the deployment manifests handle this
automatically.

```yaml
- sysMgmtdHost: <sysMgmtdHost>  # e.g. 10.10.10.1
  connAuth: <some_secret_value>
- sysMgmtdHost: <sysMgmtdHost>  # e.g. 10.10.10.100
  connAuth: <some_secret_value>
```

NOTE: Unlike general configuration, connAuth configuration is only applied at a 
per file system level. There is no default connAuth and the concept of a node 
specific connAuth doesn't make sense.

### Kubernetes Configuration
<a name="kubernetes-configuration"></a>
When deployed into Kubernetes, a single Kubernetes ConfigMap contains the
configuration for all Kubernetes nodes. When the driver starts up on a node, it
uses the node's name to filter the global ConfigMap down to a node-specific
version. 

The instructions in the [Kubernetes Deployment](#kubernetes-deployment)
automatically create an empty ConfigMap and an empty Secret and pass them to the
driver on all nodes.

* To pass custom configuration to the driver, add the desired parameters from
[General Configuration](#general-configuration) to
  *deploy/prod/csi-beegfs-config.yaml* (or another overlay) before deploying. 
  The resulting deployment will automatically include a correctly formed 
  ConfigMap. See *deploy/prod/csi-beegfs-config-example.yaml* for an example 
  file.
* To pass connAuth configuration to the driver, modify
  *deploy/prod/csi-beegfs-connauth.yaml* (or another overlay) before deploying. 
  The resulting deployment will automatically include a correctly formed
  Secret. See *deploy/prod/csi-beegfs-config-connauth-example.yaml* for an 
  example file.

To update configuration after initial deployment, modify
*deploy/prod/csi-beegfs-config.yaml* or *deploy/prod/csi-beegfs-connauth.yaml*
and repeat the kubectl deployment step from [Kubernetes Deployment](#kubernetes-deployment). 
Kustomize will automatically update all components and restart the driver on 
all nodes so that it picks up the latest changes.

NOTE: To validate the BeeGFS Client configuration file used for a specific PVC, 
see the [Troubleshooting Guide](troubleshooting.md#k8s-determining-the-beegfs-client-conf-for-a-pvc)

### BeeGFS Client Parameters (beegfsClientConf)
<a name="beegfs-client-parameters"></a>
The following beegfs-client.conf parameters appear in the BeeGFS v7.2
[beegfs-client.conf
file](https://git.beegfs.io/pub/v7/-/blob/7.2/client_module/build/dist/etc/beegfs-client.conf).
Other parameters may exist for newer or older BeeGFS versions. The list a
parameter falls under determines its level of support in the driver.

#### No Effect
<a name="no-effect"></a>
These parameters are specified elsewhere (a Kubernetes StorageClass, etc.) or
are determined dynamically and have no effect when specified in the
`beeGFSClientConf` configuration section.

* `sysMgmtdHost` (This is specified in a `fileSystemSpecificConfigs[i]` or by
  the volume definition itself.)
* `connClientPortUDP` (An ephemeral port, obtained by binding to port 0, allows
  multiple filesystem mounts. On Linux the selected ephemeral port is
  constrained by the values of [IP
  variables](https://www.kernel.org/doc/html/latest/networking/ip-sysctl.html#ip-variables).
  [Ensure that firewalls allow UDP
  traffic](https://doc.beegfs.io/latest/advanced_topics/network_tuning.html#firewalls-network-address-translation-nat)
  between BeeGFS file system nodes and ephemeral ports on BeeGFS CSI Driver
  nodes.)
* `connPortShift`

#### Unsupported
<a name="unsupported"></a>
These parameters are specified elsewhere and may exhibit undocumented behavior
if specified here.

* `connAuthFile` - Overridden in the 
  [connAuth configuration file](#connauth-configuration).
* `connInterfacesFile` - Overridden by lists in the 
  [driver configuration file](#general-configuration).
* `connNetFilterFile` - Overridden by lists in the driver configuration file.
* `connTcpOnlyFilterFile` - Overridden by lists in the driver configuration 
  file.

### Tested
<a name="tested"></a>
These parameters have been tested and verified to have the desired effect.

* `quotaEnabled` - Documented in [Quotas](quotas.md).

#### Untested
<a name="untested"></a>
These parameters SHOULD result in the desired effect but have not been tested.

* `connHelperdPortTCP`
* `connMgmtdPortTCP`
* `connMgmtdPortUDP`
* `connPortShift`
* `connCommRetrySecs`
* `connFallbackExpirationSecs`
* `connMaxInternodeNum`
* `connMaxConcurrentAttempts`
* `connUseRDMA`
* `connRDMABufNum`
* `connRDMABufSize`
* `connRDMATypeOfService`
* `logClientID`
* `logHelperdIP`
* `logLevel`
* `logType`
* `sysCreateHardlinksAsSymlinks`
* `sysMountSanityCheckMS`
* `sysSessionCheckOnClose`
* `sysSyncOnClose`
* `sysTargetOfflineTimeoutSecs`
* `sysUpdateTargetStatesSecs`
* `sysXAttrsEnabled`
* `tuneFileCacheType`
* `tunePreferredMetaFile`
* `tunePreferredStorageFile`
* `tuneRemoteFSync`
* `tuneUseGlobalAppendLocks`
* `tuneUseGlobalFileLocks`
* `sysACLsEnabled`

## Removing the Driver from Kubernetes
<a name="removing-the-driver-from-kubernetes"></a>
If you're experiencing any issues, find functionality lacking, or our
documentation is unclear, we'd appreciate if you let us know:
https://github.com/NetApp/beegfs-csi-driver/issues. 

The driver can be removed using `kubectl delete -k` (kustomize) and the original
deployment manifests. On a machine with kubectl and access to the Kubernetes
cluster you want to remove the BeeGFS CSI driver from: 

* Remove any Storage Classes, Persistent Volumes, and Persistent Volume Claims
  associated with the driver. 
* Set the working directory to the beegfs-csi-driver repository containing the
  manifests used to deploy the driver.
* Run the following command to remove the driver: `kubectl delete -k
  deploy/prod`
