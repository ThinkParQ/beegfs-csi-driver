# Copyright 2022 NetApp, Inc. All Rights Reserved.
# Licensed under the Apache License, Version 2.0.

# Browse the full set of configuration options at https://www.nomadproject.io/docs/job-specification.

job "beegfs-csi-plugin-node" {
  # Job type system is analogous to a Kubernetes DaemonSet (runs on all nodes, restarts indefinitely). See other 
  # options at https://www.nomadproject.io/docs/schedulers.
  type = "system"

  # LIKELY TO REQUIRE MODIFICATION.
  # "dc1" is a default for basic deployments, but this depends on the environment.
  datacenters = ["dc1"]

  # A group is analagous to a Kubernetes Pod.
  group "node" {
    task "node" {
      # This plugin has only been tested with the docker driver. It may be possible to support the podman driver in the 
      # future. 
      driver = "docker"

      config {
        image = "ghcr.io/thinkparq/beegfs-csi-driver:v1.8.0"

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

        args = [
          "--driver-name=beegfs.csi.netapp.com",
          "--client-conf-template-path=/host/etc/beegfs/beegfs-client.conf",
          "--config-path=${NOMAD_TASK_DIR}/csi-beegfs-config.yaml",
          "--connauth-path=${NOMAD_SECRETS_DIR}/csi-beegfs-connauth.yaml",
          "--v=3",
          "--endpoint=${CSI_ENDPOINT}",
          "--node-id=node-${node.unique.name}",
        ]

        # We must run with privileges in order to mount volumes.
        privileged = true
      }

      csi_plugin {
        # Specific to Nomad. Some important paths include this field.
        id = "beegfs-csi-plugin"
        type = "node"

        # LIKELY TO REQUIRE MODIFICATION.
        # The BeeGFS CSI driver must be instructed to stage and publish volumes in a directory with the same path 
        # inside and outside of its container. Nomad always facilitates staging and publishing in the administrator 
        # configured Nomad data directory as seen outside the container, but by default it represents this directory 
        # inside the container as /local/csi. This usage of the stage_publish_base_dir field ensures the driver operates 
        # correctly. 
        #
        # This field should be configured like <client.state_dir>/csi/node/<csi_plugin.id>, where:
        #   * csi_plugin.id is set in this file (and is unlikely to change).
        #   * client.state_dir is set in Nomad agent configuration files (and may vary by Nomad cluster). See 
        #     https://www.nomadproject.io/docs/configuration/client for details.
        #
        # Many Nomad agent configurations do not explicitly set client.state_dir. In these cases, this field should be 
        # configured like <data_dir>/client/csi/node/<csi_plugin.id>, where:
        #   * csi_plugin.id is set in this file (and is unlikely to change).
        #   * data_dir is set in Nomad agent configuration files (and may vary by Nomad cluster). See
        #     https://www.nomadproject.io/docs/configuration for details.
        stage_publish_base_dir = "/opt/nomad/client/csi/node/beegfs-csi-plugin"
      }

      resources {
        cpu = 256
        memory = 128
      }
      
      # LIKELY TO REQUIRE MODIFICATION.
      # csi-beegfs-config.yaml is the primary means of configuring the BeeGFS CSI driver. See 
      # https://github.com/ThinkParQ/beegfs-csi-driver/blob/master/docs/deployment.md#managing-beegfs-client-configuration 
      # for details.
      # This stanza must be kept in sync with its partner in controller.nomad.
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
      # This stanza must be kept in sync with its partner in controller.nomad.
      template {
        data = <<EOH
# Place valid csi-beegfs-connauth.yaml contents here.
EOH
        destination = "${NOMAD_SECRETS_DIR}/csi-beegfs-connauth.yaml"
      }
    }
  }
}
