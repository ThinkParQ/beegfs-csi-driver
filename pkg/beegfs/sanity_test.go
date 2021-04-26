/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package beegfs

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/kubernetes-csi/csi-test/v4/pkg/sanity"
	"github.com/onsi/ginkgo/config"
	"k8s.io/utils/mount"
)

func TestSanity(t *testing.T) {
	config.DefaultReporterConfig.NoColor = true
	sanityDir, err := ioutil.TempDir("", "driver-sanity")
	if err != nil {
		t.Fatal(err)
	}
	csDataDirPath := path.Join(sanityDir, "csi-data-dir")
	endpoint := "unix://" + sanityDir + "/beegfscsi.sock"
	clientConfTemplatePath := path.Join(sanityDir, "beegfs-client.conf")

	if err := fsutil.WriteFile(clientConfTemplatePath, []byte(TestWriteClientFilesTemplate), 0644); err != nil {
		t.Fatalf("failed to write template beegfs-client.conf: %v", err)
	}

	// Create and run the driver
	driver, err := NewBeegfsDriver("", "", csDataDirPath, "testDriver", endpoint, "testID", clientConfTemplatePath, "v0.1")
	if err != nil {
		t.Fatal(err)
	}
	var mps []mount.MountPoint
	driver.cs.mounter = mount.NewFakeMounter(mps)
	driver.ns.mounter = mount.NewFakeMounter(mps)
	driver.cs.ctlExec = &fakeBeegfsCtlExecutor{}
	driver.ns.ctlExec = &fakeBeegfsCtlExecutor{}
	go driver.Run()

	// Setup configuration parameters
	reqParams := make(map[string]string)
	reqParams[sysMgmtdHostKey] = "localhost"
	reqParams[volDirBasePathKey] = "unittest"
	cfg := sanity.NewTestConfig()
	cfg.StagingPath = path.Join(sanityDir, "mnt-stage")
	cfg.TargetPath = path.Join(sanityDir, "mnt")
	cfg.Address = endpoint
	cfg.TestVolumeParameters = reqParams
	// Run the sanity tests
	sanity.Test(t, cfg)
	// Cleanup
	if err := os.RemoveAll(sanityDir); err != nil {
		t.Fatal(err)
	}
}
