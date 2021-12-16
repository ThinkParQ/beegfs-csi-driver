# BeeGFS CSI Driver Nomad Examples

This directory contains a template for a BeeGFS CSI volume that can be 
understood by Hashicorp Nomad and an example Redis job that consumes two such 
volumes. See the demo script and [README](../../deploy/nomad/README.md) for a 
simple way to get started experimenting.

## What Is This For?

At this time the BeeGFS CSI driver is NOT SUPPORTED on Nomad, and is for
demonstration and development purposes only. DO NOT USE IT IN PRODUCTION. If 
you want to get a quick idea of how BeeGFS CSI plugin works on
Nomad in a single node environment, this demo is a good option.

## Quick Notes

If you DON'T plan on using the demo script mentioned above, here are a few 
quick notes regarding the example volume (`volume.hcl`).

* The `id` and `name` fields should be modified. This is something the demo 
  script does automatically.
* The `parameters` block should be modified to refer to an existing BeeGFS file 
  system.
