# Copyright 2022 NetApp, Inc. All Rights Reserved.
# Licensed under the Apache License, Version 2.0.

# Browse the full set of configuration options at https://www.nomadproject.io/docs/job-specification.

job "beegfs-csi-job" {
  # Job type service is analogous to a Kubernetes Deployment (runs a configurable number of replicas, restarts and/or 
  # reschedules as configured). See other options at https://www.nomadproject.io/docs/schedulers.
  type = "service"
  
  # LIKELY TO REQUIRE MODIFICATION.
  # "dc1" is a default for basic deployments, but this depends on the environment.
  datacenters = ["dc1"]

  # A group is analagous to a Kubernetes Pod.
  group "beegfs-group" {
    count = 1

    task "beegfs-task-container" {
      # Change this from "docker" to "podman" to run with the Podman task driver. test-nomad.sh handles this
      # automatically.
      driver = "docker"

      config {
        image = "alpine:latest"

        # Create a file with the alloc's ID in its name to demonstrate the ability to write to BeeGFS. Then, sleep to 
        # demonstrate the container runs successfully. Use different files for different task drivers to avoid 
        # permissions issues (e.g. a docker container writes as root and an isolated user can't touch its file).
        args = [ "ash", "-c", "touch /mnt/beegfs-csi-volume/touched-by-${NOMAD_ALLOC_ID}-container && sleep 7d" ]
      }

      volume_mount {
        volume = "beegfs-csi-volume"
        destination = "/mnt/beegfs-csi-volume"
      }

      resources {
        cpu = 256
        memory = 128
      }
    }

    task "beegfs-task-exec" {
      driver = "exec"

      config {
        command = "/usr/bin/bash"

        # Create a file with the alloc's ID in its name to demonstrate the ability to write to BeeGFS. Then, sleep to 
        # demonstrate the container runs successfully. Use different files for different task drivers to avoid 
        # permissions issues (e.g. a docker container writes as root and an isolated user can't touch its file).
        args = [ "-c", "touch /mnt/beegfs-csi-volume/touched-by-${NOMAD_ALLOC_ID}-exec && sleep 7d" ]
      }

      volume_mount {
        volume = "beegfs-csi-volume"
        destination = "/mnt/beegfs-csi-volume"
      }

      resources {
        cpu = 256
        memory = 128
      }
    }

    volume "beegfs-csi-volume" {
      type            = "csi"
      source          = "beegfs-csi-volume"
      read_only       = false

      # The BeeGFS CSI driver supports all CSI access modes and is particularly well suited to multi-node workloads.
      # Optionally substitute single-node-reader-only, single-node-writer, multi-node-reader-only, or 
      # multi-node-single-writer.
      access_mode = "multi-node-multi-writer"
  
      # The BeeGFS CSI driver only supports file system attachment (there is no concept of a BeeGFS block device).
      attachment_mode = "file-system"
    }
  }
}
