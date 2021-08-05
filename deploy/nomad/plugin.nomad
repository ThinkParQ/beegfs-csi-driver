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
job "beegfs-csi-plugin" {
  # Analogous to a Kubernetes DaemonSet (runs on every node, should never fail) and "service" appears to be analogous to a Kubernetes StatefulSet or a Deployment.
  # Full type options can be found at: https://www.nomadproject.io/docs/schedulers
  type = "system"

  # Required and arbitrarily chosen, but the same as in ./redis.nomad. The ID of the datacenter that runs this job.
  datacenters = ["dc1"]

  # Required and arbitrarily chosen. The ID which defines a series of tasks in this job.
  group "csi" {

    # Required and arbitrarily chosen. The ID which creates an individual unit of work in this job. 
    # Full task options can be found at: https://www.nomadproject.io/docs/job-specification/task
    task "plugin" {
      # The BeeGFS CSI driver supports Docker. Optionally substitute, docker, qemu, java or exec. 
      driver = "docker"

      # Represents yml for BeeGFS client configuration. config, beegfsClientConf, and connUseRDMA are for an arbitrary file system.
      # https://github.com/NetApp/beegfs-csi-driver/blob/c65b53757afb1828d95521ec929e06a117f9a689/docs/deployment.md#managing-beegfs-client-configuration
      template {
        data        = <<EOH
config:
  beegfsClientConf:
    connUseRDMA: true
        EOH
        destination = "${NOMAD_TASK_DIR}/csi-beegfs-config.yaml"
      }

      # Represents yml for BeeGFS client connection authorization. connAuth and sysMgmtdHost are for an arbitrary file system.
      # https://github.com/NetApp/beegfs-csi-driver/blob/c65b53757afb1828d95521ec929e06a117f9a689/docs/deployment.md#connauth-configuration
      template {
        data        = <<EOH
- connAuth: secret1
  sysMgmtdHost: 1.1.1.1
        EOH
        destination = "${NOMAD_SECRETS_DIR}/csi-beegfs-connauth.yaml"
      }

      # Full Docker config options can be found at: https://www.nomadproject.io/docs/drivers/docker AND https://www.nomadproject.io/docs/job-specification/job
      config {

        # Full Docker mount options can be found at: https://www.nomadproject.io/docs/drivers/docker#mount
        mount {
          type     = "bind"
          # chwrap is used to execute the beegfs-ctl binary already installed on the host. We also read the beegfs-client.conf template on the host.
          # The host filesystem is mounted at: /host.
          target   = "/host"
          source   = "/"
          readonly = true
        }

        # Docker hub URL
        image = "docker.repo.eng.netapp.com/netapp/beegfs-csi-driver:v1.1.0"

        # Arguments passed directly to the container, find in "var" in main.go.
        # https://github.com/NetApp/beegfs-csi-driver/blob/c65b53757afb1828d95521ec929e06a117f9a689/cmd/beegfs-csi-driver/main.go
        args = [
          "--driver-name=beegfs.csi.netapp.com",
          "--client-conf-template-path=/host/etc/beegfs/beegfs-client.conf",
          "--cs-data-dir=/opt/nomad/data/client/csi/monolith/beegfs-plugin0",
          "--config-path=${NOMAD_TASK_DIR}/csi-beegfs-config.yaml",
          "--connauth-path=${NOMAD_SECRETS_DIR}/csi-beegfs-connauth.yaml",
          "--v=5",
          "--endpoint=unix://opt/nomad/data/client/csi/monolith/beegfs-plugin0/csi.sock",
          "--node-id=node-${NOMAD_ALLOC_INDEX}",
        ]

        # All CSI node plugins will need to run as privileged tasks so they can mount volumes to the host.
        privileged = true
      }

      # Full CSI options can be found at: https://www.nomadproject.io/docs/job-specification/csi_plugin
      csi_plugin {
        # Required and arbitrarily chosen. The ID of the CSI provided to the cluster for this job.
        id = "beegfs-plugin0"

        # The BeeGFS CSI driver has both controller and node service components. It can be deployed as a monolith and respond to both controller and node service RPCs.
        type = "monolith"

        # Also find in nomad-agent.hcl, this is the local directory used for Nomad; default w/o mount_dir is: /opt/nomad/data.
        mount_dir = "/opt/nomad/data/client/csi/monolith/beegfs-plugin0"
      }

      # Full resources options can be found at: https://www.nomadproject.io/docs/job-specification/resources
      resources {
        # cpu in MHz
        cpu = 256

        # memory in MB
        memory = 128
      }
    }
  }
}
