#!/bin/sh

OPENSHIFT_VERSIONS="\"v4.11\""

{
  echo ""
  echo "  # Set minimum OpenShift version"
  echo "  com.redhat.openshift.versions: $OPENSHIFT_VERSIONS"
} >> bundle/metadata/annotations.yaml

echo "\n# Set minimum OpenShift version" >> bundle.Dockerfile
echo "LABEL com.redhat.openshift.versions=$OPENSHIFT_VERSIONS" >> bundle.Dockerfile