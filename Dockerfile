FROM alpine:3.12

LABEL maintainers="NetApp"
LABEL description="BeeGFS CSI Driver"
ARG binary=./bin/beegfs-csi-driver
ARG chwrap=./bin/chwrap

# Allow this container to call specifically linked binaries when the host filesystem is mounted under /host.
COPY ${binary} /netapp/beegfs-csi-driver
COPY ${chwrap} /netapp/chwrap
# add util-linux to get a new version of losetup.
# chwrap beegfs-ctl to avoid BeeGFS distribution licensing. 
RUN \
apk add util-linux && \
ln -s /netapp/chwrap /netapp/beegfs-ctl && \
true

# Call chwrap linked binaries before container installed binaries.
ENV PATH "/netapp:/$PATH"

ENTRYPOINT ["beegfs-csi-driver"]
