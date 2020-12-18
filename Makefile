# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Modified from kubernetes-csi/csi-driver-hostpath Makefile. Modifications detailed in comments.
# https://github.com/kubernetes-csi/csi-driver-host-path/blob/b927e98e08bda8e27651245797d8e5f91761abb2/Makefile

CMDS ?= beegfsplugin
all: build

# The kubernetes-csi/csi-release-tools project does not include an easy way to build a binary that doesn't need its
# own container image and include it in a different image. We build chwrap any time we build the beegfsplugin container
# so the container can include chwrap.

# Additional prerequisites and the recipe for build-chwrap, container, and push are defined in release-tools/build.make.
container: build-chwrap
push: container  # not explicitly executed in release-tools/build.make

include release-tools/build.make
