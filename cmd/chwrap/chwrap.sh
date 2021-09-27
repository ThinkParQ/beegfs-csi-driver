#!/bin/sh -e

# https://github.com/NetApp/trident/blob/d9961faf141edb7d5fa081eebd4b70a6005eb96a/chwrap/make-tarball.sh

# Copyright 2020 NetApp, Inc. All Rights Reserved.
# Modifications Copyright 2021 NetApp, Inc. All Rights Reserved.
# Licensed under the Apache License, Version 2.0.

[ -n "$1" ] && [ -n "$2" ] || exit 1

PREFIX=/tmp/$(uuidgen)
mkdir -p $PREFIX/netapp
cp "$1" $PREFIX/netapp/chwrap
for BIN in beegfs-ctl mount touch umount; do
  ln -s chwrap $PREFIX/netapp/$BIN
done
tar --owner=0 --group=0 -C $PREFIX -cf "$2" netapp
rm -rf $PREFIX
