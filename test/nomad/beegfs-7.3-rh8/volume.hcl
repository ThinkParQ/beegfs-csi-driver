# Copyright 2022 NetApp, Inc. All Rights Reserved.
# Licensed under the Apache License, Version 2.0.

# Browse the full set of configuration options at https://www.nomadproject.io/docs/other-specifications/volume.

id = "beegfs-csi-volume"
name = "beegfs-csi-volume"
type = "csi"

# This must match the plugin's csi_plugin.id.
plugin_id = "beegfs-csi-plugin"

# Passed by Nomad to the BeeGFS CSI driver, but ignored (see
# https://github.com/NetApp/beegfs-csi-driver/blob/master/docs/usage.md#capacity for details).
capacity_min = "1MB"

# Passed by Nomad to the BeeGFS CSI driver, but ignored (see
# https://github.com/NetApp/beegfs-csi-driver/blob/master/docs/usage.md#capacity for details).
capacity_max = "1GB"

# Any number of capabilities can be passed to the BeeGFS CSI driver by Nomad.
capability {
  # The BeeGFS CSI driver supports all CSI access modes and is particularly well suited to multi-node workloads.
  # Optionally substitute single-node-reader-only, single-node-writer, multi-node-reader-only, or 
  # multi-node-single-writer.
  access_mode = "multi-node-multi-writer"
  
  # The BeeGFS CSI driver only supports file system attachment (there is no concept of a BeeGFS block device).
  attachment_mode = "file-system"
}

# BeeGFS CSI driver-specific parameters passed by Nomad during volume creation. See
# https://github.com/NetApp/beegfs-csi-driver/blob/master/docs/usage.md#create-a-storage-class for allowed parameters.
# sysMgmtdHost and volDirBasePath are required.
parameters {
  # Change this to the IP address or FQDN of the BeeGFS management service for an accessible BeeGFS file system.
  sysMgmtdHost   = "10.113.4.71"

  # Change this to a path on an accessible BeeGFS file system that makes sense for dynamic volume creation.
  volDirBasePath = "/nomad/vol/"
}
