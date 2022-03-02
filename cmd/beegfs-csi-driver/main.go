/*
Copyright 2017 The Kubernetes Authors.

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

/*
Modifications Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/netapp/beegfs-csi-driver/pkg/beegfs"
	"k8s.io/klog/v2"
)

var (
	connAuthPath           = flag.String("connauth-path", "", "path to connection authentication file")
	configPath             = flag.String("config-path", "", "path to plugin configuration file")
	csDataDir              = flag.String("cs-data-dir", "/tmp/beegfs-csi-data-dir", "path to directory the controller service uses to store client configuration files and mount file systems")
	driverName             = flag.String("driver-name", "beegfs.csi.netapp.com", "name of the driver")
	endpoint               = flag.String("endpoint", "unix://tmp/csi.sock", "CSI endpoint")
	nodeID                 = flag.String("node-id", "", "node id")
	showVersion            = flag.Bool("version", false, "Show version.")
	clientConfTemplatePath = flag.String("client-conf-template-path", "", "path to template beegfs-client.conf")
	nodeUnstageTimeout     = flag.Uint64("node-unstage-timeout", 0, "seconds DeleteVolume waits for NodeUnstageVolume to complete on all nodes")

	// Set by the build process
	version = ""
)

func main() {
	klog.InitFlags(nil)
	if err := flag.Set("logtostderr", "true"); err != nil {
		beegfs.LogFatal(nil, err, "Failed to set klog flag logtostderr=true")
	}
	flag.Parse()

	if *showVersion {
		baseName := path.Base(os.Args[0])
		fmt.Println(baseName, version)
		return
	}

	handle()
	os.Exit(0)
}

func handle() {
	driver, err := beegfs.NewBeegfsDriver(*connAuthPath, *configPath, *csDataDir, *driverName, *endpoint, *nodeID,
		*clientConfTemplatePath, version, *nodeUnstageTimeout)
	if err != nil {
		beegfs.LogFatal(nil, err, "Failed to initialize driver")
	}
	driver.Run()
}
