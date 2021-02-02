# BeeGFS CSI Driver

This repository hosts the BeeGFS CSI Driver and all of its build and dependent configuration files to deploy the driver.

## Prerequisite(s)
- Kubernetes, 1.17 or later, cluster with BeeGFS client software installed
- Access to terminal with `kubectl` and BeeGFS client software installed
- BeeGFS file system(s) accesible from the Kubernetes cluster and the terminal

## Deployment
Deployment can be customized depending on your environment and goals:
- [Deployment](docs/deployment.md)
- [Developer deployment](docs/developer-deployment.md)

## Examples
After the driver is deployed and validated you can try some of the provided [examples](examples/README.md).

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
