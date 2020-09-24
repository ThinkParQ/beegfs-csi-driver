FROM alpine
LABEL maintainers="Kubernetes Authors"
LABEL description="HostPath Driver"
ARG binary=./bin/hostpathplugin

# Modify /etc/apk/repositories to point to VED mirrors
# TODO(webere): Split out the Dockerfile so that we have an in-VED and out-of-VED version (not important now).
RUN sed -i 's_dl-cdn.alpinelinux.org/alpine_repomirror-ict.eng.netapp.com/alpine-linux_g' /etc/apk/repositories

# Add util-linux to get a new version of losetup.
RUN apk add util-linux
COPY ${binary} /hostpathplugin
ENTRYPOINT ["/hostpathplugin"]
