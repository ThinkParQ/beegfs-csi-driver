# BeeGFS CSI Driver Kubernetes Examples 

This directory contains example Kubernetes applications:

|Directory|Description|
|---|---|
|all/      |is a combination of multiple provisioning methods. Feature-gated methods k8s disables by default are excluded.|
|dyn/      |dynamically provisions a persistent volume.  This is useful for persistent scratch space.                     |
|ge/       |dynamically provisions a generic ephemeral `ge` volume.  This is useful for ephemeral scratch space.          |
|static/   |statically provisions a persistent volume.  This is useful for shared data.                                   |
|static-ro/|statically provisions a persistent volume and mounts it read-only.  This is useful for reading shared data.   |

Usage: `kubectl apply -f ${EXAMPLE_DIRECTORY}`

Warning:  The `all/` example shares content from other examples.  It may be
necessary to remove, `kubectl delete -f ${EXAMPLE_DIRECTORY}`, other examples
before applying `all/`.
