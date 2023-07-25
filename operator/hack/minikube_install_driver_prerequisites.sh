#!/bin/bash
set -euo pipefail

export BEEGFS_VERSION=7.3.4

# Install the BeeGFS beegfs-ctl tool into the Minikube container:
minikube ssh "sudo apt-get update"
minikube ssh "sudo apt-get install wget -y"
minikube ssh "sudo wget -q https://www.beegfs.io/release/beegfs_$BEEGFS_VERSION/gpg/GPG-KEY-beegfs -O- | sudo apt-key add -"
minikube ssh "sudo wget -P /etc/apt/sources.list.d/ https://www.beegfs.io/release/beegfs_$BEEGFS_VERSION/dists/beegfs-focal.list"
minikube ssh "sudo apt-get update"
minikube ssh "sudo apt-get install beegfs-utils -y"
