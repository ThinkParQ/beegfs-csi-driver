# BeeGFS CSI Driver Deployment

## General Deployment
TODO: when we know how

## Kubernetes Deployment
TODO: when we know how

## General Configuration

The driver is ready to be used right out of the box, but many environments may
either require or benefit from additional configuration.

The driver loads a configuration file on startup which it uses as a template to
create the necessary configuration files to properly mount a BeeGFS file system.
A beegfs-client.conf file does NOT ship with the driver, so it applies the
values defined in its configuration file on top of the default
beegfs-client.conf that ships with each BeeGFS distribution. Each *config*
section may optionally contain parameters that override previous sections.

Depending on the topology of your cluster, some nodes MAY need different
configuration than others, so each node maintains its own unique copy of the
configuration file. For non-Kubernetes deployments, it is the administrator's
responsibility to distribute an appropriate file to each node. See [Kubernetes
Configuration](#kubernetes-configuration) for the Kubernetes-native way to
manage configuration within a Kubernetes cluster.

The *beegfsClientConf* section contains parameters taken directly out of a
beegfs-client.conf configuration file. In particular, the beegfs-client.conf 
file contains a number of references to other files (e.g. 
*connInterfacesFile*). The CSI configuration file instead expects a YAML list,
which it uses to generate the expected file. See [beegfsClientConf
Parameters](#beegfsclientconf-parameters) for more detail about supported
beegfsClientConf parameters.

The order of precedence for configuration option overrides is described by
"PRECEDENCE" comments in the example below. In general, precedence is as
follows: 
1. *fileSystemSpecificConfigs[i].config*. (A file system specific config is 
   mapped to its respective file system by the 
   *fileSystemSpecificConfigs[i].sysMgmtdHost*.)
1. The outermost *config*.
1. Locally installed BeeGFS configuration files: *beegfs-client.conf*, 
   *connInterfacesFile*, *connNetFilterFile*, *connTcpOnlyFilterFile*.

NOTE: All configuration, and in particular *fileSystemSpecificConfigs*
configuration is OPTIONAL! In many situations, only the outermost *config* is 
required.

```yaml
# when more specific configuration is not provided; PRECEDENCE 1 (lowest)
config:
  connInterfaces:
    - <interface_name>  # e.g. ib0
    - <interface_name>
  connNetFilter:
    - <ip_subnet>  # e.g. 10.0.0.1/24
    - <ip_subnet>
  connTcpOnlyFilter:
    - <ip_subnet>  # e.g. 10.0.0.1/24
    - <ip_subnet>
  beegfsClientConf:
    <beegfs-client.conf_key>: <beegfs-client.conf_value>  
    # e.g. connMgmtdPortTCP: 9008
    # SEE BELOW FOR RESTRICTIONS

fileSystemSpecificConfigs:
    # for a specific filesystem; PRECEDENCE 0 (highest)
  - sysMgmtdHost: <sysMgmtdHost>  # e.g. 10.10.10.1
    config:  # as above

    # for a specific filesystem; PRECEDENCE 0 (highest)
  - sysmMgmtdHost: <sysMgmtdHost>  # e.g. 10.10.10.1
    config:  # as above
    
```

For security purposes, the contents of BeeGFS connAuthFiles are stored in a
separate file. This file is optional, and should only be
used if the connAuthFile configuration option is used on a file system's
other services.

```yaml
- sysMgmtdHost: <sysMgmtdHost>  # e.g. 10.10.10.1
  connAuth: <some_secret_value>
- sysMgmtdHost: <sysMgmtdHost>
  connAuth: <some_secret_value>
```

## Kubernetes Configuration

When deployed into Kubernetes, a single Kubernetes ConfigMap contains the
configuration for all Kubernetes nodes. The ConfigMap includes the same
information as the configuration file above, with the addition of
the *nodeSpecificConfigs* sections. These more specific sections can override 
values specified (or not specified) in a more general section. When the driver 
starts up on a node, it uses the node's name to filter the global ConfigMap 
down to the node-specific configuration defined in 
[General Configuration](#general-configuration). In later versions,
[matchExpressions-based node label 
matching](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
may also be available.

The order of precedence for configuration option overrides is described by
"PRECEDENCE" comments in the example below. In general, precedence is as
follows: default < file system < node < file system AND node. When conflicts
occur between configurations of equal precedence, configuration set lower in the
file takes precedence over configuration set higher in the file.

NOTE: All configuration, and in particular *fileSystemSpecificConfigs* and
*nodeSpecificConfigs* configuration is OPTIONAL! In many situations, only the 
outermost *config* is required.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: beegfs-csi-config
data:
  beegfsCSI: |
    # when more specific configuration is not provided; PRECEDENCE 3 (lowest)
    config:
      connInterfaces:
        - <interface_name>  # e.g. ib0
        - <interface_name>
      connNetFilter:
        - <ip_subnet>  # e.g. 10.0.0.1/24
        - <ip_subnet>
      connTcpOnlyFilter:
        - <ip_subnet>  # e.g. 10.0.0.1/24
        - <ip_subnet>
      beegfsClientConf:
        <beegfs-client.conf_key>: <beegfs-client.conf_value>  
        # e.g. connMgmtdPortTCP: 9008
        # SEE BELOW FOR RESTRICTIONS
    
    fileSystemSpecificConfigs:  # OPTIONAL
        # for a specific filesystem; PRECEDENCE 2
      - sysMgmtdHost: <sysMgmtdHost>  # e.g. 10.10.10.1
        config:  # as above

        # for a specific filesystem; PRECEDENCE 2
      - sysMgmtdHost: <sysMgmtdHost>  # e.g. 10.10.10.1
        config:  # as above
  
    nodeSpecificConfigs:  # OPTIONAL
      - nodeList:
          - <node_name>  # e.g. node1
          - <node_name>
        # matchExpressions:  may be supported in >v1.0
        # default for a specific set of nodes; PRECEDENCE 1
        config:  # as above:
        # for a specific node AND filesystem; PRECEDENCE 0 (highest)
        fileSystemSpecificConfigs:  # as above

      - nodeList:
          - <node_name>  # e.g. node1
          - <node_name>
        # matchExpressions:  may be supported in >v1.0
        # default for a specific set of nodes; PRECEDENCE 1
        config:  # as above:
        # for a specific node AND filesystem; PRECEDENCE 0 (highest)
        fileSystemSpecificConfigs:  # as above
```

For security purposes, the contents of BeeGFS connAuthFiles are stored in a
separate Kubernetes Secret object. This file is optional, and should only be
deployed if the connAuthFile configuration option is used when configuring a
file system's other services.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: beegfs-csi-secret
data:
  connAuths: |
    - sysMgmtdHost: <sysMgmtdHost>  # e.g. 10.10.10.1
      connAuth: <some_secret_value>
    - sysMgmtdHost: <sysMgmtdHost>
      connAuth: <some_secret_value>
```

## beeGFSClientConf Parameters

The following beegfs-client.conf parameters appear in the BeeGFS v7.2
[beegfs-client.conf
file](https://git.beegfs.io/pub/v7/-/blob/7.2/client_module/build/dist/etc/beegfs-client.conf).
Other parameters may exist for newer or older BeeGFS versions. The list a
parameter falls under determines its level of support in the driver.

### No Effect

These parameters are specified elsewhere (a Kubernetes StorageClass, etc.) and
have no effect when specified in the beeGFSClientConf configuration section.

* sysMgmtdHost (specified in a *fileSystemSpecificConfigs[i]* or by the volume
  definition itself)
* connClientPortUDP (semi-random to allow multiple filesystem mounts)

### Unsupported

These parameters are specified elsewhere and may exhibit undocumented behavior
if specified here.

* connAuthFile
* connInterfacesFile
* connNetFilterFile
* connTcpOnlyFilterFile

### Untested

These parameters SHOULD result in the desired effect but have not been tested.

* connHelperdPortTCP
* connMgmtdPortTCP
* connMgmtdPortUDP
* connPortShift
* connCommRetrySecs
* connFallbackExpirationSecs
* connMaxInternodeNum
* connMaxConcurrentAttempts
* connUseRDMA
* connRDMABufNum
* connRDMABufSize
* connRDMATypeOfService
* logClientID 
* logHelperdIP
* logLevel
* logType
* quotaEnabled
* sysCreateHardlinksAsSymlinks
* sysMountSanityCheckMS
* sysSessionCheckOnClose
* sysSyncOnClose
* sysTargetOfflineTimeoutSecs
* sysUpdateTargetStatesSecs
* sysXAttrsEnabled
* tuneFileCacheType
* tunePreferredMetaFile
* tunePreferredStorageFile
* tuneRemoteFSync
* tuneUseGlobalAppendLocks
* tuneUseGlobalFileLocks
* sysACLsEnabled
