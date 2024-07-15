# https://github.com/kubernetes-csi/csi-driver-host-path/blob/b927e98e08bda8e27651245797d8e5f91761abb2/Makefile

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

# Modifications Copyright 2021 NetApp, Inc. All Rights Reserved.
# Modifications Copyright 2024 ThinkParQ, GmbH. All Rights Reserved.
# Licensed under the Apache License, Version 2.0.

CMDS ?= beegfs-csi-driver
# Speed up unit testing by explicitly NOT building anything in the e2e folder.
# Do not run any operator tests during normal testing.
TEST_GO_FILTER_CMD = -e '/test/e2e' -e '/operator'
all: build build-chwrap bin/chwrap.tar

check-go-version:
	./hack/check-go-version.sh

.PHONY: generate-notices
generate-notices:
	@go-licenses report ./cmd/beegfs-csi-driver ./cmd/chwrap --template hack/notice.tpl > NOTICE.md --ignore github.com/thinkparq

# The kubernetes-csi/csi-release-tools project does not include an easy way to build a binary that doesn't need its
# own container image and include it in a different image. This build-% recipe mirrors an analogous recipe in
# release-tools/buildmake and allows us to explicitly build the binary specified by %.
build-%: check-go-version-go
	# Commands are taken directly from build.make build-%.
	mkdir -p bin
	echo '$(BUILD_PLATFORMS)' | tr ';' '\n' | while read -r os arch buildx_platform suffix base_image addon_image; do \
		if ! (set -x; CGO_ENABLED=0 GOOS="$$os" GOARCH="$$arch" go build $(GOFLAGS_VENDOR) -a -ldflags \
		'$(FULL_LDFLAGS)' -o "./bin/$*$$suffix" ./cmd/$*); then \
			echo "Building $* for GOOS=$$os GOARCH=$$arch failed, see error(s) above."; \
			exit 1; \
		fi; \
	done

# Put symbolic links between various commands (e.g. beegfs-ctl, mount, and umount) and cmd/chwrap into a .tar file to
# be unpacked in the container. chwrap.tar is obviously not a binary file, but bin/ is where release-tools/build.make
# outputs files and it is cleaned out on "make clean". If we BUILD_PLATFORMS is set then we will create multiple tar
# files each suffixed with the appropriate architecture. Otherwise we will create a single tar file with no suffix
# for the current architecture.
bin/chwrap.tar: build-chwrap cmd/chwrap/chwrap.sh
	echo '$(BUILD_PLATFORMS)' | tr ';' '\n' | while read -r os arch buildx_platform suffix base_image addon_image; do \
		if ! (set -x; cmd/chwrap/chwrap.sh bin/chwrap$$arch bin/chwrap$$arch.tar osutils); then \
			echo "Building $* for $$arch failed, see error(s) above."; \
			exit 1; \
		fi; \
	done	


# This target is mainly used for development to first rebuild the driver binary before building the
# container for local testing. Since the beegfs-csi-driver container requires chwrap to be built and
# included, we also build it anytime container, push, or push-multiarch are made. Additional
# prerequisites and the recipes for container and push are defined in release-tools/build.make.
#
# IMPORTANT: Because the release tool's build.make file specifies BUILD_PLATFORMS= and cannot be
# modified, a default set of build platforms cannot be specified in this file and thus must be
# always provided on the command line otherwise the resulting files will not work correctly with how
# the project's Dockerfile expects them to be named.
#
# For ARM: `make BUILD_PLATFORMS="linux arm64 arm64 arm64" container` For x86: `make
# BUILD_PLATFORMS="linux amd64 amd64 amd64" container`
container: all
push-multiarch: build-chwrap bin/chwrap.tar
push: container  # not explicitly executed in release-tools/build.make

# For details on what licenses are disallowed see
# https://github.com/google/go-licenses#check 
#
# IMPORTANT: Any exceptions (using --ignore) such as the one for HCL must be
# manually added AFTER the NOTICE file has been updated and/or other appropriate
# steps have been taken based on the license requirements.
test-licenses: generate-notices
	@echo "Checking license compliance..."
	@go-licenses check ./cmd/beegfs-csi-driver/ ./cmd/chwrap --ignore github.com/thinkparq --disallowed_types=forbidden,permissive,reciprocal,restricted,unknown
	@if [ -n "$$(git status --porcelain NOTICE.md)" ]; then \
        echo "NOTICE file is not up to date. Please run 'make generate-notices' and commit the changes."; \
        exit 1; \
    fi

# Skip sanity tests that are known to fail. Use override directive to append to TESTARGS passed in on the command line. 
# TODO(webere, A387): Correctly adhere to the CSI spec.
override TESTARGS += -ginkgo.skip='Controller Service \[Controller Server\] CreateVolume should fail when requesting to create a volume with already existing name and different capacity'

include release-tools/build.make
