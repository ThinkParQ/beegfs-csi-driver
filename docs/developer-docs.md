# BeeGFS CSI Driver Developer Documentation

## Contents
* [Overview](#overview)
* [Building the Project](#building-the-project)
* [Developer Kubernetes Deployment](#developer-kubernetes-deployment)
* [Style Guidelines](#style-guidelines)
  * [YAML Files](#style-guidelines-yaml)
* [Frequently Asked Questions](#frequently-asked-questions)

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
*/deploy/k8s/overlays/dev/* and edit it as necessary. This specific path is 
.gitignored for convenience, so your local changes won't be (and shouldn't be) 
included in Git commits. For example:
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

## Style Guidelines
<a name="style-guidelines"></a>

### YAML Files
<a name="style-guidelines-yaml"></a>

YAML files in this project are formatted according to the following 
rules/restrictions:

* Strings are only quoted when required (when a correct YAML parser would 
  otherwise interpret an intended string as another type or when character 
  escaping necessitates).
  * `hello world` should NOT be quoted (obviously a string).
  * `127.0.0.1` should NOT be quoted (can only be interpreted as a string).
  * `8000` SHOULD be quoted where a string is expected and should NOT be quoted 
    where an integer is expected.
  * `true` SHOULD be quoted where a string is expected and should NOT be quoted 
    where a boolean is expected.
* If it is necessary to quote a string, double quotes `"` are preferred.

## Frequently Asked Questionsc
<a name="style-guidelines"></a>

### Why do we use sigs.k8s.io/yaml instead of gopkg.in/yaml?

See the answer to, "Why do we use JSON tags instead of YAML tags on our Go 
structs?"

### Why do we use JSON tags instead of YAML tags on our Go structs?

Our driver configuration file is written in YAML and all of our configuration
types originally had YAML tags to facilitate unmarshalling. In v1.0.0 we used 
gopkg.in/yaml and YAML tags to read configuration.

When we added the operator to the project, the configuration types were moved to
the `operator/` directory and incorporated into the Custom Resource Definition 
(CRD). The controller runtime built into the operator unmarshalls the CRD into
our configuration structs and passes those structs to our logic on each 
reconcile. The controller runtime uses sigs.k8s.io/yaml under the hood, and 
sigs.k8s.io/yaml uses JSON tags instead of YAML tags.

Kubernetes projects (like the controller runtime) use sigs.k8s.io/yaml
exclusively. It is advertised as a "better way of handling YAML", and the basic
premise is that it first converts YAML to JSON and then unmarshalls the JSON
using JSON tags. One obvious benefit is that a single package can easily handle
both YAML and JSON in an identical fashion. This is great for Kubernetes and
Kubernetes projects, which generally accept both YAML and JSON as configuration
methods. Even if a Kubernetes project wanted to use gopkg.in/yaml, it would be
very difficult, as the Kubernetes API structures
(e.g., Pods) only have JSON tags, which gopkg.in/yaml cannot interpret.

Rather than maintain two sets of identical tags and two separate unmarshalling
packages (with their own individual quirks), we decided to maintain only JSON
tags and use sigs.k8s.io/yaml exclusively moving forward.

### Why must beegfsClientConf values be strings in the driver configuration file?

The purpose of the beegfsClientConf field is to provide a slice of key value 
pairs that we can write into a beegfs-client.conf file on a node. That 
beegfs-client.conf file is then used to mount a BeeGFS file system. We 
do NOT want to maintain a set of accepted keys and values or to do any major
validation on these keys and values. This would potentially tie us to a 
particular BeeGFS version and/or introduce significant additional testing. We 
leave it to the administrator to make sure they only specify keys and values 
that are valid for their particular BeeGFS version and environment.

One challenge is that the beegfsClientConf field is part of our public API. 
Before the operator was introduced, this API was really just a description of a 
YAML file with no strict typing, and v1.0.0 of the driver actually accepted any 
type of beegfsClientConf value (although all values were converted to strings 
internally). When we added the operator to the project, the configuration types 
were moved to the `operator/` directory and incorporated into the Custom 
Resource Definition (CRD). The CRD is strongly typed, which forced us to choose 
a single type for beegfsClientConf values (the keys must obviously be strings). 

The only type that can represent all potential values in the beegfs-client.conf 
file (strings, integers, and booleans) is string, so we require string values. 
Because values like `8000` and `true` are interpreted as integers and booleans, 
respectively, by YAML parsers (including sigs.k8s.io/yaml), these values must 
be quoted to force a string interpretation. We maintain this requirement 
everywhere configuration including a beegfsClientConf field is unmarshalled for 
consistency.
