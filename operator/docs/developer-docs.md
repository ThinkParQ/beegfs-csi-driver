# BeeGFS CSI Driver Operator Developer Documentation <!-- omit in toc -->

## Contents <!-- omit in toc -->
<a name="contents"></a>

- [Overview](#overview)
- [Important Links](#important-links)
- [Development Environment](#development-environment)
- [Directory Structure](#directory-structure)
- [General Workflows](#general-workflows)
  - [Build and Test the Operator Controller Manager](#build-and-test-the-operator-controller-manager)
  - [Change the BeegfsDrivers API](#change-the-beegfsdrivers-api)
  - [Change the Behavior of the Controller](#change-the-behavior-of-the-controller)
  - [Change the Way OLM Displays or Installs the Operator](#change-the-way-olm-displays-or-installs-the-operator)
  - [Prepare Changes For a Pull Request](#prepare-changes-for-a-pull-request)
  - [Update the operator-sdk version](#update-the-operator-sdk-version)
- [Testing](#testing)
  - [Bundle Validation](#bundle-validation)
  - [Unit Testing](#unit-testing)
  - [Integration Testing with EnvTest](#integration-testing-with-envtest)
  - [Functional Testing](#functional-testing)
    - [Run Operator Locally Against Any Cluster](#run-operator-locally-against-any-cluster)
    - [Run Operator With OLM Integration Using kubectl](#run-operator-with-olm-integration-using-kubectl)
    - [Run Operator With OLM Integration Using OpenShift Console (Deprecated)](#run-operator-with-olm-integration-using-openshift-console-deprecated)
    - [Install Operator as if From OperatorHub in OpenShift Console (Deprecated)](#install-operator-as-if-from-operatorhub-in-openshift-console-deprecated)

## Overview
<a name="overview"></a>

The BeeGFS CSI driver operator was scaffolded using the Operator SDK and
continues to be maintained according to the operator pattern and Operator SDK
best practices. Operator SDK, in turn, generally wraps the Kubebuilder SDK for
creating Kubernetes controllers. This document does not aim to serve as a
replacement for operator, Operator SDK, or Kubebuilder documentation.

While operators can be written in any language, and many operators make use of
the Helm or Ansible integration provided by the Operator SDK, the BeeGFS CSI
driver operator is written in Go.

## Important Links
<a name="important-links"></a>

* [Introduction to the operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
* [Operator SDK home page](https://sdk.operatorframework.io/)
* [Operator SDK Go documentation](https://sdk.operatorframework.io/docs/building-operators/golang/)
* [Kubebuilder book](https://book.kubebuilder.io/)

## Development Environment
<a name="development-environment"></a>

The operator is built using a Makefile and many Make commands result in the
automatic download of necessary binaries (e.g., controller-gen, kustomize,
etc.). The following are not automatically downloaded and must be pre-installed
in a dev environment:
* [Golang (the version specified in *go.mod*)](https://golang.org/doc/install)
* [Operator SDK (currently v1.25.0)](https://sdk.operatorframework.io/docs/installation/)

## Directory Structure
<a name="directory-structure"></a>

All operator related code is maintained in the *operator* directory, which was
originally scaffolded with operator-sdk using the following commands:

```bash
operator-sdk init --domain csi.netapp.com --repo github.com/netapp/beegfs-csi-driver/operator --owner 'NetApp, Inc' --project-name beegfs-csi-driver-operator
operator-sdk create api --group beegfs --kind BeegfsDriver --version v1 --resource --controller
```

The following files and directories are included:

* *api/*
  * Originally scaffolded, but intended to be edited.
  * Contains the BeegfsDriver struct and all dependent structs.
  * Go code, kubebuilder markers, and operator-sdk markers here are largely
    responsible for generated code and manifests in other directories.
* *bin/*
  * .gitignored.
  * The default location for Make to dump downloaded tools.
* *bundle/*
  * Entirely generated via `make bundle`. DO NOT EDIT.
  * The bundle directory is what we submit to OperatorHub and Openshift. It
    enables installation of the operator using Operator Lifecyle Manager (OLM).
  * *bundle/manifests/beegfs.csi.netapp.com_beegfsdrivers.yaml* is the copy of 
    the CRD that is deployed by OLM on install.
  * *bundle/manifests/beegfs-csi-driver-operator.clusterserviceversion.yaml* is 
    the Cluster Service Version. It is the primary source of information OLM 
    uses to display information about and install the operator. If something 
    about the operator appears on OperatorHub or in Openshift, it is probably in
    this file somewhere.
* *config/*
  * Partially scaffolded, partially generated. Some files can be edited.
  * *config/crd/bases* 
    * Entirely generated via `make manifests`. DO NOT EDIT.
    * To change the CRD (which is generally built with Kustomize), either:
      * Edit structs and kubebuilder markers in *api/*, or
      * Add a patch in *config/crd/patches*.
  * *config/manifests* 
    * Partially generated via `make bundle`. Take care when editing.
    * To change the manifests (which directly results in a change to the
      bundle):
      * Edit structs and operator-sdk markers for
        [some fields](https://sdk.operatorframework.io/docs/olm-integration/generation/#csv-fields)
        in *api/*,
        * Edit
          *config/manifests/bases/beegfs-csi-driver-operator.clustersericeversion.yaml*
          for other fields, or
        * Add a patch in *config/crd/patches* or (potentially)
          *config/manifests/patches* (least preferred).
  * *config/samples*
    * Originally scaffolded, but intended to be edited.
    * Appears in the Cluster Service Version (and in OLM) as the 
      sample/template for the BeegfsDriver Custom Resource (CR).
    
* *controllers/*
  * Originally scaffolded, but intended to be edited.
  * Contains the code that makes the operator controller function.
* *docs/*
  * Not scaffolded and intended to be edited.
  * Contains operator-specific documentation.
* *hack/*
  * Originally scaffolded, but can be edited.
  * Contains miscellaneous files that may assist in development or testing,
    but which may be changed or removed at any time.
* *testbin/*
  * .gitignored
  * The default location for
    [EnvTest](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest)
    to dump downloaded tools.
* *Dockerfile*
  * Originally scaffolded, but intended to be edited.
  * Builds the container the operator controller manager runs in.
* *main.go*
  * Originally scaffolded, but intended to be edited.
  * Bootstraps the operator controller manager. The operator controller
      manager can theoretically run multiple controllers at once (one for
      every API in a group), but we only maintain the BeegfsDrivers API.

## General Workflows
<a name="general-workflows"></a>

### Build and Test the Operator Controller Manager
<a name="build-and-test-the-operator-controller-manager"></a>

From the *operator/* directory

* `make build` builds the controller manager binary.
* `make test` runs unit and EnvTest-enabled integration tests.
* `make docker-build` builds the operator controller manager container.

### Change the BeegfsDrivers API
<a name="change-the-driver-api"></a>

1. Modify the structs and kubebuilder markers in *api/*.
2. Run `make generate manifests` to update generated deep copy functions and
   the CRD.

### Change the Behavior of the Controller
<a name="change-the-behavior-of-the-controller"></a>

1. Modify the code in *controllers/*.
2. Build and/or test as described above.

### Change the Way OLM Displays or Installs the Operator
<a name="change-olm-display-or-install"></a>

1. Modify the CSV by:
   * Modifying the operator-sdk markers in *api/* for
     [some fields](https://sdk.operatorframework.io/docs/olm-integration/generation/#csv-fields),
     or
   * Modifying *config/manifests/bases/beegfs-csi-driver-operator.clustersericeversion.yaml* for other fields.
2. Run `make bundle`.

### Prepare Changes For a Pull Request
<a name="prepare-changes-for-a-pull-request"></a>

If you have done manual testing of changes with builds created from the dev environment, you'll want
to unset any custom VERSION or IMAGE_TAG_BASE variable overridden using the environment or changes
to the Makefile . Then run through the following sequence to generate the necessary changes without
pushing any builds to the registry.

* make generate manifests
* make build
* make manifests bundle

Note: Generally the VERSION set in the Makefile and in various manifests should not be updated until
it is time to actually release a new version of the driver. If this version is updated beforehand,
then it becomes difficult to find all the places in the repository where the old version needs to be
updated when it is actually time to release. Unless a new Git tag is pushed, the container images
tagged with a released version of the operator and bundle will not be overwritten.

### Update the operator-sdk version
<a name="update-the-operator-sdk-version"></a>

When updating the operator-sdk version start by reviewing the [Upgrade SDK
Version](https://sdk.operatorframework.io/docs/upgrading-sdk-version/) section
of the Operator SDK documentation.

First identify the operator-sdk version currently being used which is documented
in the [development environment](#development-environment) section. For each new
operator-sdk version beyond the version that is currently being used, go to the
upgrade documentation for that version. The upgrade documentation may list a set
of modifications that should be applied to the project to make the project
compatible with the newer operator-sdk version. Apply any changes that are
targeted for 'go/v3' based operators. Some changes are not applicable if they
only apply to other operator types (Ex. Helm or Ansible).

NOTE: Changes need to be applied for **all** operator-sdk versions up to and
including the new target version.

Follow the [operator-sdk
installation](https://sdk.operatorframework.io/docs/installation/) instructions
to install the version of the operator-sdk that will be used. If there were
multiple versions released make sure to apply the changes for all versions, but
only install the operator-sdk version that you want to end up using.

After the new operator-sdk version is installed follow the steps for [preparing
changes for a pull request](#prepare-changes-for-a-pull-request) to generate the
changes related to the new operator-sdk version that need to be committed.

## Testing
<a name="testing"></a>

We can test the BeeGFS CSI driver operator in a number of ways. Each of the 
methods outlined below can be used manually and is also used (at least 
partially) in the NetApp-internal Jenkins pipeline.

### Bundle Validation
<a name="bundle-validation"></a>

`make bundle` executes `operator-sdk bundle validate ./bundle --select-optional 
suite=operatorframework` under the hood. [This 
command](https://sdk.operatorframework.io/docs/cli/operator-sdk_bundle_validate/) 
validates both the content and format of an operator bundle.

`operator-sdk scorecard bundle` runs a suite of tests in Pods on the Kubernetes 
cluster that KUBECONFIG points to. Currently, we only run the [built-in 
tests](https://sdk.operatorframework.io/docs/testing-operators/scorecard/#built-in-tests),
which do additional bundle validation. The Jenkins pipeline runs Scorecard 
tests, but they are currently run by any `make` command.

### Unit Testing
<a name="unit-testing"></a>

Most controller helper methods in *controllers/* have a corresponding unit test 
in *controllers/beegfsdriver_controller_test.go*. These tests run on every 
invocation of `make test`. The main reconcile loop is too big (and has too many 
external interactions) to test this way, so we use EnvTest-enabled integration 
tests for it instead.

### Integration Testing with EnvTest
<a name="envtest"></a>

The controller-runtime 
[EnvTest](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest) 
package enables limited integration testing of Kubernetes controllers, like the 
BeeGFS CSI driver operator. EnvTest runs local etcd and kube-apiserver 
instances, providing the ability to create, modify, and delete custom resources 
and to make assertions against these custom resources. EnvTest does NOT run a 
kube-controller-manager or a kube-scheduler. This means anything that might be 
expected to happen "out-of-band" (e.g., Pods start spinning up to meet the 
requirements of a Deployment or resources are garbage collected) does NOT 
occur. *controllers/beegfsdriver_controller_test.go* and 
*controllers/suite_test.go* implement a suite of integration tests that run 
with every invocation of `make test`.

### Functional Testing
<a name="functional-testing"></a>

Outlined below are a number of ways of running the BeeGFS CSI driver operator 
against or within a Kubernetes cluster. After deploying with one of these 
methods, it is possible to test arbitrary interactions with the operator or run 
an end-to-end test suite against an operator deployed driver.

#### Run Operator Locally Against Any Cluster
<a name="functional-testing-run-local"></a>

The easiest way to verify a change to the operator API or logic is to build and
run it locally with access to a Kubernetes cluster. In this scenario, the
operator is NOT packaged into a container or deployed as a cluster workload.

Prerequisites:

* The KUBECONFIG environment variable points to a valid kubeconfig file.
* The cluster referenced by the kubeconfig file does NOT have a running BeeGFS
  CSI driver operator or BeeGFS CSI driver deployment.
* The go.mod referenced Go version is installed on the path.

Steps:

In one terminal window:

1. Navigate to the *operator/* directory.
1. Execute `make install` to add the BeegfsDriver CRD to the cluster.
1. Execute `make run` to start the operator locally.

In a separate terminal window (the first is consumed by the running operator):

1. Navigate to the *operator/* directory.
1. Create your own *csi-beegfs-cr.yaml* file from scratch or copy one from 
   *../test/env* if you have access to NetApp internal test resources.  
   NOTE: For many test cases, you will want to set the 
   `containerImageOverrides.beegfsCsiDriver.image` and/or 
   `containerImageOverrides.beegfsCsiDriver.tag` fields to ensure the default 
   driver image (usually the last released version) is not used.
1. Execute `kubectl apply -f csi-beegfs-cr.yaml` to deploy the driver.
1. Optionally modify *csi-beegfs-cr.yaml* and redeploy the driver with `kubectl 
   apply -f hack/example_driver.yaml` to test reconfiguration scenarios.
1. Use commands like `kubectl describe beegfsdriver`, `kubectl describe sts`,
   and `kubectl describe ds` to verify expected behavior.
1. Delete the driver with `kubectl delete -f csi-beegfs-cr.yaml`.

As commands are executed, logs in the first terminal window show the
actions the operator is taking.

#### Run Operator With OLM Integration Using kubectl

Prerequisites:

* You have `kubectl` access to Kubernetes cluster with the Operator Life Cycle
  Manager (OLM) already installed.
  * Refer to the OLM documentation for how to [get
    started](https://olm.operatorframework.io/docs/getting-started/.)
  * For example you might run:
```
curl -L https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.25.0/install.sh -o install.sh
chmod +x install.sh
./install.sh v0.25.0
```
* The Kubernetes cluster does NOT have a running BeeGFS CSI driver operator or
  BeeGFS CSI driver deployment.
* The go.mod referenced Go version is installed on the path.
* operator-sdk is installed on the path.
* All prerequisites for the BeeGFS CSI driver must be installed on your
  Kubernetes nodes. If you are using Minikube there is a script to do this at
  `hack/minikube_install_driver_prerequisites.sh`.
   * You must also provide Minikube its own base client configuration file. For example you might
     bind mount /etc/beegfs from the host OS into the Minikube container using `minikube mount
     /etc/beegfs:/etc/beegfs` (note the command must stay running for the mount to stay active).

Steps:

1. In a terminal, navigate to the *operator/* directory.
2. Set the IMAGE_TAG_BASE environment variable so that it refers to a
   container registry namespace you have access to. For example, `export
   IMAGE_TAG_BASE=ghcr.io/thinkparq/test-beegfs-csi-driver-operator`.
3. Set the VERSION environment variable. For example, execute
   `export VERSION=1.5.0`. The version MUST be semantic (e.g. 0.1.0) and
   consistent through all operator related make commands. It is easiest to
   simply use the VERSION already specified in *operator/Makefile* if there
   is no compelling reason not to.
4. Execute `make build docker-build docker-push` to build the operator and
   push it to the configured registry namespace.
5. Execute `make manifests bundle bundle-build bundle-push` to build and push a
   bundle image operator-sdk can understand.
6. Execute `operator-sdk run bundle $IMAGE_TAG_BASE-bundle:v$VERSION` to cause
   operator-sdk to create a pod that serves the bundle to OLM via subscription
   (as well as other OLM objects).
7. To verify the operator is deployed run `kubectl get operators -A`
8. Experiment with creating/modifying/deleting BeegfsDriver objects. For example
   to deploy with the default minimal configuration run `kubectl apply -f
   config/samples/beegfs_v1_beegfsdriver.yaml`. NOTE: For many test cases, you
   will want to set the `containerImageOverrides.beegfsCsiDriver.image` and/or
   `containerImageOverrides.beegfsCsiDriver.tag` fields before deploying a CR to
   ensure the default driver image (usually the last released version) is not
   used.
9. OPTIONAL: Deploy one or more examples to verify the driver is working
   correctly. If you are using Minikube there is a script at
   `hack/minikube_deploy_all_examples.sh` that handles deploying a BeeGFS file
   system into Kubernetes and deploying all examples.
10.  In the terminal, execute `operator-sdk cleanup beegfs-csi-driver-operator`
   to undo the above steps.


#### Run Operator With OLM Integration Using OpenShift Console (Deprecated)
<a name="functional-testing-install-bundle"></a>

Much of the reason we created the operator was for installation of the driver
into clusters running Operator Lifecycle Manager (OLM), and in particular for
installation of the driver into OpenShift clusters.

Besides the actual operator image itself, OLM requires a "bundle" of files,
including a ClusterServiceVersion, all CRDs, annotations, etc. operator-sdk can
automatically create this bundle. Additionally, operator-sdk can package this
bundle into a container and deploy it into an OpenShift cluster.

This test method should be used when testing interactions between OLM and the
operator or when testing the experience of deploying the driver via the
Openshift console. It skips simulating the installation of the operator itself,
but allows for all expected interactions with the installed operator.

The Jenkins pipeline uses this method to deploy the operator and uses the 
operator to deploy the driver, before running end-to-end driver tests against an 
OpenShift cluster.

Prerequisites:

* The KUBECONFIG environment variable points to a valid kubeconfig file.
* The cluster referenced by the kubeconfig file is an OpenShift cluster.
* The console for the OpenShift cluster referenced by the kubeconfig file
  is accessible via a browser.
* The Openshift cluster referenced by the kubeconfig file does NOT have a
  running BeeGFS CSI driver operator or BeeGFS CSI driver deployment.
* The go.mod referenced Go version is installed on the path.
* operator-sdk is installed on the path.

Steps:

1. In a terminal, navigate to the *operator/* directory.
1. Set the IMAGE_TAG_BASE environment variable so that it refers to a
   container registry namespace you have access to. For example, `export
   IMAGE_TAG_BASE=ghcr.io/thinkparq/test-beegfs-csi-driver-operator`.
1. Set the VERSION environment variable. For example, execute
   `export VERSION=1.2.0`. The version MUST be semantic (e.g. 0.1.0) and
   consistent through all operator related make commands. It is easiest to
   simply use the VERSION already specified in *operator/Makefile* if there
   is no compelling reason not to.
1. Execute `make build docker-build docker-push` to build the operator and
   push it to the configured registry namespace.
1. Execute `make manifests bundle bundle-build bundle-push` to build and push a
   bundle image operator-sdk can understand.
1. Execute `operator-sdk run bundle $IMAGE_TAG_BASE-bundle:v$VERSION` to cause
   operator-sdk to create a pod that serves the bundle to OLM via subscription
   (as well as other OLM objects).
1. In a browser, navigate to the OpenShift console -> Operators -> Installed
   Operators and look for "BeeGFS CSI Driver".
1. Experiment with creating/modifying/deleting BeegfsDriver objects.  
   NOTE: For many test cases, you will want to set the
   `containerImageOverrides.beegfsCsiDriver.image` and/or
   `containerImageOverrides.beegfsCsiDriver.tag` fields before deploying a CR 
   to ensure the default driver image (usually the last released version) is not 
   used.
1. In the terminal, execute `operator-sdk cleanup beegfs-csi-driver-operator`
   to undo the above steps.

#### Install Operator as if From OperatorHub in OpenShift Console (Deprecated)
<a name="functional-testing-install-openshift"></a>

This test method simulates the entire OpenShift deployment workflow. It adds to
the previous method by creating a custom CatalogSource and deploying it such
that the BeeGFS CSI Driver operator itself is available to install alongside
all the other OperatorHub operators.

Prerequisites:

See the previous method.

Steps:

1. Complete the first five steps of the previous test method. Do NOT execute
   `operator-sdk run...`.
1. Execute `make catalog-build catalog-push` to create an index image that
   knows about the operator bundle.
1. Modify the image field of *operator/hack/test_catalog_source.yaml*.
1. Execute `oc apply -f hack/test_catalog_source.yaml` to deploy a
   CatalogSource that references the index image.
1. In a browser, navigate to the OpenShift console -> Operators -> OperatorHub
   Operators, filter for storage, and look for the BeeGFS CSI Driver operator.
1. Create a new namespace to install the operator or choose and existing one.
1. Install the operator.
1. Experiment with creating/modifying/deleting BeegfsDriver objects as with
   the previous method.  
   NOTE: For most test cases, you will want to set the
   `containerImageOverrides.beegfsCsiDriver.image` and/or
   `containerImageOverrides.beegfsCsiDriver.tag` fields before deploying a CR
   to ensure the default driver image (usually the last released version) is not
   used.
1. In Operators -> Installed Operators, use the three dots to the right of
   the BeeGFS CSI Driver operator to uninstall it.
1. In the terminal, execute `oc delete -f hack/test_catalog_source.yaml` to
   delete the CatalogSource.