/*
Copyright 2021 NetApp, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

//+kubebuilder:validation:Optional

package v1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BeegfsDriverSpec defines the desired state of BeegfsDriver
type BeegfsDriverSpec struct {
	// TODO(webere, A259): Add additional fields.

	ContainerImageOverrides ContainerImageOverrides `json:"containerImageOverrides,omitempty"`
	//+kubebuilder:validation:Minimum:=0
	//+kubebuilder:validation:Maximum:=5
	// The logging level of deployed containers expressed as an integer from 0 (low detail) to 5 (high detail). Empty
	// results in the BeeGFS CSI driver project default.
	LogLevel             *int                 `json:"logLevel,omitempty"` // Pointer per https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#optional-vs-required.
	PluginConfigFromFile PluginConfigFromFile `json:"pluginConfig,omitempty"`
}

// BeegfsDriverStatus defines the observed state of BeegfsDriver
type BeegfsDriverStatus struct {
	//+operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []metav1.Condition `json:"conditions"`
}

const (
	// Possible values for BeegfsDriverStatus.Conditions[].Type.
	ConditionControllerServiceReady = "ControllerServiceReady"
	ConditionNodeServiceReady       = "NodeServiceReady"

	// Possible values for BeegfsDriverStatus.Conditions[].Reason.
	ReasonServiceNotCreated = "ServiceNotCreated"
	ReasonPodsNotScheduled  = "PodsNotScheduled"
	ReasonPodsNotReady      = "PodsNotReady"
	ReasonPodsReady         = "PodsReady"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+operator-sdk:csv:customresourcedefinitions:displayName="BeeGFS Driver"
//+operator-sdk:csv:customresourcedefinitions:resources={{ConfigMap,v1,},{DaemonSet,v1,},{Secret,v1,},{StatefulSet,v1,}}

// BeegfsDriver is the Schema for the beegfsdrivers API
type BeegfsDriver struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BeegfsDriverSpec   `json:"spec,omitempty"`
	Status BeegfsDriverStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BeegfsDriverList contains a list of BeegfsDriver
type BeegfsDriverList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BeegfsDriver `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BeegfsDriver{}, &BeegfsDriverList{})
}

// A structure that allows for default container images and tags to be overridden. Use it in air-gapped networks,
// networks with private registry mirrors, or to pin a particular container version. Unless otherwise noted, versions
// other than the default are not supported.
type ContainerImageOverrides struct {
	// Defaults to docker.io/netapp/beegfs-csi-driver:<the operator version>.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="BeeGFS CSI Driver"
	BeegfsCsiDriver ContainerImageOverride `json:"beegfsCsiDriver"`
	// Defaults to k8s.gcr.io/sig-storage/csi-node-driver-registrar:<the most current version at operator release>.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="CSI Node Driver Registrar"
	CsiNodeDriverRegistrar ContainerImageOverride `json:"csiNodeDriverRegistrar"`
	// Defaults to k8s.gcr.io/sig-storage/csi-provisioner:<the most current version at operator release>.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="CSI Provisioner"
	CsiProvisioner ContainerImageOverride `json:"csiProvisioner"`
	// Defaults to k8s.gcr.io/sig-storage/livenessprobe:<the most current version at operator release>.
	LivenessProbe ContainerImageOverride `json:"livenessProbe"`
}

type ContainerImageOverride struct {
	// A combination of registry and image (e.g. k8s.gcr.io/csi-provisioner or docker.io/netapp/beegfs-csi-driver).
	Image string `json:"image"`
	// A tag (e.g. v2.2.2 or latest).
	Tag string `json:"tag"`
}

// The primary configuration structure containing all of the custom configuration (beegfs-client.conf keys/values and
// additional CSI driver specific fields) associated with a single BeeGFS file system except for sysMgmtdHost, which is
// specified elsewhere.
type BeegfsConfig struct {
	// A list of interfaces the BeeGFS client service can communicate over (e.g. "ib0" or "eth0"). Often not required.
	// See beegfs-client.conf for more details.
	ConnInterfaces []string `yaml:"connInterfaces" json:"connInterfaces"`
	// A list of subnets the BeeGFS client service can use for outgoing communication (e.g. "10.10.10.10/24"). Often
	// not required. See beegfs-client.conf for more details.
	ConnNetFilter []string `yaml:"connNetFilter" json:"connNetFilter"`
	// A list of subnets in which RDMA communication can/should not be established (e.g. "10.10.10.11/24"). Often not
	// required. See beegfs-client.conf for more details.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Conn TCP Only Filter"
	ConnTcpOnlyFilter []string `yaml:"connTcpOnlyFilter" json:"connTcpOnlyFilter"`
	// A map of additional key value pairs matching key value pairs in the beegfs-client.conf file. See
	// beegfs-client.conf for more details.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Additional beegfs-client.conf Parameters"
	BeegfsClientConf map[string]string `yaml:"beegfsClientConf" json:"beegfsClientConf"`
	ConnAuth         string            `yaml:"-" json:"-"` // Do not support unmarshalling from a configuration file.
}

func NewBeegfsConfig() *BeegfsConfig {
	return &BeegfsConfig{
		BeegfsClientConf: make(map[string]string),
	}
}

// MarshalJSON overrides the default JSON encoding for the BeegfsConfig struct. klogr uses JSON encoding to log
// struct values and thus implicitly calls this method. BeegfsConfig does not support marshalling the ConnAuth field,
// so MarshalJSON encodes a new anonymous struct that includes an marshalled connAuth field and replaces it's value
// with "******" if it is not empty.
func (c BeegfsConfig) MarshalJSON() ([]byte, error) {
	var connAuthString string
	if c.ConnAuth != "" {
		connAuthString = "******"
	}

	// See https://blog.gopheracademy.com/advent-2016/advanced-encoding-decoding/ for more context on how this works.
	type beegfsConfigAlias BeegfsConfig // Use an alias to avoid an infinite loop and a stack overflow.
	return json.Marshal(&struct {
		// Use omitempty to avoid logging in "impossible" locations like DefaultConfig.
		ConnAuth          string `json:"connAuth,omitempty"`
		beegfsConfigAlias        // Embed the BeegfsConfig type to avoid retyping all of its fields.
	}{
		ConnAuth:          connAuthString,
		beegfsConfigAlias: beegfsConfigAlias(c),
	})
}

// A file system specific configuration that overrides the default configuration for a specific file system.
type FileSystemSpecificConfig struct {
	// The sysMgmtdHost used by the BeeGFS client service to make initial contact with the BeeGFS mgmtd service.
	//+kubebuilder:validation:Required
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="SysMgmtdHost"
	SysMgmtdHost string `yaml:"sysMgmtdHost" json:"sysMgmtdHost"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="File System Specific Config"
	Config BeegfsConfig `yaml:"config" json:"config"`
}

// A node specific configuration that overrides file system specific configurations and the default configuration on
// specific nodes.
type NodeSpecificConfig struct {
	// The list of nodes this configuration should be applied on. Each entry is the hostname of the node or the name
	// assigned to the node by the container orchestrator (e.g. "node1" or "cluster05-node03").
	//+kubebuilder:validation:Required
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Names"
	NodeList []string `yaml:"nodeList" json:"nodeList"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Default Config for Nodes"
	DefaultConfig BeegfsConfig `yaml:"config" json:"config"`
	// A list of file system specific configurations that override the default configuration for specific file systems
	// on these nodes.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="File System Specific Configs for Nodes"
	FileSystemSpecificConfigs []FileSystemSpecificConfig `yaml:"fileSystemSpecificConfigs" json:"fileSystemSpecificConfigs"`
}

// The configuration structure containing default configuration (applied to all file systems on all nodes) and file
// system specific configuration. On initialization, the driver squashes all node specific configuration for the node
// it is running on into this structure and maintains it until restart.
type PluginConfig struct {
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Default Config"
	DefaultConfig BeegfsConfig `yaml:"config" json:"config"`
	// A list of file system specific configurations that override the default configuration for specific file systems.
	//+operator-sdk:csv:customresourcedefinitions:displayName="Default File System Specific Configs"
	FileSystemSpecificConfigs []FileSystemSpecificConfig `yaml:"fileSystemSpecificConfigs" json:"fileSystemSpecificConfigs"`
}

// The top level configuration structure containing default configuration (applied to all file systems on all nodes),
// file system specific configuration, and node specific configuration. Fields from node and file system specific
// configurations override fields from the default configuration. Often not required.
type PluginConfigFromFile struct {
	PluginConfig `yaml:",inline" json:",inline"` // embedded structs must be inlined
	// A list of node specific configurations that override file system specific configurations and the default
	// configuration on specific nodes.
	NodeSpecificConfigs []NodeSpecificConfig `yaml:"nodeSpecificConfigs" json:"nodeSpecificConfigs"`
}

// ConnAuthConfig associates a ConnAuth with a SysMgmtdHost.
type ConnAuthConfig struct {
	SysMgmtdHost string `yaml:"sysMgmtdHost" json:"sysMgmtdHost"`
	ConnAuth     string `yaml:"connAuth" json:"connAuth"`
}

// MarshalJSON overrides the default JSON encoding for the ConnAuthConfig struct. klogr uses JSON encoding to log
// struct values and thus implicitly calls this method. MarshalJSON replaces ConnAuthConfig.ConnAuth with "******" if
// it is not empty.
func (c ConnAuthConfig) MarshalJSON() ([]byte, error) {
	var connAuthString string
	if c.ConnAuth != "" {
		connAuthString = "******"
	}

	// See https://blog.gopheracademy.com/advent-2016/advanced-encoding-decoding/ for more context on how this works.
	type connAuthConfigAlias ConnAuthConfig // Use an alias to avoid an infinite loop and a stack overflow.
	return json.Marshal(connAuthConfigAlias{SysMgmtdHost: c.SysMgmtdHost, ConnAuth: connAuthString})
}
