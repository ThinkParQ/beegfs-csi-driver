package beegfs

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

const beegfsDefaultMountPath = "/mnt/"                        // Default path where BeeGFS instances will be mounted under (may be overridden).
const beegfsMountsConfFile = "/etc/beegfs/beegfs-mounts.conf" // Location of beegfs-mounts.conf.
const beegfsNewConfPath = "/etc/beegfs/"                      // Default where conf files for each BeeGFS instance will be created/updated (may be overridden).
const beegfsDefaultClientConfPath = "/etc/beegfs/"            // A path where a BeeGFS conf file used to create new conf files exists.
const beegfsDefaultClientConfFile = "beegfs-client.conf"      // The name of the file in the above path copied to create conf files for each BeeGFS instance.

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

// parseBeegfsUrl parses a URL and returns a sysMgmtdHost and path
func parseBeegfsUrl(rawUrl string) (sysMgmtdHost string, path string, err error) {
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

// getBeegfsConfValueFromParams looks through a map[string]string of parameters, many of which are prefaced by beegfsConf/ and
// returns the value associated with a key if it exists in the map as beegfsConf/key. It also returns true if the key is found
// and false otherwise.
func getBeegfsConfValueFromParams(beegfsConfKey string, params map[string]string) (sysMgmtdHost string, ok bool) {
	for key, value := range params {
		key = strings.TrimPrefix(key, beegfsConfPrefix)
		if key == beegfsConfKey {
			return value, true
		}
	}
	return "", false
}

// beegfsCtlExec executes arbitrary beegfs-ctl commands like "sudo /bin/beegfs-ctl --arg1 --arg2=value". It returns the logs
// the stdout and stderr at a high verbosity, and it returns stdout as a string (as well as any potential errors).
func beegfsCtlExec(cfgFilePath string, args []string) (stdOut string, err error) {
	args = append([]string{fmt.Sprintf("--cfgFile=%s", cfgFilePath)}, args...)
	args = append([]string{"/bin/beegfs-ctl"}, args...)
	cmd := exec.Command("sudo", args...) // use sudo in case we are not root but have sudo priveleges
	glog.Infof("Executing command: %s", cmd.Args)

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	err = cmd.Run()
	stdOutString := stdoutBuffer.String()
	stderrString := stderrBuffer.String()
	glog.V(5).Infof(stdOutString)
	glog.V(5).Infof(stderrString)

	return stdOutString, err
}

// generateBeeGFSClientConf generates <beegfsConf/sysMgmtdHost>_beegfs-client.conf files.
// Requires a params map including at minimum a beegfsConf/sysMgmtdHost entry.
// 	Optionally the map can include additional beegfsConf/* entries corresponding to keys in beegfs-client.conf.
// 	Entries that do not correspond to keys in beegfs-client.conf are ignored.
//	Entries not prefixed with beegfsConf/ are ignored.
// Requires a confPath string corresponding with the location to generate new beegfs-client.conf files.
// 	If this is set to "" the default specified by beegfsNewConfPath will be used.
// Requires a boolean indicating whether or not an existing configuration file should be overwritten if found.
// Returns the path to the new/existing/updated configuration file, a boolean indicating if changes were made, and an error or nil.
//  generateBeeGFSClientConf does NOT generate an error if an existing file is found and allowOverwrite is false.
func generateBeeGFSClientConf(params map[string]string, confPath string, allowOverwrite bool) (string, bool, error) {

	changed := false

	if _, ok := params["beegfsConf/sysMgmtdHost"]; !ok {
		return "", false, fmt.Errorf("required parameter beegfsConf/sysMgmtdHost was not included in params")
	}

	if confPath == "" {
		confPath = beegfsNewConfPath
	}

	params = getParsedClientParams(params)

	// (jmccormi) TODO: This is intended to be a temporary workaround for the fact we need a unique connClientPortUDP value for each instance.
	// This is poor implementation as we're picking a port out of the ephemeral port range (49152-65535) without verifying if its already in use.
	// This also won't work if we are provided a hostname or IPv6 instead of an IPv4 address and assumes all possible mgmt IPs are in the same subnet.
	if _, ok := params["connClientPortUDP"]; !ok {
		lastOctet := strings.Split(params["sysMgmtdHost"], ".")[3]
		if len(lastOctet) == 2 {
			params["connClientPortUDP"] = "500" + lastOctet
		} else {
			params["connClientPortUDP"] = "50" + lastOctet
		}
	}

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
		glog.Infof("Found existing configuration file at %s.", requestedConfPath)
	}

	if allowOverwrite {
		glog.Infof("Checking if any updates are required for configuration file at %s.", requestedConfPath)
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
	} else {
		glog.Infof("No overwrites requested by caller for configuration file at %s.", requestedConfPath)
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

// updateBeegfsMountsFile manages entries in the beegfs-mounts.conf file but does not handle actually mounting BeeGFS.
// Requires a requestedMountPath string with the requested path to mount BeeGFS.
// 	If this is set to "" it will default to "beegfsDefaultMountPath/<sysMgmtdHost>_beegfs" (ex. /mnt/10.113.123.124_beegfs).
// Requires a requestedConfPath string with the full path to an existing BeeGFS client configuration file for the file system you wish to add to beegfs-mounts.conf.
// Returns the path where this BeeGFS file system would be mounted, a boolean indicating if changes were made, and an error or nil.
func updateBeegfsMountsFile(requestedMountPath string, requestedConfPath string) (string, bool, error) {

	changed := false

	if requestedMountPath == "" {
		// (jmccormi) Generate a default mount location using the format beegfsDefaultMountPath/<sysMgmtdHost>_beegfs
		requestedMountPath = strings.Split(filepath.Base(requestedConfPath), "_"+beegfsDefaultClientConfFile)[0]
		requestedMountPath = fmt.Sprintf("%s%s_beegfs", beegfsDefaultMountPath, requestedMountPath)
	}

	beegfsMount := fmt.Sprintf("%s %s", strings.TrimSpace(requestedMountPath), strings.TrimSpace(requestedConfPath))

	glog.Infof("updateBeegfsMountsFile: Attempting to read configuration file at %s.", beegfsMountsConfFile)
	fileContents, err := ioutil.ReadFile(beegfsMountsConfFile)
	if err != nil {
		return "", changed, fmt.Errorf("unknown error reading %s file to check for required updates", beegfsMountsConfFile)
	}

	changeRequired := true // (jmccormi) we'll presume a change is required and toggle this to false if we find the beegfsMount in the beegfsMountsConfFile.
	fileLines := strings.Split(string(fileContents), "\n")
	for _, line := range fileLines {
		if line == beegfsMount {
			changeRequired = false
		}
	}

	if changeRequired == true {
		fileLines = append(fileLines, beegfsMount)

		glog.Infof("updateBeegfsMountsFile: Attempting to write updates to %s.", beegfsMountsConfFile)
		output := strings.Join(fileLines, "\n")
		err = ioutil.WriteFile(beegfsMountsConfFile, []byte(output), 0644)
		changed = true
		if err != nil {
			// (jmccormi) Technically at this point we have no way of knowing if the file on disk changed but we still return changed true.
			return "", changed, fmt.Errorf("Error writing updates to " + beegfsMountsConfFile)
		}
		glog.Infof("updateBeegfsMountsFile: Successfully wrote updates to %s.", beegfsMountsConfFile)
	} else {
		glog.Infof("updateBeegfsMountsFile: No changes required to %s.", beegfsMountsConfFile)
	}

	return requestedMountPath, changed, nil
}
