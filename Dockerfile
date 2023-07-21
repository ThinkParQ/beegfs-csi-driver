# Modifications Copyright 2021 NetApp, Inc. All Rights Reserved.
# Licensed under the Apache License, Version 2.0.

# Use distroless as minimal base image to package the driver binary. Refer to
# https://github.com/GoogleContainerTools/distroless for more details.
FROM gcr.io/distroless/static:latest

LABEL maintainers="ThinkParQ"
LABEL description="BeeGFS CSI Driver"
LABEL org.opencontainers.image.description="BeeGFS CSI Driver"
LABEL org.opencontainers.image.source="https://github.com/ThinkParQ/beegfs-csi-driver"
LABEL org.opencontainers.image.licenses="Apache-2.0"

# Copy all built binaries to netapp/ directory.
COPY bin/beegfs-csi-driver bin/chwrap netapp/

# Add chwrap symbolic links to netapp/ directory.
ADD bin/chwrap.tar /

# Call chwrap linked binaries before container installed binaries.
ENV PATH "/netapp:/$PATH"

ENTRYPOINT ["beegfs-csi-driver"]
