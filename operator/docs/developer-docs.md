# BeeGFS CSI Driver Operator Developer Documentation

## Contents
<a name="contents"></a>

* [Overview](#overview)
* [Important Links](#important-links)
* [Development Environment](#development-environment)
* [Directory Structure](#directory-structure)
* [General Workflows](#general-workflows)
* [Testing](#testing)
  * [Bundle Validation](#bundle-validation)
  * [Unit Testing](#unit-testing)
  * [Integration Testing with EnvTest](#envtest)
  * [Functional Testing](#functional-testing)
    * [Run Operator Locally Against Any Cluster](#functional-testing-run-local)
    * [Run Operator With OLM Integration Using OpenShift Console](#functional-testing-install-bundle)
    * [Install Operator as if From OperatorHub in OpenShift Console](#functional-testing-install-openshift)

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
* [Operator SDK (currently v1.22.2)](https://sdk.operatorframework.io/docs/installation/)

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

From the *operator/* directory

* `make build` builds the controller manager binary.
* `make test` runs unit and EnvTest-enabled integration tests.
* `make docker-build` builds the operator controller manager container.

### Change the BeegfsDrivers API

1. Modify the structs and kubebuilder markers in *api/*.
2. Run `make generate manifests` to update generated deep copy functions and
   the CRD.

### Change the Behavior of the Controller

1. Modify the code in *controllers/*.
2. Build and/or test as described above.

### Change the Way OLM Displays or Installs the Operator

1. Modify the CSV by:
   * Modifying the operator-sdk markers in *api/* for
     [some fields](https://sdk.operatorframework.io/docs/olm-integration/generation/#csv-fields),
     or
   * Modifying *config/manifests/bases/beegfs-csi-driver-operator.clustersericeversion.yaml* for other fields.
2. Run `make bundle`.

### Prepare Changes For a Pull Request

If you have done manual testing of changes with builds created from the dev
environment, you'll want to unset any custom VERSION environment variables or
changes to the Makefile for the version. Then run through the following sequence
to generate the necessary changes without pushing any builds to the registry.

* make generate manifests
* make build
* rm bin/kustomize
* make manifests bundle

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

#### Run Operator With OLM Integration Using OpenShift Console
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
   container registry namespace you have access to. For example, NetApp
   developers should execute `export
   IMAGE_TAG_BASE=docker.repo.eng.netapp.com/<sso>/beegfs-csi-driver-operator`.
   External developers might execute `export
   IMAGE_TAG_BASE=docker.io/<Docker ID>/beegfs-csi-driver-operator`.
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

#### Install Operator as if From OperatorHub in OpenShift Console
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