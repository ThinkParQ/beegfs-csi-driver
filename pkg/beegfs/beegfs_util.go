package beegfs

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/golang/glog"
)

const beegfsNewConfPath = "/etc/beegfs/"                 // Default where conf files for each BeeGFS instance will be created/updated (may be overridden).
const beegfsDefaultClientConfPath = "/etc/beegfs/"       // A path where a BeeGFS conf file used to create new conf files exists.
const beegfsDefaultClientConfFile = "beegfs-client.conf" // The name of the file in the above path copied to create conf files for each BeeGFS instance.

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

func getBeegfsConfValueFromParams(beegfsConfKey string, params map[string]string) (sysMgmtdHost string, ok bool) {
	for key, value := range params {
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

// generateBeeGFSClientConf generates <beegfsConf/sysMgmtdHost>_beegfs-client.conf files.
// Requires a map including at minimum a beegfsConf/sysMgmtdHost entry.
// Optionally the map can include additional beegfsConf/* entries corresponding to keys in beegfs-client.conf.
// 	Entries that do not correspond to keys in beegfs-client.conf are ignored.
//	Entries not prefixed with beegfsConf/ are ignored.
// Requires a confPath string corresponding with the location to generate new beegfs-client.conf files.
// 	If this is set to "" the default specified by beegfsNewConfPath will be used.
// Returns the path the the configuration file (if created/updated), a boolean indicating if changes were made, and an error or nil.
func generateBeeGFSClientConf(params map[string]string, confPath string) (string, bool, error) {

	changed := false

	if _, ok := params["beegfsConf/sysMgmtdHost"]; !ok {
		return "", false, fmt.Errorf("required parameter beegfsConf/sysMgmtdHost was not included in params")
	}

	if confPath == "" {
		confPath = beegfsNewConfPath
	}

	params = getParsedClientParams(params)

	requestedConfPath := confPath + strings.Replace(params["sysMgmtdHost"], ".", "_", 3) + "_" + beegfsDefaultClientConfFile
	defaultConfPath := beegfsDefaultClientConfPath + beegfsDefaultClientConfFile

	glog.Infof("Checking for existing configuration file at %s.", requestedConfPath)
	_, err := os.Stat(requestedConfPath)

	// (jmccormi) If stat returned an error we likely need to create the file.
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") != true {
			return "", changed, fmt.Errorf("unknown error occured trying to read %s", requestedConfPath)
		}

		defaultFile, err := os.Open(defaultConfPath)
		if err != nil {
			return "", changed, fmt.Errorf("A beegfs-client.conf file was not found in /etc/beegfs. Are beegfs-client, beegfs-helperd, and beegfs-utils installed?")
		}
		defer defaultFile.Close()

		newFile, err := os.Create(requestedConfPath)
		if err != nil {
			return "", changed, fmt.Errorf("an unknown error occured trying to write to %s", requestedConfPath)
		}
		defer newFile.Close()

		_, err = io.Copy(newFile, defaultFile)
		if err != nil {
			return "", changed, fmt.Errorf("an unknown error occured copying from %s to %s", defaultConfPath, requestedConfPath)
		}

		err = newFile.Sync()
		if err != nil {
			return "", changed, fmt.Errorf("an unknown error occured flushing %s to disk", requestedConfPath)
		}

		glog.Infof("Created new configuration file at %s.", requestedConfPath)
	} else {
		glog.Infof("Found existing configuration file at %s, checking if any updates are required.", requestedConfPath)
	}

	fileContents, err := ioutil.ReadFile(requestedConfPath)
	if err != nil {
		return "", changed, fmt.Errorf("unknown error reading %s file to check for required updates", requestedConfPath)
	}

	fileLines := strings.Split(string(fileContents), "\n")

	for i, line := range fileLines {

		// (jmccormi) For each uncommented key=value in the .conf file:
		if !strings.Contains(line, "#") && strings.Contains(line, "=") {

			// (jmccormi) Parse out just the key and value:
			parsedLine := strings.Split(string(line), "=")

			// (jmccormi) Check if we were passed a param corresponding with the key in beegfs-client.conf:
			if param, ok := params[strings.TrimSpace(parsedLine[0])]; ok {

				// (jmccormi) Check if the param value differs from the value in beegfs-client.conf:
				if param != strings.TrimSpace(parsedLine[1]) {
					fileLines[i] = parsedLine[0] + "= " + param
					changed = true
				}
			}
		}
	}

	// (jmccormi) If any of the fileLines changed, overwrite the file with the new version:
	if changed == true {
		glog.Infof("Attempting to write updates to %s.", requestedConfPath)
		output := strings.Join(fileLines, "\n")
		err = ioutil.WriteFile(requestedConfPath, []byte(output), 0644)
		if err != nil {
			// (jmccormi) Technically at this point we have no way of knowing if the file on disk changed but we still return changed true.
			return "", changed, fmt.Errorf("Error writing updates to " + requestedConfPath)
		}
		glog.Infof("Successfully wrote updates to %s.", requestedConfPath)
	} else {
		glog.Infof("No updates required to %s.", requestedConfPath)
	}

	return requestedConfPath, changed, nil
}

// getParsedClientParams removes the key prefix "beegfsConf/" from all items in a provided map and returns a map of only those items.
func getParsedClientParams(params map[string]string) map[string]string {

	clientParams := make(map[string]string)

	for k, v := range params {
		if strings.HasPrefix(k, "beegfsConf/") {
			clientParams[strings.TrimPrefix(k, "beegfsConf/")] = v
		}
	}

	return clientParams
}
