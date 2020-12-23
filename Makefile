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
# own container image and include it in a different image. This build-% recipe mirrors an analogous recipe in
# release-tools/buildmake and allows us to explicitly build the binary specified by %.
build-%: check-go-version-go
	# Commands are taken directly from build.make build-%.
	mkdir -p bin
	echo '$(BUILD_PLATFORMS)' | tr ';' '\n' | while read -r os arch suffix; do \
		if ! (set -x; CGO_ENABLED=0 GOOS="$$os" GOARCH="$$arch" go build $(GOFLAGS_VENDOR) -a -ldflags \
		'$(FULL_LDFLAGS)' -o "./bin/$*$$suffix" ./cmd/$*); then \
			echo "Building $* for GOOS=$$os GOARCH=$$arch failed, see error(s) above."; \
			exit 1; \
		fi; \
	done

# The beegfsplugin container requires chwrap to be built and included, so we build it anytime container or push are
# made. Additional prerequisites and the recipes for container and push are defined in release-tools/build.make. A
# different workaround will likely be required for multiarch builds.
container: build-chwrap
push: container  # not explicitly executed in release-tools/build.make

include release-tools/build.make
