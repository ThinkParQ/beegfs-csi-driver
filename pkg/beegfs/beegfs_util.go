package beegfs

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"k8s.io/utils/mount"
)

const beegfsDefaultMountPath = "/mnt"                         // Default path where BeeGFS instances will be mounted under (may be overridden).
const beegfsMountsConfFile = "/etc/beegfs/beegfs-mounts.conf" // Location of beegfs-mounts.conf.
const beegfsNewConfPath = "/etc/beegfs"                       // Default where conf files for each BeeGFS instance will be created/updated (may be overridden).
const beegfsDefaultClientConfPath = "/etc/beegfs"             // A path where a BeeGFS conf file used to create new conf files exists.
const beegfsDefaultClientConfFile = "beegfs-client.conf"      // The name of the file in the above path copied to create conf files for each BeeGFS instance.

const beegfsUrlScheme = "beegfs"

type beegfsVolStagingTargetPath interface {
	GetVolumeId() string
	GetStagingTargetPath() string
}

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

// beegfsCtlExec executes arbitrary beegfs-ctl commands like "sudo /opt/beegfs/sbin/beegfs-ctl --arg1 --arg2=value". It returns
// the stdout and stderr at a high verbosity, and it returns stdout as a string (as well as any potential errors).
func beegfsCtlExec(cfgFilePath string, args []string) (stdOut string, err error) {
	args = append([]string{fmt.Sprintf("--cfgFile=%s", cfgFilePath)}, args...)
	args = append([]string{"/opt/beegfs/sbin/beegfs-ctl"}, args...)
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

	// (jmccormi) If connClientPortUDP wasn't specified loop through UDP ports in the ephemeral range and find an available port.
	// Note if generateBeeGFSClientConf is rerun against the same confPath with allowOverwrite this always results in the file being updated with a new UDP port.
	// This seemed safer than trying to see if connClientPortUDP was already set to ensure we always mount BeeGFS with a (probably) available UDP port.
	// If BeeGFS is actively mounted it will continue to use the original UDP port (presumably until remounted). 
	if _, ok := params["connClientPortUDP"]; !ok {
		for i := 49152; i < 65535; i++ {
			available, err := isUDPPortAvailable(strconv.Itoa(i))
			if err != nil {
				return "", false, err
			} else if available == true {
				params["connClientPortUDP"] = strconv.Itoa(i)
				break
			}
		}
	}

	requestedConfPath := path.Join(confPath, strings.Replace(params["sysMgmtdHost"], ".", "_", 3)+"_"+beegfsDefaultClientConfFile)
	defaultConfPath := path.Join(beegfsDefaultClientConfPath, beegfsDefaultClientConfFile)

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
		requestedMountPath = fmt.Sprintf("%s_beegfs", path.Join(beegfsDefaultMountPath, requestedMountPath))
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

// mountBeeGFS handles mounting BeeGFS and creating a directory for the mount point (along with any necessary parents).
// Requires a mountUnder string pointing to a parent directory for the BeeGFS mount point.
// 	If this is set to "" it will default to <beegfsDefaultMountPath>/.
// Requires a requestedConfPath string pointing the BeeGFS client conf file for the file system to mount.
// Returns the full path to the BeeGFS mount point (ex. /mnt/192_168_10_13_beegfs).
func mountBeegfs(mountUnder string, requestedConfPath string) (requestedMountPath string, changed bool, err error) {

	changed = false
	requestedMountPath = ""
	beegfsMountOpts := []string{"rw", "relatime", "cfgFile=" + requestedConfPath}

	if mountUnder == "" {
		// (jmccormi) If needed generate a default mount location using the format beegfsDefaultMountPath/<sysMgmtdHost>_beegfs
		mountDir := strings.Split(filepath.Base(requestedConfPath), "_"+beegfsDefaultClientConfFile)[0]
		requestedMountPath = fmt.Sprintf("%s_beegfs", path.Join(beegfsDefaultMountPath, mountDir))
	} else {
		mountDir := strings.Split(filepath.Base(requestedConfPath), "_"+beegfsDefaultClientConfFile)[0]
		requestedMountPath = fmt.Sprintf("%s_beegfs", path.Join(mountUnder, mountDir))
	}

	beegfsMounter := mount.New("/usr/bin/mount")
	// (jmccormi) We can't use this as BeeGFS doesn't meet whatever heuristics IsLikelyNotMountPoint uses to determine if a dir is a mountpoint.
	// alreadyMounted, err := beegfsMounter.IsLikelyNotMountPoint(requestedMountPath)

	currentMounts, err := beegfsMounter.List()
	if err != nil {
		return "", changed, fmt.Errorf("encountered error '%s' while attempting to get the list of currently mounted file systems", err)
	}

	for _, mount := range currentMounts {
		if mount.Type == "beegfs" && mount.Path == requestedMountPath {
			glog.Infof("mountBeegfs: BeeGFS is already mounted to %v.", requestedMountPath)
			return requestedMountPath, changed, nil
		}
	}

	_, err = os.Stat(requestedMountPath)
	if os.IsNotExist(err) {
		glog.Infof("mountBeegfs: requested path to mount BeeGFS doesn't exist, attempting to create directory %v.", requestedMountPath)
		err := os.MkdirAll(requestedMountPath, 0755)
		changed = true
		if err != nil {
			return "", changed, fmt.Errorf("encountered an unknown error '%v' when attempting to create directory %v", err, requestedMountPath)
		}
	} else if err != nil {
		return "", changed, fmt.Errorf("encountered an unknown error '%v' when whem checking if %v is a directory", err, requestedMountPath)
	}

	glog.Infof("mountBeegfs: attempting to mount BeeGFS to %v.", requestedMountPath)
	if err = beegfsMounter.Mount("beegfs_nodev", requestedMountPath, "beegfs", beegfsMountOpts); err != nil {
		if cleanupErr := os.Remove(requestedMountPath); cleanupErr != nil {
			return requestedMountPath, changed, fmt.Errorf("failed to cleanup directory %v after mount failure occured: %v", requestedMountPath, err)
		}
		return requestedMountPath, changed, err
	}
	changed = true
	return requestedMountPath, changed, nil
}

// unmountBeegfsAndCleanUpConf cleans up a globally mounted BeeGFS filesystem ONLY if it is not bind mounted somewhere
// else. This is necessary to avoid trying to unmount a BeeGFS filesystem that is still in use by some container.
// "Cleans up" in this context means unmount the BeeGFS filesystem, delete the mount point (mountPath), and delete the
// configuration file (confPath).
// Requires a mountPath to the global BeeGFS mountpoint (e.g. .../10_113_72_217_beegfs).
// Requires a confPath to a BeeGFS client.conf file (e.g. .../10_113_72_217_beegfs-client.conf).
// Quietly returns WITHOUT error if the BeeGFS filesystem should not be unmounted.
func unmountBeegfsAndCleanUpConf(mountPath string, confPath string) (err error) {
	beegfsMounter := mount.New("/bin/mount")

	// Decide whether or not to unmount BeeGFS filesystem by checking whether it is bind mounted somewhere else.
	safeToUnmount := true
	// Cannot use beegfsMounter.GetRefs() because we are bind mounting subdirectories (e.g. .../10_113_72_217_beegfs is
	// the initial mount point but .../10_113_72_217_beegfs/scratch is the directory we bind mount). beegfsMounter.GetRefs()
	// is incapable of discovering this.
	allMounts, err := beegfsMounter.List()
	if err != nil {
		return err
	}
	for _, entry := range allMounts {
		if entry.Device == "beegfs_nodev" && entry.Path != mountPath {
			for _, opt := range entry.Opts {
				if strings.Contains(opt, confPath) {
					// This is a bind mount of the BeeGFS filesystem mounted at mountPath
					glog.Infof("Refusing to unmount filesystem at %v because it is bind mounted at %v", mountPath, entry.Path)
					safeToUnmount = false
				}
			}
		}
	}

	if safeToUnmount {
		glog.Infof("Unmounting filesystem at %v and removing mountpoint", mountPath)
		if err = mount.CleanupMountPoint(mountPath, beegfsMounter, false); err != nil {
			return err
		}
		glog.Infof("Removing configuration file at %v", confPath)
		if err = os.Remove(confPath); err != nil {
			return err
		}
	}
	return nil
}

// isUDPPortAvailable checks if a port is already listed in "sudo netstat -lu".
// This is a rudimentary implementation that doesn't validate if it was passed a valid port vs. some other string.
// If the port is already listed returns false, if the port is not listed returns true.
func isUDPPortAvailable(port string) (available bool, err error) {

	glog.Infof("Checking for in use UDP port with: sudo netstat -lu")
	cmd, err := exec.Command("sudo", "netstat", "-lu").Output() // use sudo in case we are not root but have sudo privileges

	if err != nil {
		return false, fmt.Errorf("error '%s' checking if UDP port %s is available with netstat -lu", err, port)
	}

	if strings.Contains(string(cmd), port) {
		return false, nil
	}

	return true, err
}


/* 
Determines the unique path within the local root file system for a specific BeeGFS URL / volume ID.

The full volumeStagingTargetPath within the local root filesystem for each BeeGFS volume is determined as follows:
	staging_target_path/
		sysMgmtdHost_beegfs_vols/ // Replacing . with _ if provided an IP address for sysMgmtdHost.
			volPath/			  // The full path to the requested directory within the BeeGFS instance.
				sysMgmtdHost_beegfs/ // The actual BeeGFS mount point for this volume will be created here. 
				sysMgmtdHost_beegfs-client.conf // A corresponding BeeGFS client config file will be created here.

	=== Example ===
	/mnt/
		10_113_72_217_beegfs_vols/
			jmccormi_scratch/jmccormi_test_1/
				10_113_72_217_beegfs/
				10_113_72_217_beegfs-client.conf
*/
func getBeegfsVolStagingTargetPath(req beegfsVolStagingTargetPath) (volumeStagingTargetPath string, err error) {

	sysMgmtdHost, volPath, err := parseBeegfsUrl(req.GetVolumeId())
	if err != nil {
		return "", err
	}

	volumeStagingTargetPath = path.Join(req.GetStagingTargetPath(), strings.Replace(sysMgmtdHost, ".", "_", 3)+"_beegfs_vols", volPath)

	return volumeStagingTargetPath, nil
}