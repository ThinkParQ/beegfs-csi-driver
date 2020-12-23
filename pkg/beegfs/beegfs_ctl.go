package beegfs

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/golang/glog"
)

type beegfsCtlExecutorInterface interface {
	CreateVolume(volName, sysMgmtdHost, clientConfPath, volDirPathBeegfsRoot, volDirBasePathBeegfsRoot string) error
}

type beegfsCtlExecutor struct{}

func (ctlExec *beegfsCtlExecutor) CreateVolume(volName, sysMgmtdHost, clientConfPath, volDirPathBeegfsRoot, volDirBasePathBeegfsRoot string) error {
	// Check if volume already exists.
	_, err := ctlExec.execute(clientConfPath, []string{"--unmounted", "--getentryinfo", volDirPathBeegfsRoot})
	if errors.As(err, &ctlNotExistError{}) {
		// We can't find the volume so we need to create one.
		glog.Infof("Volume %s does not exist under directory %s on BeeGFS instance %s", volName, volDirBasePathBeegfsRoot,
			sysMgmtdHost)

		// Create parent directories if necessary.
		// Create a slice of paths where the first path is the most general and each subsequent path is less general.
		dirsToMake := []string{volDirPathBeegfsRoot}
		for dir := path.Dir(volDirPathBeegfsRoot); dir != "/"; { // path.Dir() returns "." if there is no parent.
			dirsToMake = append([]string{dir}, dirsToMake...) // Prepend so the more general path comes first.
			dir = path.Dir(dir)
		}
		// Starting with the most general path, create all directories required to eventually create mountDirPath.
		for _, dir := range dirsToMake {
			_, err := ctlExec.execute(clientConfPath, []string{"--unmounted", "--createdir", dir})
			if err != nil && !errors.As(err, &ctlExistError{}) {
				// We can't create the volume.
				return fmt.Errorf("cannot create directory with path %s on filesystem %s", dir, sysMgmtdHost)
			}
		}
	} else if err != nil {
		return err
	} else {
		glog.Infof("Volume %s already exists under directory %s on BeeGFS instance %s", volName,
			volDirBasePathBeegfsRoot, sysMgmtdHost)
	}
	return nil
}

// beegfsCtlExec executes arbitrary beegfs-ctl commands like "beegfs-ctl --arg1 --arg2=value". It logs the stdout and
// stderr when running at a high verbosity and returns stdout as a string (as well as any potential errors).
// beegfsCtlExec fails if beegfs-ctl is not on the PATH.
func (*beegfsCtlExecutor) execute(cfgFilePath string, args []string) (stdOut string, err error) {
	args = append([]string{fmt.Sprintf("--cfgFile=%s", cfgFilePath)}, args...)
	cmd := exec.Command("beegfs-ctl", args...)
	glog.Infof("Executing command: %s", cmd.Args)

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	err = cmd.Run()
	stdOutString := stdoutBuffer.String()
	stdErrString := stderrBuffer.String()
	if err != nil {
		fmt.Println(err.Error())
		if strings.Contains(stdErrString, "does not exist") {
			err = newCtlNotExistError(stdOutString, stdErrString)
		} else if strings.Contains(stdErrString, "exists already") {
			err = newCtlExistError(stdOutString, stdErrString)
		} else {
			err = fmt.Errorf("beegfs-ctl failed: %w", err)
		}
	}
	glog.V(5).Infof(stdOutString)
	glog.V(5).Infof(stdErrString)

	return stdOutString, err
}

// ctlNotExistError indicates that beegfs-ctl failed to stat or modify an entry that does not exist.
type ctlNotExistError struct {
	stdOutString string
	stdErrString string
}

func newCtlNotExistError(stdOutString, stdErrString string) ctlNotExistError {
	return ctlNotExistError{stdOutString: stdOutString, stdErrString: stdErrString}
}
func (err ctlNotExistError) Error() string {
	return fmt.Sprintf("beegfs-ctl failed with stdOut: %v and stdErr: %v", err.stdOutString, err.stdErrString)
}

// ctlExistError indicates the beegfs-ctl failed to create an entry that already exists.
type ctlExistError struct {
	stdOutString string
	stdErrString string
}

func newCtlExistError(stdOutString, stdErrString string) ctlExistError {
	return ctlExistError{stdOutString: stdOutString, stdErrString: stdErrString}
}
func (err ctlExistError) Error() string {
	return fmt.Sprintf("beegfs-ctl failed with stdOut: %v and stdErr: %v", err.stdOutString, err.stdErrString)
}

type fakeBeegfsCtlExecutor struct{}

func (*fakeBeegfsCtlExecutor) CreateVolume(volName, sysMgmtdHost, clientConfPath, volDirPathBeegfsRoot, volDirBasePathBeegfsRoot string) error {
	return nil
}
