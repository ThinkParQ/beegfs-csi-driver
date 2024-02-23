# Modifications Copyright 2021 NetApp, Inc. All Rights Reserved.
# Modifications Copyright 2024 ThinkParQ, GmbH. All Rights Reserved.
# Licensed under the Apache License, Version 2.0.

# Use distroless as minimal base image to package the driver binary. Refer to
# https://github.com/GoogleContainerTools/distroless for more details.
FROM --platform=$TARGETPLATFORM gcr.io/distroless/static:latest
LABEL maintainers="ThinkParQ"
LABEL description="BeeGFS CSI Driver"
LABEL org.opencontainers.image.description="BeeGFS CSI Driver"
LABEL org.opencontainers.image.source="https://github.com/ThinkParQ/beegfs-csi-driver"
LABEL org.opencontainers.image.licenses="Apache-2.0"

# Multi-arch images can be built from this Dockerfile. When the container image is built it is
# expected binaries and a chwrap tar file were already created under bin/ using Make. By default
# calling Make with no arguments builds these files for the current architecture with no suffix
# allowing the container image to be built without multiarch support by default.
#
# If Make is called with the `BUILD_PLATFORMS` build argument, then binaries and chwrap tar files
# will be generared for each platform with an architecture suffix. These can then be used to build a
# multiarch container image using `docker buildx build` by specifying the same list of platforms
# using the `--platform` flag. Note the buildx flag and BUILD_PLATFORMS argument accept slightly
# different values, for example to build for both amd64 and arm64:
#
# `make BUILD_PLATFORMS="linux amd64 amd64 amd64;linux arm64 arm64 arm64" all`
# `docker buildx build --platform=linux/amd64,linux/arm64`
ARG TARGETARCH
# Work around the fact TARGETARCH is not set consistently when building multiarch images using
# release-tools versus docker buildx. While release-tools isn't currently used by GitHub Actions to
# publish multiarch images, this is the only thing preventing use of release-tools, which may be
# useful for local testing.
ARG ARCH=$TARGETARCH
WORKDIR /

# Copy architecture specific BeeGFS CSI driver to the image.
COPY bin/beegfs-csi-driver$ARCH /beegfs-csi-driver

# Unpack architecture specific chwrap symbolic links into osutils directory.
ADD bin/chwrap$ARCH.tar /

# Call chwrap linked binaries before container installed binaries.
ENV PATH "/osutils:$PATH"

ENTRYPOINT ["/beegfs-csi-driver"]
