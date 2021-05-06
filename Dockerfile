# Modifications Copyright 2021 NetApp, Inc. All Rights Reserved.
# Licensed under the Apache License, Version 2.0.

FROM alpine:3.13.5

LABEL maintainers="NetApp"
LABEL description="BeeGFS CSI Driver"
ARG binary=./bin/beegfs-csi-driver
ARG chwrap=./bin/chwrap

# TODO(jmccormi, A210): Remove and bump Alpine version when this is available in a stable release.
# Temporary mitigation for CVE-2021-28831.
RUN apk add --no-cache "busybox>=1.33" --repository=http://dl-cdn.alpinelinux.org/alpine/edge/main

# Allow this container to call specifically linked binaries when the host filesystem is mounted under /host.
COPY ${binary} /netapp/beegfs-csi-driver
COPY ${chwrap} /netapp/chwrap
# chwrap beegfs-ctl to avoid BeeGFS distribution licensing. 
RUN \
ln -s /netapp/chwrap /netapp/beegfs-ctl && \
true

# Call chwrap linked binaries before container installed binaries.
ENV PATH "/netapp:/$PATH"

ENTRYPOINT ["beegfs-csi-driver"]
