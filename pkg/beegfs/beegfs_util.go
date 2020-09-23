package beegfs

import (
	"bytes"
	"fmt"
	"github.com/golang/glog"
	"net/url"
	"os/exec"
	"strings"
)

const beegfsUrlScheme = "beegfs"

// newBeegfsUrl converts a hostname or IP address and path into a URL
func newBeegfsUrl(host string, path string) string {
	structURL := url.URL{
		Scheme: beegfsUrlScheme,
		Host:   host,
		Path:   path,
	}
	return structURL.String()
}

// parseBeegfsUrl parses a URL and returns a hostname and path
func parseBeegfsUrl(rawUrl string) (host string, path string, err error) {
	var structUrl *url.URL
	if structUrl, err = url.Parse(rawUrl); err != nil {
		return "", "", err
	}
	if structUrl.Scheme != beegfsUrlScheme {
		return "", "", fmt.Errorf("URL has incorrect scheme")
	}
	// TODO(webere) more checks for bad values
	return structUrl.Host, structUrl.Path, nil
}

func getBeegfsConfValueFromParams(beegfsConfKey string, params map[string]string) (sysMgmtdHost string, ok bool){
	for key, value := range(params) {
		key = strings.TrimPrefix(key, beegfsConfPrefix)
		if key == beegfsConfKey {
			return value, true
		}
	}
	return "", false
}

func beegfsCtlExec(cfgFilePath string, args []string) (stdOut string, err error) {
	args = append([]string{fmt.Sprintf("--cfgFile=%s", cfgFilePath)}, args...)
	cmd := exec.Command("/bin/beegfs-ctl", args...)

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	err = cmd.Run()
	glog.Info(stderrBuffer.String())
	glog.Info(stdoutBuffer.String())

	return stdoutBuffer.String(), err
}
