# Copyright 2021 NetApp authors
# Copyright 2021 HashiCorp authors

# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at: https://mozilla.org/MPL/2.0/.

# The HashiCorp Nomad LICENSE can be found at:
# https://github.com/hashicorp/nomad/blob/main/LICENSE

# The functions in this file are derived from:
# https://github.com/hashicorp/nomad/tree/main/demo/csi/hostpath

# Full configuration options can be found at: https://www.nomadproject.io/docs/configuration

# Path for agent config is /etc/nomad.d/nomad.hcl

# Local directory used to store agent state.
data_dir = "/opt/nomad/data"

# "0.0.0.0" is the IP address of the default private network interface advertised.
bind_addr = "0.0.0.0"

# Configures the Nomad agent to operate in server mode to participate in scheduling decisions, register with service discovery, handle join failures, and more.
server {
  enabled = true

  # A value of 1 does not provide any fault tolerance and is not recommended for production use cases.
  bootstrap_expect = 1
}

# Configures the Nomad agent to accept jobs as assigned by the Nomad server, join the cluster, and specify driver-specific configuration.
client {
  enabled = true

  # Addresses for the Nomad servers this client should join.
  servers = ["127.0.0.1:4646"]
}

# Full plugin and Docker options can be found at: https://www.nomadproject.io/docs/configuration/plugin
plugin "docker" {

  config {
    # Also enabled in plugin.nomad
    allow_privileged = true

    volumes {
      # Allows tasks to bind host paths (volumes) inside their container and use volume drivers (volume_driver).
      enabled = true
    }
  }
}
