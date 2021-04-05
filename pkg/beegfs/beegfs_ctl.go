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

	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// beegfsCtlExecutorInterface abstracts beegfs-ctl so tests can run without access to a beegfs-ctl binary or a BeeGFS
// file system.
type beegfsCtlExecutorInterface interface {
	createDirectoryForVolume(ctx context.Context, vol beegfsVolume, cfg permissionsConfig) error
	statDirectoryForVolume(ctx context.Context, vol beegfsVolume) (string, error)
	setPatternForVolume(ctx context.Context, vol beegfsVolume, cfg stripePatternConfig) error
}

// beegfsCtlExecutor is the standard implementation of beegfsCtlExecutorInterface.
type beegfsCtlExecutor struct{}

// createDirForVolume uses a "beegfs-ctl --createdir" command to create the directory specified by
// vol.volDirPathBeegfsRoot on the BeeGFS file system specified by vol.sysMgmtdHost. createDirectory returns an error
// if it cannot create the directory, but does not return an error if the directory already exists.
func (ctlExec *beegfsCtlExecutor) createDirectoryForVolume(ctx context.Context, vol beegfsVolume, cfg permissionsConfig) error {
	LogDebug(ctx, "Creating BeeGFS directory", "volDirPathBeegfsRoot", vol.volDirPathBeegfsRoot, "volumeID", vol.volumeID)
	// Check if volume already exists.
	_, err := ctlExec.statDirectoryForVolume(ctx, vol)
	if errors.As(err, &ctlNotExistError{}) {
		// We can't find the volume so we need to create one.
		LogDebug(ctx, "BeeGFS directory does not exist", "volDirPathBeegfsRoot", vol.volDirPathBeegfsRoot, "volumeID", vol.volumeID)

		// Construct the set of arguments that will be used to create any necessary directories.
		createDirArgs := constructCreateDirForVolumeArgs(cfg)

		// Multiple parent directories may need to be created.
		// Create a slice of paths where the first path is the most general and each subsequent path is less general.
		dirsToMake := []string{vol.volDirPathBeegfsRoot}
		for dir := path.Dir(vol.volDirPathBeegfsRoot); dir != "/"; { // path.Dir() returns "." if there is no parent.
			dirsToMake = append([]string{dir}, dirsToMake...) // Prepend so the more general path comes first.
			dir = path.Dir(dir)
		}
		// Starting with the most general path, create all directories required to eventually create vol.volDirPathBeegfsRoot.
		for _, dir := range dirsToMake {
			_, err := ctlExec.execute(ctx, vol.clientConfPath, append(createDirArgs, dir))
			if err != nil && !errors.As(err, &ctlExistError{}) {
				// We can't create the volume.
				return errors.WithMessagef(err, "cannot create BeeGFS directory %s for %s", dir, vol.volumeID)
			}
		}
	} else if err != nil {
		return err
	} else {
		LogDebug(ctx, "BeeGFS directory already exists", "volDirPathBeegfsRoot", vol.volDirPathBeegfsRoot, "volumeID", vol.volumeID)
	}
	return nil
}

// statDirForVolume returns the information output by "beegfs-ctl --getentryinfo" as a string, or an empty string
// and an error if the stat fails.
func (ctlExec *beegfsCtlExecutor) statDirectoryForVolume(ctx context.Context, vol beegfsVolume) (string, error) {
	return ctlExec.execute(ctx, vol.clientConfPath, []string{"--unmounted", "--getentryinfo", vol.volDirPathBeegfsRoot})
}

