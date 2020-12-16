package beegfs

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

// beegfsConfig contains all of the custom configuration (above and beyond whatever is in the beegfs-client.conf file)
// associated with a single BeeGFS file system EXCEPT for sysMgmtdHost, which is stored separately.
type beegfsConfig struct {
	ConnInterfaces    []string          `yaml:"connInterfaces"`
	ConnNetFilter     []string          `yaml:"connNetFilter"`
	ConnTcpOnlyFilter []string          `yaml:"connTcpOnlyFilter"`
	BeegfsClientConf  map[string]string `yaml:"beegfsClientConf"`
}

func newBeegfsConfig() *beegfsConfig {
	return &beegfsConfig{
		BeegfsClientConf: make(map[string]string),
	}
}

// fileSystemSpecificConfig associates a beegfsConfig with a sysMgmtdHost.
type fileSystemSpecificConfig struct {
	SysMgmtdHost string       `yaml:"sysMgmtdHost"`
	Config       beegfsConfig `yaml:"config"`
}

// nodeSpecificConfig associates a default beegfsConfig and a list of file system specific configurations with a list
// of nodes.
type nodeSpecificConfig struct {
	NodeList                  []string                   `yaml:"nodeList"`
	DefaultConfig             beegfsConfig               `yaml:"config"`
	FileSystemSpecificConfigs []fileSystemSpecificConfig `yaml:"fileSystemSpecificConfigs"`
}

// pluginConfig contains a default beegfsConfig and a list of file system specific configurations. It is the
// configuration that is maintained for the life of the running plugin. It does NOT contain node specific
// configurations. The plugin creates its pluginConfig on startup by iterating through any  node specific
// configurations and accounting for those that apply to the node it is running on.
type pluginConfig struct {
	DefaultConfig             beegfsConfig               `yaml:"config"`
	FileSystemSpecificConfigs []fileSystemSpecificConfig `yaml:"fileSystemSpecificConfigs"`
}

// pluginConfigFromFile contains a pluginConfig and a list of node specific configurations. It is only used
// intermediately during the configuration file parsing process, as it may contain configurations that do NOT apply to
// the node the plugin is running on.
type pluginConfigFromFile struct {
	pluginConfig        `yaml:",inline"`     // embedded structs must be inlined
	NodeSpecificConfigs []nodeSpecificConfig `yaml:"nodeSpecificConfigs"`
}

// parseConfigFromFile reads the file at the specified path, unmarshalls it into a pluginConfigFromFile, and constructs
// a pluginConfig. It uses nodeID to determine if any node specific configuration applies to the node the plugin is
// running on. If it does, the final pluginConfig contains node specific overrides.
func parseConfigFromFile(path, nodeID string) (pluginConfig, error) {
	var rawConfig pluginConfigFromFile
	var newPluginConfig pluginConfig

	// read and parse configuration file
	// return immediately if an error occurs
	rawConfigBytes, err := fsutil.ReadFile(path)
	if err != nil {
		return pluginConfig{}, fmt.Errorf("failed to read configuration file: %v", err)
	}
	if err := yaml.UnmarshalStrict(rawConfigBytes, &rawConfig); err != nil {
		return pluginConfig{}, fmt.Errorf("failed to unmarshal configuration file: %v", err)
	}

	// start populating newPluginConfig using values directly from rawConfig
	newPluginConfig = pluginConfig{
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
			newPluginConfig.DefaultConfig.overwriteFrom(nodeConfig.DefaultConfig)
			newPluginConfig.FileSystemSpecificConfigs = overwriteFileSystemSpecificConfigs(
				newPluginConfig.FileSystemSpecificConfigs, nodeConfig.FileSystemSpecificConfigs)
		}
	}

	return newPluginConfig, nil
}

// overwriteFileSystemSpecificConfigs looks for FileSystemSpecificConfigs in writeTo and writeFrom with the same
// sysMgmtdHost. When it finds a match, overwriteFileSystemSpecificConfigs ONLY overwrites configuration in writeTo
// that is also defined in writeFrom.
func overwriteFileSystemSpecificConfigs(writeTo, writeFrom []fileSystemSpecificConfig) []fileSystemSpecificConfig {
	for _, writeFromConfig := range writeFrom {
		writeToHadConfig := false
		for i, writeToConfig := range writeTo { // use index to modify writeTo in place
			if writeToConfig.SysMgmtdHost == writeFromConfig.SysMgmtdHost {
				writeToHadConfig = true
				writeTo[i].Config.overwriteFrom(writeFromConfig.Config)
			}
		}
		if writeToHadConfig == false {
			writeTo = append(writeTo, writeFromConfig)
		}
	}
	return writeTo
}

// overwriteFrom ONLY overwrites configuration in the receiving beegfsConfig that is also defined in writeFrom, while
// leaving writeFrom configuration untouched.
func (c *beegfsConfig) overwriteFrom(writeFrom beegfsConfig) {
	if len(writeFrom.ConnInterfaces) != 0 {
		c.ConnInterfaces = make([]string, len(writeFrom.ConnInterfaces))
		copy(c.ConnInterfaces, writeFrom.ConnInterfaces)
	}
	if len(writeFrom.ConnNetFilter) != 0 {
		c.ConnNetFilter = make([]string, len(writeFrom.ConnNetFilter))
		copy(c.ConnNetFilter, writeFrom.ConnNetFilter)
	}
	if len(writeFrom.ConnTcpOnlyFilter) != 0 {
		c.ConnTcpOnlyFilter = make([]string, len(writeFrom.ConnTcpOnlyFilter))
		copy(c.ConnTcpOnlyFilter, writeFrom.ConnTcpOnlyFilter)
	}
	for k, v := range writeFrom.BeegfsClientConf {
		c.BeegfsClientConf[k] = v
	}
}
