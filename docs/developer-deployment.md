## Kubernetes Deployment
Copy deploy/dev/kustomization-template.yaml to deploy/dev/kustomization.yaml 
and edit it as necessary. deploy/dev/kustomization.yaml is .gitignored, so your 
local changes won't be (and shouldn't be) included in Git commits. For example:
* Change images\[beegfs-csi-driver\].newTag to whatever tag you are building and 
  pushing.
* Change images\[].newName to include whatever repo you can pull from.
* Change namespace to whatever makes sense.

You can also download/install kustomize and use "kustomize set ..." commands 
either from the command line or in a script to modify your deployment as 
necessary.

When you are ready to deploy, verify you have kubectl access to a 
cluster and use "kubectl apply -k" (kustomize).

```bash
-> kubectl cluster-info
Kubernetes control plane is running at https://some.fqdn.or.ip:6443

-> kubectl apply -k deploy/dev
serviceaccount/csi-beegfs-controller-sa created
clusterrole.rbac.authorization.k8s.io/csi-beegfs-provisioner-role created
clusterrolebinding.rbac.authorization.k8s.io/csi-beegfs-provisioner-binding created
statefulset.apps/csi-beegfs-controller created
statefulset.apps/csi-beegfs-socat created
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