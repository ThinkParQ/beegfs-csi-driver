FROM alpine
LABEL maintainers="NetApp"
LABEL description="BeeGFS Driver"
ARG binary=./bin/beegfsplugin

# Add util-linux to get a new version of losetup.
RUN apk add util-linux
COPY ${binary} /beegfsplugin
ENTRYPOINT ["/beegfsplugin"]