// constructSetPatternForVolumeArgs constructs the slice of arguments that will be passed to ctlExec.execute() in a
// setPatternForVolume() call. We keep this logic in a separate function for easy testing.
func constructSetPatternForVolumeArgs(cfg stripePatternConfig) ([]string, bool) {
	var needToExecute bool
	var args []string
	if cfg.stripePatternNumTargets != "" {
		args = append([]string{fmt.Sprintf("--numtargets=%s", cfg.stripePatternNumTargets)}, args...)
		needToExecute = true
	}
	if cfg.stripePatternChunkSize != "" {
		args = append([]string{fmt.Sprintf("--chunksize=%s", cfg.stripePatternChunkSize)}, args...)
		needToExecute = true
	}
	if cfg.storagePoolID != "" {
		args = append([]string{fmt.Sprintf("--storagepoolid=%s", cfg.storagePoolID)}, args...)
		needToExecute = true
	}
	if needToExecute {
		args = append([]string{"--unmounted", "--setpattern"}, args...)
		return args, true
	}

	return []string{}, false
}

// constructCreateDirForVolumeArgs constructs the slice of arguments that will be passed to ctlExec.execute() in a
// createDirForVolume() call. We keep this logic in a separate function for easy testing.
func constructCreateDirForVolumeArgs(cfg permissionsConfig) []string {
	args := []string{"--unmounted", "--createdir"}
	// beegfs-ctl ignores special permissions (the first digit in the four digit octal permissions schema).
	// Construct the --access argument without this digit for clarity.
	mode := cfg.mode & 0o777
	args = append(args, fmt.Sprintf("--access=%03o", mode)) // Print 3 digits padded with 0s to the left.
	if cfg.uid != 0 {
		args = append(args, fmt.Sprintf("--uid=%d", cfg.uid))
	}
	if cfg.gid != 0 {
		args = append(args, fmt.Sprintf("--gid=%d", cfg.gid))
	}
	return args
}

// setPatternForVolume uses a "beegfs-ctl --unmounted --setpattern" command to set the pattern for a directory specified by
// vol.volDirPathBeegfsRoot on the BeeGFS file system. setPatternForVolume returns an error if it cannot set the pattern for
// the directory, but does not return an error if the pattern on the directory already exists. setPatternForVolume has no
// effect and does not return an error if config is empty.
func (ctlExec *beegfsCtlExecutor) setPatternForVolume(ctx context.Context, vol beegfsVolume, cfg stripePatternConfig) error {
	args, needToExecute := constructSetPatternForVolumeArgs(cfg)
	if needToExecute {
		args = append(args, vol.volDirPathBeegfsRoot)
		_, err := ctlExec.execute(ctx, vol.clientConfPath, args)
		if err != nil {
			return errors.WithMessagef(err, "cannot set pattern for BeeGFS directory %s for volume %s", vol.volDirPathBeegfsRoot, vol.sysMgmtdHost)
		}
	}

	return nil
}

// execute runs arbitrary beegfs-ctl commands like "beegfs-ctl --arg1 --arg2=value". It logs the stdout and stderr
// when running at a high verbosity and returns stdout as a string (as well as any potential errors). execute fails if
// beegfs-ctl is not on the PATH.
func (*beegfsCtlExecutor) execute(ctx context.Context, clientConfPath string, args []string) (stdOut string, err error) {
	args = append([]string{fmt.Sprintf("--cfgFile=%s", clientConfPath)}, args...)
	cmd := exec.Command("beegfs-ctl", args...)
	LogDebug(ctx, "Executing command", "command", cmd.Args)

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
			err = errors.Wrapf(err, "beegfs-ctl failed with stdOut: %s and stdErr: %s", stdOutString, stdErrString)
		}
	}
	if stdOutString != "" {
		LogVerbose(ctx, "stdout from command", "command", cmd.Args, "stdout", stdOutString)
	}
	if stdErrString != "" {
		LogVerbose(ctx, "stderr from command", "command", cmd.Args, "stderr", stdErrString)
	}

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

func (*fakeBeegfsCtlExecutor) createDirectoryForVolume(ctx context.Context, vol beegfsVolume, cfg permissionsConfig) error {
	return nil
}

func (*fakeBeegfsCtlExecutor) statDirectoryForVolume(ctx context.Context, vol beegfsVolume) (string, error) {
	return "", nil
}

func (*fakeBeegfsCtlExecutor) setPatternForVolume(ctx context.Context, vol beegfsVolume, cfg stripePatternConfig) error {
	return nil
}
