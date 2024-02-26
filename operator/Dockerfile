# The operator-sdk scaffolded Dockerfile builds the manager in an intermediate image and copies it to the final
# distroless image. This process assumes the scaffolded operator directory is its own project with its own go module,
# and the fact that we include our operator under the go module of the larger beegfs-csi-driver project complicates it.
# Because we already do not build the larger project within an intermediate container, it is more straightforward to
# build the manager directly and then copy it into the container.

# Use distroless as minimal base image to package the manager binary. Refer to
# https://github.com/GoogleContainerTools/distroless for more details.
FROM --platform=$TARGETPLATFORM gcr.io/distroless/static:nonroot
LABEL maintainers="ThinkParQ"
LABEL description="BeeGFS CSI Driver Operator"
LABEL org.opencontainers.image.description="BeeGFS CSI Driver Operator"
LABEL org.opencontainers.image.source="https://github.com/ThinkParQ/beegfs-csi-driver/operator"
LABEL org.opencontainers.image.licenses="Apache-2.0"

# Multi-arch images can be built from this Dockerfile. When the container image is built it is
# expected the controller binary was already created and exists bin/ using Make. By default calling
# Make with no arguments builds these files for the current architecture with no suffix allowing the
# container image to be built without multiarch support by default.
#
# If Make is called with the `BUILD_PLATFORMS` build argument, a controller binary will be
# compiled for each platform with an architecture suffix. These can then be used to build a
# multiarch container image using `docker buildx build` by specifying the same list of platforms
# using the `--platform` flag. Note the buildx flag and BUILD_PLATFORMS argument accept slightly
# different values, for example to build for both amd64 and arm64:
#
# `make BUILD_PLATFORMS="linux amd64 amd64 amd64;linux arm64 arm64 arm64" all` 
# `docker buildx build --platform=linux/amd64,linux/arm64`
ARG TARGETARCH
WORKDIR /

# Copy architecture specific manager to the image.
COPY bin/manager$TARGETARCH /manager
USER 65532:65532

ENTRYPOINT ["/manager"]
