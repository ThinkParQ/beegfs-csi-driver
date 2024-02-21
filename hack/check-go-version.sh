#!/usr/bin/env bash

# Copyright 2022 NetApp, Inc. All Rights Reserved.
# Licensed under the Apache License, Version 2.0.

# The GO_BUILD_VERSION is the official go version we are building the BeeGFS CSI driver with.
GO_BUILD_VERSION="go1.22.0"
INSTALLED_VERSION=$(go version | { read _ _ ver _; echo ${ver}; } )  || die "determining version of go failed"

if [ "$INSTALLED_VERSION" == "$GO_BUILD_VERSION" ]
then
  echo "The installed go version (${INSTALLED_VERSION}) matches the expected build go version (${GO_BUILD_VERSION})."
  
else
  echo "WARNING - we expect to build with go version ${GO_BUILD_VERSION} but found go version ${INSTALLED_VERSION}"
fi