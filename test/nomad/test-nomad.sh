#!/bin/bash

# Copyright 2022 NetApp, Inc. All Rights Reserved.
# Licensed under the Apache License, Version 2.0.

USAGE=$(cat << EOH
Usage: test-nomad.sh <directory containing nomad files> [start/stop]

This script is primarily intended for automated testing.

This script is only successful in ideal situations. If BeeGFS CSI artifacts already exist in Nomad, failure is likely.

This script takes the following actions:
  * If command is start (or empty):
      * Deploys the BeeGFS CSI controller service.
      * Deploys the BeeGFS CSI node service.
      * Creates a BeeGFS CSI volume.
      * Deploys an application to consume the BeeGFS CSI volume.
  * If command is stop (or empty):
      * Stops the application consuming a BeeGFS CSI volume.
      * Deletes the BeeGFS CSI volume.
      * Deletes the BeeGFS CSI controller service.
      * Deletes the BeeGFS CSI node service.

Assumptions:
  * NOMAD_ADDR is set (so that the CLI can communicate with Nomad)
  * NOMAD_CACERT is set (so that the CLI can communicate with Nomad)
  * CSI_CONTAINER_IMAGE is set (if necessary) (so that an appropriate container image is deployed)
  * CONTAINER_DRIVER is set (optionally) (Podman can be used instead of Docker with CONTAINER_DRIVER=podman)
  * The following files exist:
      * <$1>/<CONTAINER_DRIVER>/controller.nomad (containing appropriate csi-beegfs-config.yaml and csi-beegfs-connauth.yaml)
      * <$1>/<CONTAINER_DRIVER>/node.nomad (containing appropriate csi-beegfs-config.yaml and csi-beegfs-connauth.yaml)
      * <$1>/volume.hcl (containing appropriate sysMgmtdHost and volDirBasePath parameters)
      * job.nomad
EOH
)

set -e

if [ -z $1 ]; then
    echo "${USAGE}"
    exit 1
fi

if [ ! -z $2 ] && [ "$2" != "start" ] && [ "$2" != "stop" ]; then
    echo "$USAGE"
    exit 2
fi

CONTAINER_DRIVER="${CONTAINER_DRIVER:-docker}"
if [ $CONTAINER_DRIVER != "docker" ] && [ $CONTAINER_DRIVER != "podman" ]; then
    echo "CONTAINER_DRIVER must be docker or podman, not $CONTAINER_DRIVER"
    exit 3
fi

if [ -z $2 ] || [ $2 == "start" ]; then
    echo "Running deployment start..."
    if [ ! -z $CSI_CONTAINER_IMAGE ]; then
        sed "s|docker.repo.eng.netapp.com/globalcicd/apheleia/beegfs-csi-driver:master|$CSI_CONTAINER_IMAGE|g" $(realpath $1/$CONTAINER_DRIVER/controller.nomad) | nomad job run -
        sed "s|docker.repo.eng.netapp.com/globalcicd/apheleia/beegfs-csi-driver:master|$CSI_CONTAINER_IMAGE|g" $(realpath $1/$CONTAINER_DRIVER/node.nomad) | nomad job run -
    else
        nomad job run "$(realpath $1/$CONTAINER_DRIVER/controller.nomad)"
        nomad job run "$(realpath $1/$CONTAINER_DRIVER/node.nomad)"
    fi

    # It can take some time for the Nomad plugin infrastructure to realize a controller service is available.
    # This test is somewhat brittle, but it works "for now".
    i=0
    while ! [ $(nomad plugin status beegfs-csi-plugin | awk '/Controllers Healthy/{print $NF}') -eq $(nomad plugin status beegfs-csi-plugin | awk '/Controllers Expected/{print $NF}') ]; do
        if [ $i -lt 30 ]; then
            echo "Waited $i seconds for controller to be healthy..."
            sleep 5
            i=$((i+5))
        else
            echo "Waited too long for controller to be healthy. Exiting..."
            exit 3
        fi
    done

    # It can take some time for the Nomad plugin infrastructure to realize a node service is available.
    # This test is somewhat brittle, but it works "for now".
    i=0
    while ! [ $(nomad plugin status beegfs-csi-plugin | awk '/Nodes Healthy/{print $NF}') -eq $(nomad plugin status beegfs-csi-plugin | awk '/Nodes Expected/{print $NF}') ]; do
        if [ $i -lt 30 ]; then
            echo "Waited $i seconds for nodes to be healthy..."
            sleep 5
            i=$((i+5))
        else
            echo "Waited too long for nodes to be healthy. Exiting..."
            exit 3
        fi
    done

    nomad volume create "$(realpath $1/volume.hcl)"
    
    if [ $CONTAINER_DRIVER == podman ]; then
        sed 's|docker|podman|g' "$(dirname $0)/job.nomad" | nomad job run -
    else
        nomad job run "$(dirname $0)/job.nomad"
    fi
fi

if [ -z $2 ] || [ $2 == "stop" ]; then
    echo "Running deployment stop..."
    nomad job stop -purge "beegfs-csi-job"
    nomad volume delete "beegfs-csi-volume"
    nomad job stop -purge "beegfs-csi-plugin-controller"
    nomad job stop -purge "beegfs-csi-plugin-node"
fi
