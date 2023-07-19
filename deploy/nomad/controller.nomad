# Copyright 2022 NetApp, Inc. All Rights Reserved.
# Licensed under the Apache License, Version 2.0.

# Browse the full set of configuration options at https://www.nomadproject.io/docs/job-specification.

job "beegfs-csi-plugin-controller" {
  # Job type service is analogous to a Kubernetes Deployment (runs a configurable number of replicas, restarts and/or 
  # reschedules as configured). See other options at https://www.nomadproject.io/docs/schedulers.
  type = "service"

  # LIKELY TO REQUIRE MODIFICATION.
  # "dc1" is a default for basic deployments, but this depends on the environment.
  datacenters = ["dc1"]

  # A group is analagous to a Kubernetes Pod.
  group "controller" {
    count = 1

    task "controller" {
      # This plugin has only been tested with the docker driver. It may be possible to support the podman driver in the 
      # future. 
      driver = "docker"

      config {
        image = "ghcr.io/thinkparq/beegfs-csi-driver:v1.5.0"

        # chwrap is used to execute the beegfs-ctl binary already installed on the host. We also read the 
        # beegfs-client.conf template already installed on the host.
        # The host filesystem is mounted at: /host.
        mount {
          type     = "bind"
          target   = "/host"
          source   = "/"
          readonly = true
          bind_options {
            # Because we chwrap mount/umount, we must propagate the container's /host mounts to the node.
            propagation = "rshared"
          }
        }

        # The BeeGFS CSI driver requires a data directory with the same path inside and outside of its container. 
        # This  directory is already set up for the use of the driver (bind mounted to /local/csi). We set up this 
        # second bind mount to ensure paths are consistent.
        mount {
          type     = "bind"
          target   = "/opt/nomad/client/csi/controller/beegfs-csi-plugin"
          source   = "/opt/nomad/client/csi/controller/beegfs-csi-plugin"
          readonly = false
          bind_options {
            # We must know whether a directory is a mount point in order to decide how to handle it.
            propagation = "rslave"
          }
        }

        args = [
          "--driver-name=beegfs.csi.netapp.com",
          "--client-conf-template-path=/host/etc/beegfs/beegfs-client.conf",
          "--cs-data-dir=/opt/nomad/client/csi/controller/beegfs-csi-plugin",
          "--config-path=${NOMAD_TASK_DIR}/csi-beegfs-config.yaml",
          "--connauth-path=${NOMAD_SECRETS_DIR}/csi-beegfs-connauth.yaml",
          "--v=3",
          "--endpoint=${CSI_ENDPOINT}",
          "--node-id=node-${node.unique.name}",
          # In limited testing, Nomad appears to be susceptible to the same race between NodeUnstageVolume and 
          # DeleteVolume that Kubernetes is. See docs/troubleshooting#orphan-mounts-unstage-timeout-exceeded.
          "--node-unstage-timeout=60"
        ]

        # We must run with privileges in order to mount volumes.
        privileged = true
      }

      csi_plugin {
        # Specific to Nomad. Some important paths include this field.
        id = "beegfs-csi-plugin"
        type = "controller"
      }

      resources {
        cpu = 256
        memory = 128
      }
      
      # LIKELY TO REQUIRE MODIFICATION.
      # csi-beegfs-config.yaml is the primary means of configuring the BeeGFS CSI driver. See 
      # https://github.com/ThinkParQ/beegfs-csi-driver/blob/master/docs/deployment.md#managing-beegfs-client-configuration 
      # for details.
      # This stanza must be kept in sync with its partner in node.nomad.
      template {
        data = <<EOH
# Place valid csi-beegfs-config.yaml contents here.
EOH
        destination = "${NOMAD_TASK_DIR}/csi-beegfs-config.yaml"
      }

      # LIKELY TO REQUIRE MODIFICATION.
      # csi-beegfs-connauth.yaml container connauth information required by the BeeGFS client to mount secured file 
      # systems. See https://github.com/ThinkParQ/beegfs-csi-driver/blob/master/docs/deployment.md#connauth-configuration 
      # for details.
      # This stanza must be kept in sync with its partner in node.nomad.
      template {
        data = <<EOH
# Place valid csi-beegfs-connauth.yaml contents here.
EOH
        destination = "${NOMAD_SECRETS_DIR}/csi-beegfs-connauth.yaml"
      }
    }
  }
}
