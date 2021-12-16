# Copyright 2021 NetApp authors
# Copyright 2021 HashiCorp authors

# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at: https://mozilla.org/MPL/2.0/.

# The HashiCorp Nomad LICENSE can be found at:
# https://github.com/hashicorp/nomad/blob/main/LICENSE

# The functions in this file are derived from:
# https://github.com/hashicorp/nomad/tree/main/demo/csi/hostpath

#!/bin/bash
# Run the hostpath plugin and create some volumes, and then claim them.
set -e

DEPLOY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
EXAMPLE_DIR="$DEPLOY_DIR/../../examples/nomad"
VOLUME_BASE_NAME=test-volume

run_plugin() {
    local expected
    expected=$(nomad node status | grep -cv ID)
    echo "$ nomad job run ./plugin.nomad"
    nomad job run "${DEPLOY_DIR}/plugin.nomad"

    while :; do
        nomad plugin status beegfs |
            grep "Nodes Healthy        = $expected" && break
        sleep 2
    done
    echo
    echo "$ nomad plugin status beegfs"
    nomad plugin status beegfs
}

create_volumes() {
    echo
    echo "$ cat volume.hcl | sed | nomad volume create -"
    sed -e "s/VOLUME_NAME/${VOLUME_BASE_NAME}[0]/" \
        "${EXAMPLE_DIR}/volume.hcl" | nomad volume create -

    echo
    echo "$ cat volume.hcl | sed | nomad volume create -"
    sed -e "s/VOLUME_NAME/${VOLUME_BASE_NAME}[1]/" \
        "${EXAMPLE_DIR}/volume.hcl" | nomad volume create -
}

claim_volumes() {
    echo
    echo "An example Nomad job that uses the volumes we created can be found in examples/nomad/redis.nomad"
    echo "$ nomad job run ../../examples/nomad/redis.nomad"
    nomad job run "${EXAMPLE_DIR}/redis.nomad"
}

show_status() {
    echo
    echo "$ nomad volume status"
    nomad volume status
}

clean_up() {
    echo
    echo "$ nomad job stop example"
    nomad job stop example
    sleep 10
    echo
    echo "$ nomad volume delete test-volume[0]"
    nomad volume delete "${VOLUME_BASE_NAME}[0]"
    sleep 2
    echo
    echo "$ nomad volume delete test-volume[1]"
    nomad volume delete "${VOLUME_BASE_NAME}[1]"
    sleep 2
    echo
    echo "$ nomad job stop beegfs"
    nomad job stop beegfs
}

run_plugin
create_volumes
claim_volumes
show_status
clean_up
