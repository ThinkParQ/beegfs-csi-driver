package beegfs

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/kubernetes-csi/csi-test/pkg/sanity"
	"k8s.io/utils/mount"
)

func TestSanity(t *testing.T) {
	sanityDir, err := ioutil.TempDir("", "driver-sanity")
	if err != nil {
		t.Fatal(err)
	}
	dataRoot = path.Join(sanityDir, "csi-data-dir")
	endpoint := "unix://" + sanityDir + "/beegfscsi.sock"
	confTemplatePath := path.Join(sanityDir, "beegfs-client.conf")

	if err := fsutil.WriteFile(confTemplatePath, []byte(TestWriteClientFilesTemplate), 0644); err != nil {
		t.Fatalf("failed to write template beegfs-client.conf: %v", err)
	}

	// Create and run the driver
	driver, err := NewBeegfsDriver("testDriver", "testID", endpoint, "",
		confTemplatePath, false, 100, "v0.1")
	if err != nil {
		t.Fatal(err)
	}
	var mps []mount.MountPoint
	driver.cs.mounter = mount.NewFakeMounter(mps)
	driver.ns.mounter = mount.NewFakeMounter(mps)
	driver.cs.ctlExec = &fakeBeegfsCtlExecutor{}
	go driver.Run()

	// Setup paths for mounting and staging
	mntDir := path.Join(sanityDir, "mnt")
	if err := os.Mkdir(mntDir, 0750); err != nil {
		t.Fatal(err)
	}
	mntStageDir := path.Join(sanityDir, "mnt-stage")
	if err := os.Mkdir(mntStageDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Setup configuration parameters
	reqParams := make(map[string]string)
	reqParams[sysMgmtdHostKey] = "localhost"
	reqParams[volDirBasePathKey] = "unittest"
	cfg := &sanity.Config{
		StagingPath:          mntStageDir,
		TargetPath:           mntDir,
		Address:              endpoint,
		TestVolumeParameters: reqParams,
	}
	// Run the sanity tests
	sanity.Test(t, cfg)
	// Cleanup
	if err := os.RemoveAll(sanityDir); err != nil {
		t.Fatal(err)
	}
}
