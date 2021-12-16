# Copyright 2021 NetApp authors
# Copyright 2021 HashiCorp authors

# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at: https://mozilla.org/MPL/2.0/.

# The HashiCorp Nomad LICENSE can be found at:
# https://github.com/hashicorp/nomad/blob/main/LICENSE

# The functions in this file are derived from:
# https://github.com/hashicorp/nomad/tree/main/demo/csi/hostpath

# Full volume spec options can be found at: https://www.nomadproject.io/docs/commands/volume 
# ex. https://www.nomadproject.io/docs/commands/volume/register
# ex. https://www.nomadproject.io/docs/commands/volume/create

# Replaced by sed in run.sh (e.g. id = "test-volume[0]"). The bracketed integer is important for jobs run
# with job.group.count >1 and job.group.volume.per_alloc = true.
id = "VOLUME_NAME"

# Replaced by sed in run.sh (e.g. id = "test-volume[0]").
name = "VOLUME_NAME"

# Only "csi" is supported.
type = "csi"

# Arbitrarily chosen, but the same as in ./plugin.nomad. The ID of the CSI plugin that manages this volume.
plugin_id = "beegfs-plugin0"

# Passed by Nomad to the BeeGFS CSI driver, but ignored (see
# https://github.com/NetApp/beegfs-csi-driver/blob/master/docs/usage.md#capacity for details).
capacity_min = "1MB"

# Passed by Nomad to the BeeGFS CSI driver, but ignored (see
# https://github.com/NetApp/beegfs-csi-driver/blob/master/docs/usage.md#capacity for details).
capacity_max = "1GB"

# Any number of capabilities can be passed to the BeeGFS CSI driver by Nomad.
capability {
  # The BeeGFS CSI driver supports all CSI access modes and is particularly well suited to multi-node workloads.
  # Optionally substitute single-node-writer, multi-node-reader-only, multi-node-single-writer, or
  # multi-node-multi-writer.
  access_mode = "single-node-reader-only"
  
  # The BeeGFS CSI driver only supports file system attachment (there is no concept of a BeeGFS block device).
  attachment_mode = "file-system"
}

# Any number of capabilities can be passed to the BeeGFS CSI driver by Nomad.
capability {
  # The BeeGFS CSI driver supports all CSI access modes and is particularly well suited to multi-node workloads.
  # Optionally substitute single-node-reader-only, multi-node-reader-only, multi-node-single-writer, or
  # multi-node-multi-writer.
  access_mode = "single-node-writer"

    # The BeeGFS CSI driver only supports file system attachment (there is no concept of a BeeGFS block device).
  attachment_mode = "file-system"
}

# BeeGFS CSI driver-specific parameters passed by Nomad during volume creation. See
# https://github.com/NetApp/beegfs-csi-driver/blob/master/docs/usage.md#create-a-storage-class for allowed parameters.
# sysMgmtdHost and volDirBasePath are required.
parameters {
  # Change this to the IP address or FQDN of the BeeGFS management service for an accessible BeeGFS file system.
  sysMgmtdHost   = "1.1.1.1"

  # Change this to a path on an accessible BeeGFS file system that makes sense for dynamic volume creation.
  volDirBasePath = "/nomad/vol/"
}
