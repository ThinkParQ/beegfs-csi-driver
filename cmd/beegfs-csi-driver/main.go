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

package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	beegfs "github.com/netapp/beegfs-csi-driver/pkg/beegfs"
)

func init() {
	flag.Set("logtostderr", "true")
}

var (
	configPath             = flag.String("config-path", "", "path to plugin configuration file")
	csDataDir              = flag.String("cs-data-dir", "/tmp/beegfs-csi-data-dir", "path to directory the controller service uses to store client configuration files and mount file systems")
	driverName             = flag.String("driver-name", "beegfs.csi.netapp.com", "name of the driver")
	endpoint               = flag.String("endpoint", "unix://tmp/csi.sock", "CSI endpoint")
	nodeID                 = flag.String("node-id", "", "node id")
	showVersion            = flag.Bool("version", false, "Show version.")
	clientConfTemplatePath = flag.String("client-conf-template-path", "/etc/beegfs/beegfs-client.conf", "path to template beegfs-client.conf")

	// Set by the build process
	version = ""
)

func main() {
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
	driver, err := beegfs.NewBeegfsDriver(*configPath, *csDataDir, *driverName, *endpoint, *nodeID, *clientConfTemplatePath, version)
	if err != nil {
		fmt.Printf("Failed to initialize driver: %s", err.Error())
		os.Exit(1)
	}
	driver.Run()
}
