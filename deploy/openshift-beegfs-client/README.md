# BeeGFS Client Deployment for OpenShift

<a name="contents"></a>
## Contents

* [What Is This For?](#what-is-this-for)
* [Requirements](#requirements)
* [Steps](#steps)
* [How Does It Work?](#how-does-it-work)
* [Is It Tested?](#is-it-tested)
* [What Is Next?](#what-is-next)

<a name="what-is-this-for"></a>
## What Is This For?

The BeeGFS CSI driver typically requires all nodes to have BeeGFS pre-installed and pre-enabled. It relies on a inserted
kernel module and several user space utilities to operate correctly. The default OS for nodes in an OpenShift cluster is
RedHat CoreOS (RHCOS). While the atomic nature of RHCOS is beneficial in many ways, it makes installing packages and
inserting kernel modules by typical means impossible. These deployment manifests allow a set of containers to prep all
nodes for the deployment of the BeeGFS CSI driver by:

1. Building and inserting a kernel module in each node, and
2. Putting user space files and utilities in a place on each node where the BeeGFS CSI driver can access them.

The BeeGFS CSI driver already supports RHEL nodes in OpenShift clusters by traditional means. These manifests are
intended for use  in OpenShift clusters in which BeeGFS workloads must run on RHCOS nodes.

THESE MANIFESTS ARE CURRENTLY EXPERIMENTAL AND NOT RECOMMENDED FOR PRODUCTION USE.

<a name="requirements"></a>
## Requirements

* Administrative access to an OpenShift 4.9+ cluster with RHCOS nodes (via the oc CLI tool).
* If the cluster also includes RHEL nodes, BeeGFS should NOT be preinstalled on these nodes (because this process preps
  BeeGFS on both RHCOS and RHEL nodes).
* If the cluster also includes RHEL nodes, these nodes must be entitled (with Subscription Manager).
* The OpenShift internal image registry is enabled and functioning correctly: 
    *  `oc get configs.imageregistry.operator.openshift.io cluster -ojsonpath='{.spec.managementState}{"\n"}'` returns
       "Managed"
    *  `oc get pod -n openshift-image-registry` returns at least one "image-registry-##########-#####" Pod.

<a name="steps"></a>
## Steps

1.  Navigate to the directory containing this README and the `beegfs-client-*.yaml` deployment manifests.
1.  Apply the BuildConfig (and related objects) and verify that build completes successfully.
    ```
    -> oc create namespace beegfs-csi 
    namespace/beegfs-csi created  # This namespace may already exist if the BeeGFS CSI driver has been deployed.

    -> oc apply -f beegfs-client-bc.yaml
    imagestream.image.openshift.io/beegfs-client created
    configmap/beegfs-client-poststart created
    buildconfig.build.openshift.io/beegfs-client created

    -> oc get build -n beegfs-csi
    NAME              TYPE     FROM         STATUS     STARTED         DURATION
    beegfs-client-1   Docker   Dockerfile   Complete   6 minutes ago   1m18s

    -> oc get is -n beegfs-csi
    NAME            IMAGE REPOSITORY                                                            TAGS     UPDATED
    beegfs-client   image-registry.openshift-image-registry.svc:5000/beegfs-csi/beegfs-client   latest   6 minutes ago
    ```
1.  Apply the DaemonSet (and related objects) and verify that the beegfs-client Pod comes up on all nodes.
    ```
    -> oc apply -f beegfs-client-ds.yaml 
    serviceaccount/beegfs-client created
    role.rbac.authorization.k8s.io/beegfs-client created
    rolebinding.rbac.authorization.k8s.io/beegfs-client created
    daemonset.apps/beegfs-client created

    -> oc get ds -n beegfs-csi
    NAME            DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
    beegfs-client   8         8         8       8            8           <none>          11m
    ```
1. Deploy the BeeGFS CSI operator and driver as normal from the OpenShift GUI. When defining the BeegfsDriver spec, 
   delete the default affinity rules that prevent the driver from installing on RHCOS nodes.

<a name="how-does-it-work"></a>
## How Does It Work?

The BeeGFS client essentially consists of the source code for a kernel module and scripts/systemd units that are
responsible for building and inserting the module into the running kernel. In order to build the module, the included
scripts must have access to the kernel headers for the particular version of the kernel that is running. Each time
OpenShift is updated, certain ImageStreamTags are also updated to point to appropriate versions of a number of useful
container images. One of these images, driver-toolkit, contains up-to-date kernel headers for the version of the kernel
that is now running on all RHCOS nodes in the cluster.

On install:

1. A Dockerfile in the new beegfs-client BuildConfig takes the driver-toolkit ImageStreamTag as input. It pulls BeeGFS
   packages onto the driver-image, builds the BeeGFS kernel module, enables necessary BeeGFS services (the driver
   toolkit image includes systemd), and outputs the result to the beegfs-client ImageStreamTag. The result is a
   ready-to-use beegfs-client container image accessible by all nodes in the cluster.
1. The new beegfs-client DaemonSet deploys a beegfs-client Pod to all nodes in the cluster. The Pod uses the previously
   built beegfs-client container image.
    * On RHCOS nodes: The BeeGFS kernel module is inserted into the kernel and is ready to go.
    * On RHEL nodes: The BeeGFS kernel module may need to be rebuilt, as RHEL node kernels are not kept in sync with
      RHCOS nodes in the cluster. A poststart script attempts to pull in the correct version of the kernel-headers,
      rebuild the kernel module, and restart the BeeGFS services. This is only possible if the underlying node is
      entitled.
    * On all nodes: A poststart script places user space files and utilities are somewhere on the host that the BeeGFS
      CSI driver can access them. These files would normally install to locations like `/opt/beegfs` and `/etc/beegfs`.
      However, to avoid "polluting" nodes, they are instead placed in
      `/var/lib/kubelet/plugins/beegfs.csi.netapp.com/client`, a directory that the BeeGFS CSI driver already has
      control over.
1. The BeeGFS CSI driver deploys as normal, locates the necessary files and utilities, and operates as expected.

On cluster upgrade:

1. The new beegfs-client BuildConfig detects the existence of an updated container image associated with the
   driver-toolkit ImageStreamTag. This updated image contains updated kernel headers. It rebuilds the beegfs-client
   container image and again outputs it to the beegfs-client ImageStreamTag.
1. Each node restarts as part of the cluster upgrade process. When a node comes back online and starts its beegfs-client
   pod, it uses the updated beegfs-client image (with a kernel module built against updated kernel-headers).

<a name="is-it-tested"></a>
## Is It Tested?

These manifests are used to deploy the BeeGFS client to OpenShift clusters in the BeeGFS CSI driver CI infrastructure
today. The driver runs effectively in that environment, including RDMA over Converged Ethernet (RoCE) functionality
(thanks to IB and RDMA packages included inbox in RHCOS).

Limited testing shows that the BeeGFS client deployed by these manifests rebuilds correctly when RHCOS updates or when
a new BeeGFS version is selected, but this testing is not rigorous. Unusual environments and/or upgrade paths or
upgrades to unreleased versions of OpenShift may break these manifests in unexpected ways.

<a name="what-is-next"></a>
## What Is Next?

These manifests are currently experimental and not intended for production use. Kernel module support in OpenShift is
evolving and a more general "automatic worker node prep" feature that can be utilized on a wider range of operating
systems and Kubernetes distributions is being considered. Potential improvements include:

* Automate the building and deployment of the beegfs-client container within the BeeGFS CSI driver operator (to 
  eliminate these manual deployment steps).
* Standardize around the [kmods-via-containers
  implementation](https://access.redhat.com/documentation/si-lk/openshift_container_platform/4.9/html/specialized_hardware_and_driver_enablement/driver-toolkit#create-simple-kmod-image_driver-toolkit)
  for kernel module build and insertion (these manifests use the standard BeeGFS systemd units and scripts).
* Ship a BeeGFS CSI driver container that includes necessary user space utilities and files (so these utilities and files
  do not need to be made available outside the client or driver containers).
