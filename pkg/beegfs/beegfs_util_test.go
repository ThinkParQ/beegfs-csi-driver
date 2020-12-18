package beegfs

import (
	"path"
	"testing"

	"github.com/spf13/afero"
)

// included here as a constant for formatting reasons (string literal looks better with no indentation involved)
const TestWriteClientFilesTemplate = `# A minimal configuration file that allows connInterfaces, connNetFilter, 
# connTcpOnlyFilter, and one arbitrary override.
sysMgmtdHost          =
connClientPortUDP     =
# One arbitrary key that can be overridden.
connMgmtdPortTCP      = 8008
connInterfacesFile    =
connNetFilterFile     =
connTcpOnlyFilterFile =
`

// do not remove extra newline at end of file; go-ini writes one that we must match
const TestWriteClientFilesBeegfsClientConf = `# A minimal configuration file that allows connInterfaces, connNetFilter,
# connTcpOnlyFilter, and one arbitrary override.
sysMgmtdHost          = 127.0.0.1
connClientPortUDP     = 49152
# One arbitrary key that can be overridden.
connMgmtdPortTCP      = 8000
connInterfacesFile    = /testvol/connInterfacesFile
connNetFilterFile     = /testvol/connNetFilterFile
connTcpOnlyFilterFile = /testvol/connTcpOnlyFilterFile

`

func TestWriteClientFiles(t *testing.T) {
	fs = afero.NewMemMapFs() // test sets up its own, new, memory-mapped file system
	fsutil = afero.Afero{Fs: fs}
	confTemplateDirPath := "/etc/beegfs"
	confTemplatePath := path.Join(confTemplateDirPath, "beegfs-client.conf")
	confDirPath := "/testvol"
	sysMgmtdHost := "127.0.0.1"
	testConfig := beegfsConfig{
		ConnInterfaces:    []string{"ib0"},
		ConnNetFilter:     []string{"127.0.0.0/24"},
		ConnTcpOnlyFilter: []string{"127.0.0.0"},
		BeegfsClientConf: map[string]string{
			"connMgmtdPortTCP": "8000",
		},
	}
	wantConnInterfacesFile := "ib0\n"          // desired connInterfacesFile contents
	wantConnNetFilterFile := "127.0.0.0/24\n"  // desired connNetFilterFile contents
	wantConnTcpOnlyFilterFile := "127.0.0.0\n" // desired connTcpOnlyFilterFile contents

	// set up template directory in memory-mapped filesystem
	if err := fs.MkdirAll(confTemplateDirPath, 0755); err != nil {
		t.Fatalf("failed to set up configuration template directory: %v", err)
	}
	if err := fsutil.WriteFile(confTemplatePath, []byte(TestWriteClientFilesTemplate), 0644); err != nil {
		t.Fatalf("failed to write template beegfs-client.conf: %v", err)
	}

	// set up conf directory in memory-mapped filesystem
	if err := fs.Mkdir(confDirPath, 0755); err != nil {
		t.Fatalf("failed to set up new configuration directory: %v", err)
	}

	if _, _, err := writeClientFiles(sysMgmtdHost, confDirPath, confTemplatePath, testConfig); err != nil {
		t.Fatalf("expected no error to occur: %v", err)
	}

	// check written beegfs-client.conf
	got, err := fsutil.ReadFile(path.Join(confDirPath, "beegfs-client.conf"))
	if err != nil {
		t.Errorf("could not read output beegfs-client.conf")
	}
	if TestWriteClientFilesBeegfsClientConf != string(got) {
		t.Errorf("beegfs-client.conf does not match; expected:\n%vgot:\n%v",
			TestWriteClientFilesBeegfsClientConf, string(got))
	}

	// check written connInterfacesFile
	got, err = fsutil.ReadFile(path.Join(confDirPath, "connInterfacesFile"))
	if err != nil {
		t.Errorf("could not read output connInterfacesFile")
	}
	if wantConnInterfacesFile != string(got) {
		t.Errorf("connInterfacesFile does not match; expected:\n%vgot:\n%v",
			wantConnInterfacesFile, string(got))
	}

	// check written connNetFilterFile
	got, err = fsutil.ReadFile(path.Join(confDirPath, "connNetFilterFile"))
	if err != nil {
		t.Errorf("could not read output connNetFilterFile")
	}
	if wantConnNetFilterFile != string(got) {
		t.Errorf("connNetFilterFile does not match; expected:\n%vgot:\n%v",
			wantConnNetFilterFile, string(got))
	}

	// check written connTcpOnlyFilterFile
	got, err = fsutil.ReadFile(path.Join(confDirPath, "connTcpOnlyFilterFile"))
	if err != nil {
		t.Errorf("could not read output connInterfacesFile")
	}
	if wantConnTcpOnlyFilterFile != string(got) {
		t.Errorf("connTcpOnlyFilterFile does not match; expected:\n%vgot:\n%v",
			wantConnTcpOnlyFilterFile, string(got))
	}
}

func TestSanitizeVolumeID(t *testing.T) {
	tests := map[string]struct {
		provided string
		want     string
	}{
		"basic ip example": {
			provided: "beegfs://127.0.0.1/path/to/volume",
			want:     "127.0.0.1_path_to_volume",
		},
		"basic FQDN example": {
			provided: "beegfs://some.domain.com/path/to/volume",
			want:     "some.domain.com_path_to_volume",
		},
		"example with underscores": {
			provided: "beegfs://some.domain.com/path_with_underscores/to/volume",
			want:     "some.domain.com_path__with__underscores_to_volume",
		},
		"example with too many characters": {
			provided: "beegfs://some.domain.com/lots/of/characters/lots/of/characters/lots/of/characters/" +
				"lots/of/characters/lots/of/characters/lots/of/characters/lots/of/characters/lots/of/characters/" +
				"lots/of/characters/lots/of/characters/lots/of/characters/lots/of/characters/lots/of/characters",
			want: "20d02d3ce23bd842f5a9334f478c87c3f131e51e",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := sanitizeVolumeID(tc.provided)
			if tc.want != got {
				t.Fatalf("expected: %s, got: %s", tc.want, got)
			}
		})
	}
}
