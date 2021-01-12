# CSI BeeGFS Driver

This repository hosts the CSI BeeGFS driver and all of its build and dependent configuration files to deploy the driver.

## Prerequisite(s)
- Kubernetes cluster
- Running version 1.17 or later
- Access to terminal with `kubectl` installed

## Deployment
Deployment can be customized depending on your environment and goals:
- [Deployment](docs/deployment.md)
- [Developer deployment](docs/developer-deployment.md)

## Examples
The following examples assume that the CSI BeeGFS driver has been deployed and validated:
- Volume snapshots
  - [Kubernetes 1.17 and later](docs/example-snapshots-1.17-and-later.md)
  - [Kubernetes 1.16 and earlier](docs/example-snapshots-pre-1.17.md)
- [Inline ephemeral volumes](docs/example-ephemeral.md)

## Building the binaries
If you want to build the driver yourself, you can do so with the following command from the root directory:

```shell
make
```

## Building the containers

```shell
make container
```

## Building and pushing the containers

```shell
make push
```

Optionally set REGISTRY_NAME or IMAGE_TAGS:

```shell
# Prerequisite(s):
#   Change 'docker.netapp.com/k8scsi'.
#   Change 'devBranchName-canary'.
#   $ docker login docker.netapp.com 
# REGISTRY_NAME and IMAGE_TAGS must be specified as make arguments.
# REGISTRY_NAME and IMAGE_TAGS cannot be pulled from the environment.
make REGISTRY_NAME=docker.netapp.com/k8scsi IMAGE_TAGS=devBranchName-canary push
```

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](http://slack.k8s.io/)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-dev)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

[owners]: https://git.k8s.io/community/contributors/guide/owners.md
[Creative Commons 4.0]: https://git.k8s.io/website/LICENSE
