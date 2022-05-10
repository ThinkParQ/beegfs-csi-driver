/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package beegfs

// BeeGFS CSI Driver configuration types are defined (by necessity of the operator-sdk manifest generation tools) in
// github.com/netapp/beegfs-csi-driver-operator/api/v1. This file provides additional functionality specific to the
// needs of the driver itself. If any of this functionality becomes useful outside the driver (e.g. in the operator),
// consider moving it.

import (
	"net"
	"regexp"

	beegfsv1 "github.com/netapp/beegfs-csi-driver/operator/api/v1"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

// These parameters have no effect when specified in the beeGFSClientConf configuration section.
var noEffectBeegfsConfOptions = []string{
	"sysMgmtdHost",
	"connClientPortUDP",
	"connPortShift",
}

// These parameters are unsupported when specified in the beeGFSClientConf configuration section.
var unsupportedBeegfsConfOptions = []string{
	"connInterfacesFile",
	"connNetFilterFile",
	"connTcpOnlyFilterFile",
	"connAuthFile",
	"connRDMAInterfacesFile",
}

// parseConfigFromFile reads the file at the specified path, unmarshalls it into a PluginConfigFromFile, and constructs
// a PluginConfig. It uses nodeID to determine if any node specific configuration applies to the node the plugin is
// running on. If it does, the final PluginConfig contains node specific overrides.
func parseConfigFromFile(path, nodeID string) (beegfsv1.PluginConfig, error) {
	var rawConfig beegfsv1.PluginConfigFromFile
	var newPluginConfig beegfsv1.PluginConfig

	// read and parse configuration file
	// return immediately if an error occurs
	rawConfigBytes, err := fsutil.ReadFile(path)
	if err != nil {
		return beegfsv1.PluginConfig{}, errors.Wrap(err, "failed to read configuration file")
	}
	if err := yaml.UnmarshalStrict(rawConfigBytes, &rawConfig); err != nil {
		// This is a "best effort" attempt to add additional context to an unmarshalling error that is likely caused by
		// missing quotes in the beegfsClientConf field. It is generally bad practice to base program logic on
		// "reading" and error string, but here we only add to the error message we write and fall back to simply
		// logging the error as is if anything goes wrong.
		re := ".*cannot unmarshal .* into Go struct field .*\\.beegfsClientConf of type string.*"
		if matched, regexErr := regexp.MatchString(re, err.Error()); regexErr == nil && matched {
			return beegfsv1.PluginConfig{}, errors.Wrap(err, "likely missing quotes around an integer or "+
				"boolean beegfsClientConf value")
		}
		return beegfsv1.PluginConfig{}, errors.Wrap(err, "failed to unmarshal configuration file")
	}
	LogDebug(nil, "Raw configuration parsed", "parsePath", path, "rawConfig", rawConfig)

	// start populating newPluginConfig using values directly from rawConfig
	newPluginConfig = beegfsv1.PluginConfig{
		DefaultConfig:             rawConfig.DefaultConfig,
		FileSystemSpecificConfigs: rawConfig.FileSystemSpecificConfigs,
	}

	// overwrite newPluginConfig with anything found in NodeSpecificConfigs pertaining to this node
	for _, nodeConfig := range rawConfig.NodeSpecificConfigs {
		appliesToNode := false
		for _, nodeName := range nodeConfig.NodeList {
			if nodeID == nodeName {
				appliesToNode = true
				break
			}
		}
		if appliesToNode {
			overWriteBeegfsConfig(&newPluginConfig.DefaultConfig, nodeConfig.DefaultConfig)
			newPluginConfig.FileSystemSpecificConfigs = overwriteFileSystemSpecificConfigs(
				newPluginConfig.FileSystemSpecificConfigs, nodeConfig.FileSystemSpecificConfigs)
		}
	}

	if err := validateConfig(&newPluginConfig); err != nil {
		return newPluginConfig, errors.WithMessage(err, "config validation failed")
	}
	stripConfig(&newPluginConfig)
	LogDebug(nil, "Actual configuration to be applied", "PluginConfig", newPluginConfig)

	return newPluginConfig, nil
}

// parseConnAuthFromFile reads the file at the specified path and modifies the provided PluginConfig so that it
// includes connAuth information.
func parseConnAuthFromFile(path string, newPluginConfig *beegfsv1.PluginConfig) error {
	connAuthConfigs := make([]beegfsv1.ConnAuthConfig, 0)
	rawConnAuthConfigBytes, err := fsutil.ReadFile(path)
	if err != nil {
		return errors.Wrap(err, "failed to read connAuth file")
	}
	if err := yaml.UnmarshalStrict(rawConnAuthConfigBytes, &connAuthConfigs); err != nil {
		return errors.Wrap(err, "failed to unmarshal connAuth file")
	}
	// The connAuthConfig.UnmarshallJSON method makes connAuthConfigs safe for logging.
	LogDebug(nil, "Raw connAuth configuration parsed", "parsePath", path,
		"connAuthConfigs", connAuthConfigs)

	for _, connAuth := range connAuthConfigs {
		foundMatchingConfig := false
		for i, specificConfig := range newPluginConfig.FileSystemSpecificConfigs {
			if connAuth.SysMgmtdHost == specificConfig.SysMgmtdHost {
				newPluginConfig.FileSystemSpecificConfigs[i].Config.ConnAuth = connAuth.ConnAuth
				foundMatchingConfig = true
				break
			}
		}
		if !foundMatchingConfig {
			newSpecificConfig := beegfsv1.FileSystemSpecificConfig{
				SysMgmtdHost: connAuth.SysMgmtdHost,
				Config: beegfsv1.BeegfsConfig{
					ConnAuth: connAuth.ConnAuth,
				},
			}
			newPluginConfig.FileSystemSpecificConfigs = append(newPluginConfig.FileSystemSpecificConfigs, newSpecificConfig)
		}
	}

	// The pluginConfig.UnmashallJSON method makes newPluginConfig safe for logging.
	LogDebug(nil, "Actual configuration to be applied after parsing connAuth configuration",
		"PluginConfig", newPluginConfig)

	return nil
}

// validateConfig checks the basic syntax of assorted fields in a PluginConfig and returns an error if it finds
// something incorrect.
func validateConfig(plConfig *beegfsv1.PluginConfig) error {
	beegfsConfigs := []beegfsv1.BeegfsConfig{plConfig.DefaultConfig}
	// this regex is used to determine whether a given string is a domain name
	domainRegex := regexp.MustCompile("^(?:[_a-z0-9](?:[_a-z0-9-]{0,61}[a-z0-9]\\.)|(?:[0-9]+/[0-9]{2})\\.)+(?:[a-z](?:[a-z0-9-]{0,61}[a-z0-9])?)?$")
	for _, config := range plConfig.FileSystemSpecificConfigs {
		// sysMgmtdHost can be localhost, an IP address, or a domain name. if it is none of these, return an error
		if config.SysMgmtdHost != "localhost" && net.ParseIP(config.SysMgmtdHost) == nil &&
			!domainRegex.MatchString(config.SysMgmtdHost) {
			return errors.Errorf("invalid SysMgmtdHost %s", config.SysMgmtdHost)
		}
		beegfsConfigs = append(beegfsConfigs, config.Config)
	}

	for _, config := range beegfsConfigs {
		for _, filter := range config.ConnNetFilter {
			if _, _, err := net.ParseCIDR(filter); err != nil && net.ParseIP(filter) == nil {
				return errors.Errorf("invalid ConnNetFilter %s", filter)
			}
		}
		for _, filter := range config.ConnTcpOnlyFilter {
			if _, _, err := net.ParseCIDR(filter); err != nil && net.ParseIP(filter) == nil {
				return errors.Errorf("invalid ConnTCPOnlyFilter %s", filter)
			}
		}
	}

	return nil
}

// stripConfig removes any no-effect beegfsConf options from the plugin configuration, logging a warning if any are
// found. It also logs a warning (but does not remove) any unsupported options it finds. See deployment.md for the list
// of no-effect options.
func stripConfig(plConfig *beegfsv1.PluginConfig) {
	beegfsConfigs := []beegfsv1.BeegfsConfig{plConfig.DefaultConfig}
	for _, config := range plConfig.FileSystemSpecificConfigs {
		beegfsConfigs = append(beegfsConfigs, config.Config)
	}
	for _, config := range beegfsConfigs {
		for _, noEffectOption := range noEffectBeegfsConfOptions {
			if val, present := config.BeegfsClientConf[noEffectOption]; present {
				LogDebug(nil, "WARNING: No-effect beegfs configuration option found and removed from config",
					"noEffectOption", noEffectOption, "noEffectValue", val)
				delete(config.BeegfsClientConf, noEffectOption)
			}
		}
		for _, unsupportedOption := range unsupportedBeegfsConfOptions {
			if val, present := config.BeegfsClientConf[unsupportedOption]; present {
				LogDebug(nil, "WARNING: Unsupported beegfs configuration option found and left in config",
					"unsupportedOption", unsupportedOption, "unsupportedValue", val)
			}
		}
	}
}

// overwriteFileSystemSpecificConfigs looks for FileSystemSpecificConfigs in writeTo and writeFrom with the same
// sysMgmtdHost. When it finds a match, overwriteFileSystemSpecificConfigs ONLY overwrites configuration in writeTo
// that is also defined in writeFrom.
func overwriteFileSystemSpecificConfigs(writeTo, writeFrom []beegfsv1.FileSystemSpecificConfig) []beegfsv1.FileSystemSpecificConfig {
	for _, writeFromConfig := range writeFrom {
		writeToHadConfig := false
		for i, writeToConfig := range writeTo { // use index to modify writeTo in place
			if writeToConfig.SysMgmtdHost == writeFromConfig.SysMgmtdHost {
				writeToHadConfig = true
				overWriteBeegfsConfig(&writeTo[i].Config, writeFromConfig.Config)
			}
		}
		if writeToHadConfig == false {
			writeTo = append(writeTo, writeFromConfig)
		}
	}
	return writeTo
}

// overWriteBeegfsConfig ONLY overwrites configuration in the writeTo BeegfsConfig that is also defined in the
// writeFrom BeegfsConfig, while leaving writeFrom untouched.
func overWriteBeegfsConfig(writeTo *beegfsv1.BeegfsConfig, writeFrom beegfsv1.BeegfsConfig) {
	if len(writeFrom.ConnInterfaces) != 0 {
		writeTo.ConnInterfaces = make([]string, len(writeFrom.ConnInterfaces))
		copy(writeTo.ConnInterfaces, writeFrom.ConnInterfaces)
	}
	if len(writeFrom.ConnNetFilter) != 0 {
		writeTo.ConnNetFilter = make([]string, len(writeFrom.ConnNetFilter))
		copy(writeTo.ConnNetFilter, writeFrom.ConnNetFilter)
	}
	if len(writeFrom.ConnTcpOnlyFilter) != 0 {
		writeTo.ConnTcpOnlyFilter = make([]string, len(writeFrom.ConnTcpOnlyFilter))
		copy(writeTo.ConnTcpOnlyFilter, writeFrom.ConnTcpOnlyFilter)
	}
	if len(writeFrom.ConnRDMAInterfaces) != 0 {
		writeTo.ConnRDMAInterfaces = make([]string, len(writeFrom.ConnRDMAInterfaces))
		copy(writeTo.ConnRDMAInterfaces, writeFrom.ConnRDMAInterfaces)
	}
	if writeFrom.ConnAuth != "" {
		writeTo.ConnAuth = writeFrom.ConnAuth
	}
	for k, v := range writeFrom.BeegfsClientConf {
		writeTo.BeegfsClientConf[k] = v
	}
}
