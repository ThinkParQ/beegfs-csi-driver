# Copyright 2021 NetApp authors
# Copyright 2021 HashiCorp authors

# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at: https://mozilla.org/MPL/2.0/.

# The HashiCorp Nomad LICENSE can be found at:
# https://github.com/hashicorp/nomad/blob/main/LICENSE

# The functions in this file are derived from:
# https://github.com/hashicorp/nomad/tree/main/demo/csi/hostpath

# Full job, group, and task options can be found at: https://www.nomadproject.io/docs/job-specification
job "example" {
  # Required and arbitrarily chosen, but the same as in ./plugin.nomad. The ID of the datacenter that runs this job.
  datacenters = ["dc1"]

  # Required and arbitrarily chosen. The ID which defines a series of tasks in this job.
  group "cache" {
    # Number of instances required, we have 2 volumes for this job. 
    count = 2

    # Required and arbitrarily chosen. The ID which defines a given volume from the cluster for this job.
    volume "volume0" {
      type            = "csi" # alt: "host"
      # Matches ID of volumes in run.sh (e.g. id = "test-volume[0]").
      source          = "test-volume"
      attachment_mode = "file-system" # alt: "block-device"
      access_mode     = "single-node-reader-only" # alt: "single-node-writer"
      read_only       = true
      per_alloc       = true
    }

    # Networking for database
    network {
      # Required and arbitrarily chosen. The ID for TCP/UDP allocation in this job. Referenced down below in "config" -> "ports".
      port "db" {
        # bridge mode port to map inside the task network namespace
        to = 6379
      }
    }

    # Required and arbitrarily chosen. The ID which creates an individual unit of work in this job. 
    # Full task options can be found at: https://www.nomadproject.io/docs/job-specification/task
    task "redis" {
      # Optionally substitute, docker, qemu, java or exec. 
      driver = "docker"

      # Full Docker config options can be found at: https://www.nomadproject.io/docs/drivers/docker AND https://www.nomadproject.io/docs/job-specification/job
      config {
        # Docker image pulled from Docker hub
        image = "redis:3.2"
        # ID referenced above in "network" -> "port" "db". 
        ports = ["db"]
      }

      # Specifies how group volume is mounted into the task. Full volume_mount options can be found at: https://www.nomadproject.io/docs/job-specification/volume_mount
      volume_mount {
        volume = "volume0"
        destination = "${NOMAD_ALLOC_DIR}/volume0"
      }

      # Full resources options can be found at: https://www.nomadproject.io/docs/job-specification/resources
      resources {
        # cpu in MHz 
        cpu = 500

        # memory in MB
        memory = 256
      }
    }
  }
}
