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
	"github.com/spf13/afero"
)

func TestSanity(t *testing.T) {
	fs = afero.NewOsFs() // Make sure we are using an OS-backed file system.
	fsutil = afero.Afero{Fs: fs}

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

	// Create and run the driver.
	driver, err := NewBeegfsDriverSanity("", "", csDataDirPath, "testDriver", endpoint, "testID",
		clientConfTemplatePath, "v0.1", 10)
	if err != nil {
		t.Fatal(err)
	}
	go driver.Run()

	// Set up configuration parameters.
	reqParams := make(map[string]string)
	reqParams[sysMgmtdHostKey] = "localhost"
	reqParams[volDirBasePathKey] = "unittest"
	cfg := sanity.NewTestConfig()
	cfg.StagingPath = path.Join(sanityDir, "mnt-stage")
	cfg.TargetPath = path.Join(sanityDir, "mnt")
	cfg.Address = endpoint
	cfg.TestVolumeParameters = reqParams
	// Run the sanity tests.
	sanity.Test(t, cfg)
	// Do cleanup.
	if err := os.RemoveAll(sanityDir); err != nil {
		t.Fatal(err)
	}
}
