FROM alpine:3.12

LABEL maintainers="NetApp"
LABEL description="BeeGFS Driver"
ARG binary=./bin/beegfsplugin
ARG chwrap=./bin/chwrap

# Add util-linux to get a new version of losetup.
RUN apk add util-linux

# Allow this container to call specifically linked binaries when the host filesystem is mounted under /host.
COPY ${binary} /netapp/beegfsplugin
COPY ${chwrap} /netapp/chwrap
RUN ln -s /netapp/chwrap /netapp/beegfs-ctl && ln -s /netapp/chwrap /netapp/netstat
# Call chwrap linked binaries before container installed binaries.
ENV PATH "/netapp:/$PATH"

ENTRYPOINT ["beegfsplugin"]
