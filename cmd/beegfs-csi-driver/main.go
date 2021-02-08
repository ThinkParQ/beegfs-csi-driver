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
	"github.com/golang/glog"
	"k8s.io/klog/v2"
	"os"
	"path"

	beegfs "github.com/netapp/beegfs-csi-driver/pkg/beegfs"
)

func init() {
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Fatalf("Failed to set glog flag logtostderr=true: %v", err.Error())
	}
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

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	// This block is almost identical to
	// https://github.com/kubernetes/klog/blob/v2.5.0/examples/coexist_glog/coexist_glog.go, which was released under
	// the Apache 2.0 license. The only modification is to the first if statement.
	// Sync the glog and klog flags.
	flag.CommandLine.VisitAll(func(f1 *flag.Flag) {
		f2 := klogFlags.Lookup(f1.Name)
		if f2 != nil && f1.Value.String() != f2.DefValue {
			value := f1.Value.String()
			if err := f2.Value.Set(value); err != nil {
				glog.Fatalf("Failed to set klog flag %s to %s: %s", f2.Name, value, err.Error())
			}
		}
	})
	// end of code taken from klog example

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
		glog.Fatalf("Failed to initialize driver: %s", err.Error()) // exits with code 255
	}
	driver.Run()
}
