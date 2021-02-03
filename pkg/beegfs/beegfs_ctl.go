/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package beegfs

import (
	"bytes"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

// beegfsCtlExecutorInterface abstracts beegfs-ctl so tests can run without access to a beegfs-ctl binary or a BeeGFS
// file system.
type beegfsCtlExecutorInterface interface {
	createDirectoryForVolume(vol beegfsVolume) error
	statDirectoryForVolume(vol beegfsVolume) (string, error)
	setPatternForVolume(vol beegfsVolume, config stripePatternConfig) error
}

// beegfsCtlExecutor is the standard implementation of beegfsCtlExecutorInterface.
type beegfsCtlExecutor struct{}

// createDirectoryForVolume uses a "beegfs-ctl --createdir" command to create the directory specified by
// vol.volDirPathBeegfsRoot on the BeeGFS file system specified by vol.sysMgmtdHost. createDirectory returns an error
// if it cannot create the directory, but does not return an error if the directory already exists.
func (ctlExec *beegfsCtlExecutor) createDirectoryForVolume(vol beegfsVolume) error {
	glog.V(LogDebug).Infof("Creating directory for volume %s", vol.volumeID)
	// Check if volume already exists.
	_, err := ctlExec.statDirectoryForVolume(vol)
	if errors.As(err, &ctlNotExistError{}) {
		// We can't find the volume so we need to create one.
		glog.V(LogDebug).Infof("Directory %s does not exist on BeeGFS instance %s", vol.volDirPathBeegfsRoot, vol.sysMgmtdHost)

		// Create parent directories if necessary.
		// Create a slice of paths where the first path is the most general and each subsequent path is less general.
		dirsToMake := []string{vol.volDirPathBeegfsRoot}
		for dir := path.Dir(vol.volDirPathBeegfsRoot); dir != "/"; { // path.Dir() returns "." if there is no parent.
			dirsToMake = append([]string{dir}, dirsToMake...) // Prepend so the more general path comes first.
			dir = path.Dir(dir)
		}
		// Starting with the most general path, create all directories required to eventually create vol.volDirPathBeegfsRoot.
		for _, dir := range dirsToMake {
			// todo(eastburj): Consider replacing "--access=0777" with "fsGroup support"[1].
			//   [1](https://kubernetes-csi.github.io/docs/support-fsgroup.html)
			_, err := ctlExec.execute(vol.clientConfPath, []string{"--unmounted", "--createdir", "--access=0777", dir})
			if err != nil && !errors.As(err, &ctlExistError{}) {
				// We can't create the volume.
				return errors.Errorf("cannot create directory %s on BeeGFS instance %s", dir, vol.sysMgmtdHost)
			}
		}
	} else if err != nil {
		return err
	} else {
		glog.V(LogDebug).Infof("Directory %s already exists on BeeGFS instance %s", vol.volDirPathBeegfsRoot, vol.sysMgmtdHost)
	}
	return nil
}

// statDirectoryForVolume returns the information output by "beegfs-ctl --getentryinfo" as a string, or an empty string
// and an error if the stat fails.
func (ctlExec *beegfsCtlExecutor) statDirectoryForVolume(vol beegfsVolume) (string, error) {
	return ctlExec.execute(vol.clientConfPath, []string{"--unmounted", "--getentryinfo", vol.volDirPathBeegfsRoot})
}

// constructSetPatternForVolume builds the arguments passed to setPatternForVolume.
func constructSetPatternForVolume(config stripePatternConfig) ([]string, bool) {
	var needToExecute bool
	var args []string
	if config.stripePatternNumTargets != "" {
		args = append([]string{fmt.Sprintf("--numtargets=%s", config.stripePatternNumTargets)}, args...)
		needToExecute = true
	}
	if config.stripePatternChunkSize != "" {
		args = append([]string{fmt.Sprintf("--chunksize=%s", config.stripePatternChunkSize)}, args...)
		needToExecute = true
	}
	if config.storagePoolID != "" {
		args = append([]string{fmt.Sprintf("--storagepoolid=%s", config.storagePoolID)}, args...)
		needToExecute = true
	}
	if needToExecute {
		args = append([]string{"--unmounted", "--setpattern"}, args...)
		return args, true
	}

	return []string{}, false
}

// setPatternForVolume uses a "beegfs-ctl --unmounted --setpattern" command to set the pattern for a directory specified by
// vol.volDirPathBeegfsRoot on the BeeGFS file system. setPatternForVolume returns an error if it cannot set the pattern for
// the directory, but does not return an error if the pattern on the directory already exists. setPatternForVolume has no
// effect and does not return an error if config is empty.
func (ctlExec *beegfsCtlExecutor) setPatternForVolume(vol beegfsVolume, config stripePatternConfig) error {
	args, needToExecute := constructSetPatternForVolume(config)
	if needToExecute {
		args = append(args, vol.volDirPathBeegfsRoot)
		_, err := ctlExec.execute(vol.clientConfPath, args)
		if err != nil {
			return errors.Errorf("cannot set pattern for directory %s on BeeGFS instance %s", vol.volDirPathBeegfsRoot, vol.sysMgmtdHost)
		}
	}

	return nil
}

// execute runs arbitrary beegfs-ctl commands like "beegfs-ctl --arg1 --arg2=value". It logs the stdout and stderr
// when running at a high verbosity and returns stdout as a string (as well as any potential errors). execute fails if
// beegfs-ctl is not on the PATH.
func (*beegfsCtlExecutor) execute(clientConfPath string, args []string) (stdOut string, err error) {
	args = append([]string{fmt.Sprintf("--cfgFile=%s", clientConfPath)}, args...)
	cmd := exec.Command("beegfs-ctl", args...)
	glog.V(LogDebug).Infof("Executing command: %s", cmd.Args)

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	err = cmd.Run()
	stdOutString := stdoutBuffer.String()
	stdErrString := stderrBuffer.String()
	if err != nil {
		if strings.Contains(stdErrString, "does not exist") {
			err = errors.WithStack(newCtlNotExistError(stdOutString, stdErrString))
		} else if strings.Contains(stdErrString, "exists already") {
			err = errors.WithStack(newCtlExistError(stdOutString, stdErrString))
		} else {
			err = errors.Wrap(err, "beegfs-ctl failed")
		}
	}
	glog.V(LogVerbose).Infof(stdOutString)
	glog.V(LogVerbose).Infof(stdErrString)

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

// fakeBeeGFSCtlExecutor is a mock implementation of beegfsCtlExecutorInterface useful for testing.
type fakeBeegfsCtlExecutor struct{}

func (*fakeBeegfsCtlExecutor) createDirectoryForVolume(vol beegfsVolume) error {
	return nil
}

func (*fakeBeegfsCtlExecutor) statDirectoryForVolume(vol beegfsVolume) (string, error) {
	return "", nil
}

func (*fakeBeegfsCtlExecutor) setPatternForVolume(vol beegfsVolume, config stripePatternConfig) error {
	return nil
}
