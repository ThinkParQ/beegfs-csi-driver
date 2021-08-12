# BeeGFS CSI Driver Developer Documentation

## Contents
* [Overview](#overview)
* [Building the Project](#building-the-project)
* [Developer Kubernetes Deployment](#developer-kubernetes-deployment)

## Overview 
This repository hosts the BeeGFS CSI Driver and all of its build and dependent configuration files to deploy the driver.

## Building the Project 

### Building the binaries
If you want to build the driver yourself, you can do so with the following command from the root directory:

```shell
make
```

### Building the containers

```shell
make container
```

### Building and pushing the containers

```shell
make push
```

Optionally set `REGISTRY_NAME` or `IMAGE_TAGS`:

```shell
# Prerequisite(s):
#   Change "docker.repo.eng.netapp.com/${USER}".
#   Change 'devBranchName-canary'.
#   $ docker login docker.repo.eng.netapp.com 
# REGISTRY_NAME and IMAGE_TAGS must be specified as make arguments.
# REGISTRY_NAME and IMAGE_TAGS cannot be pulled from the environment.
make REGISTRY_NAME="docker.repo.eng.netapp.com/${USER}" IMAGE_TAGS=devBranchName-canary push
```

## Developer Kubernetes Deployment
Create a new overlay by copying */deploy/k8s/overlays/default-dev/* to 
*/deploy/k8s/overlays/dev/* (for example) and edit it as necessary. All new 
directories in */deploy/k8s/overlays/* are .gitignored, so your 
local changes won't be (and shouldn't be) included in Git commits. For example:
* Change `images\[beegfs-csi-driver\].newTag` to whatever tag you are building 
  and pushing.
* Change `images\[].newName` to include whatever repo you can pull from.
* Change `namespace` to whatever makes sense.

You can also download/install kustomize and use "kustomize set ..." commands 
either from the command line or in a script to modify your deployment as 
necessary.

When you are ready to deploy, verify you have kubectl access to a 
cluster and use "kubectl apply -k" (kustomize).

```bash
-> kubectl cluster-info
Kubernetes control plane is running at https://some.fqdn.or.ip:6443

-> kubectl apply -k deploy/k8s/overlays/dev
serviceaccount/csi-beegfs-controller-sa created
clusterrole.rbac.authorization.k8s.io/csi-beegfs-provisioner-role created
clusterrolebinding.rbac.authorization.k8s.io/csi-beegfs-provisioner-binding created
configmap/csi-beegfs-config-57mtcc98f4 created
secret/csi-beegfs-connauth-m6k27kff96 created
statefulset.apps/csi-beegfs-controller created
daemonset.apps/csi-beegfs-node created
csidriver.storage.k8s.io/beegfs.csi.netapp.com created
```

Verify all components installed and are operational.

```bash
-> kubectl get pods
csi-beegfs-controller-0                   2/2     Running   0          2m27s
csi-beegfs-node-2h6ff                     3/3     Running   0          2m27s
csi-beegfs-node-dkcr5                     3/3     Running   0          2m27s
csi-beegfs-node-ntcpc                     3/3     Running   0          2m27s
csi-beegfs-socat-0                        0/1     Pending   0          17h
```