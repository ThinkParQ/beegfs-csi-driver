#!/bin/bash
set -euo pipefail

export BEEGFS_VERSION=7.4.6

# Install the BeeGFS beegfs-ctl tool into the Minikube container:
minikube ssh "sudo rm -f /etc/apt/sources.list.d/*"
minikube ssh "sudo apt-get update"
minikube ssh "sudo apt-get install wget -y"
minikube ssh "sudo wget https://www.beegfs.io/release/beegfs_$BEEGFS_VERSION/gpg/GPG-KEY-beegfs -O /etc/apt/trusted.gpg.d/beegfs.asc"
minikube ssh "sudo wget -P /etc/apt/sources.list.d/ https://www.beegfs.io/release/beegfs_$BEEGFS_VERSION/dists/beegfs-focal.list"
minikube ssh "sudo apt-get update"
minikube ssh "sudo apt-get install beegfs-utils -y"
minikube ssh "sudo wget https://raw.githubusercontent.com/ThinkParQ/beegfs/refs/heads/master/client_module/build/dist/etc/beegfs-client.conf -O /etc/beegfs/beegfs-client.conf"
