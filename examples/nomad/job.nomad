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

    # Required and arbitrarily chosen. The ID which creates an individual unit of work in this job. 
    # Full task options can be found at: https://www.nomadproject.io/docs/job-specification/task
    task "beegfs-task" {
      # This plugin has only been tested with the docker driver. It may be possible to support the podman driver in the 
      # future. 
      driver = "docker"

      config {
        image = "alpine:latest"

      }

      volume_mount {
        volume = "beegfs-csi-volume"
        destination = "/mnt/beegfs-csi-volume"
      }

      # Create a file with the alloc's ID as its name to demonstrate the ability to write to BeeGFS. Then, sleeps to 
      # demonstrate the container runs successfully.
      args = [ "ash", "-c", 'touch "/mnt/beegfs-csi-volume/touched-by-${NOMAD_ALLOC_ID}" && sleep 7d' ]

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
