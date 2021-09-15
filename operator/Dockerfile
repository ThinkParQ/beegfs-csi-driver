# The operator-sdk scaffolded Dockerfile builds the manager in an intermediate image and copies it to the final
# distroless image. This process assumes the scaffolded operator directory is its own project with its own go module,
# and the fact that we include our operator under the go module of the larger beegfs-csi-driver project complicates it.
# Because we already do not build the larger project within an intermediate container, it is more straightforward to
# build the manager directly and then copy it into the container.

# Use distroless as minimal base image to package the manager binary. Refer to
# https://github.com/GoogleContainerTools/distroless for more details.
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY bin/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
