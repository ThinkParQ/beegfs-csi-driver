/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package beegfs

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"path"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// beegfsCtlExecutorInterface abstracts beegfs-ctl so tests can run without access to a beegfs-ctl binary or a BeeGFS
// file system.
type beegfsCtlExecutorInterface interface {
	createDirectoryForVolume(ctx context.Context, vol beegfsVolume, dirPath string, cfg permissionsConfig) error
	statDirectoryForVolume(ctx context.Context, vol beegfsVolume, dirPath string) (string, error)
	setPatternForVolume(ctx context.Context, vol beegfsVolume, cfg stripePatternConfig) error
}

// beegfsCtlDispatcher handles calling either the v7 or v8 CTL depending on the beegfsVolume.
type beegfsCtlDispatcher struct {
}

// newBeeGFSCtlExecutor returns a beegfsCtlDispatcher that satisfies the beegfsCtlExecutorInterface
// interface and can execute either the v7 or v8 CTL. This checks upfront that the v7 and/or v8 CTL
// are installed. If neither are installed the driver will refuse to start so the admin knows
// immediately about the missing prerequisite (instead of later when trying to create/mount PVCs).
//
// For every subsequent command it always rechecks what versions of CTL are installed and if the
// sysMgmtdHost for the specified beegfsVolume is v7 or v8. It doesn't cache what versions of CTL
// are installed or what sysMgmtdHost (filesystems) are v7 or v8 so these can be later upgraded
// without requiring a driver restart.
func newBeeGFSCtlExecutor() (beegfsCtlExecutorInterface, error) {

	LogDebug(context.TODO(), "Detecting installed CTL versions")
	v8CTL := beegfsCtlExecutorV8{}
	v7CTL := beegfsCtlExecutorV7{}

	// We cannot simply use exec.LookPath to determine this because chwrap confuses it. Instead, we
	// execute beegfs (for v8) and beegfs-ctl (for v7) with the --help option to check which
	// versions of CTL are available.
	_, v8Err := v8CTL.execute(context.TODO(), beegfsVolume{}, []string{"--help"})
	if v8Err == nil {
		LogDebug(context.TODO(), "Found BeeGFS 8 CTL install")
	}
	_, v7err := v7CTL.execute(context.TODO(), "", []string{"--help"})
	if v7err != nil && errors.As(v7err, &ctlConnAuthError{}) {
		// A connAuth error here is not significant. We will likely be picking up conn auth config
		// on a per file system basis. For now, just verify if we can execute beegfs-ctl or not.
		v7err = nil
		LogDebug(context.TODO(), "Found BeeGFS 7 CTL install")
	}

	if v7err != nil && v8Err != nil {
		return nil, fmt.Errorf("unable to verify the BeeGFS 7 CTL is installed: %w; unable to verify the BeeGFS 8 CTL is installed: %w (one must be installed and in $PATH to proceed)", v7err, v8Err)
	}

	return beegfsCtlDispatcher{}, nil
}

// detectCTLVersion determines if this is a v7 or v8 beegfsVolume. It works by first trying to run
// the v8 `beegfs node list` command for this sysMgmtdHost. If the v8 CTL is not installed or node
// list fails, it tries to run the v7 `beegfs-ctl --listnodes` to determine if this is a v7 volume.
func (d beegfsCtlDispatcher) detectCTLVersion(ctx context.Context, vol beegfsVolume) (beegfsCtlExecutorInterface, error) {

	LogDebug(context.TODO(), "Detecting BeeGFS version for volume", "volumeID", vol.volumeID)

	var errCheckingV8NodeList error
	var errCheckingV7NodeList error
	var errString strings.Builder

	// As with above, we don't use exec.LookPath because chwrap confuses it.
	executorV8 := beegfsCtlExecutorV8{}
	if _, errCheckingV8NodeList = executorV8.execute(ctx, vol, []string{"node", "list"}); errCheckingV8NodeList == nil {
		// If we were able to execute node list for this volume, this must be BeeGFS 8.
		LogDebug(context.TODO(), "BeeGFS 8 volume detected", "volumeID", vol.volumeID)
		return executorV8, nil
	}
	// The command could fail because the beegfs tool was not installed, or due to a runtime error.
	// We don't also check if the tool is installed to reduce overhead (the handling is the same).
	errString.WriteString("BeeGFS 8 list nodes failed (" + errCheckingV8NodeList.Error() + ")")
	LogDebug(ctx, "executing the v8 ctl with this volume failed, falling back to v7", "vol", vol.volumeID, "error", errCheckingV8NodeList)

	executorV7 := beegfsCtlExecutorV7{}
	if _, errCheckingV7NodeList = executorV7.execute(ctx, vol.clientConfPath, []string{"--listnodes", "--nodetype=management"}); errCheckingV7NodeList == nil {
		// If we were able to execute node list for this volume, this must be BeeGFS 7.
		LogDebug(context.TODO(), "BeeGFS 7 volume detected", "volumeID", vol.volumeID)
		return &executorV7, nil
	}
	if errString.Len() > 0 {
		errString.WriteString("; ")
	}
	errString.WriteString("BeeGFS 7 list nodes failed (" + errCheckingV7NodeList.Error() + ")")
	// Neither the v8 or v7 CTL are installed or this is not a compatible beegfsVolume.
	return nil, fmt.Errorf("unable to verify if this is a BeeGFS 7 or 8 volume: %s (hint: verify the management address and that the correct version of BeeGFS CTL is installed and in $PATH)", errString.String())
}

func (d beegfsCtlDispatcher) createDirectoryForVolume(ctx context.Context, vol beegfsVolume, dirPath string, cfg permissionsConfig) error {
	if ctl, err := d.detectCTLVersion(ctx, vol); err != nil {
		return err
	} else {
		return ctl.createDirectoryForVolume(ctx, vol, dirPath, cfg)
	}
}

func (d beegfsCtlDispatcher) statDirectoryForVolume(ctx context.Context, vol beegfsVolume, dirPath string) (string, error) {
	if ctl, err := d.detectCTLVersion(ctx, vol); err != nil {
		return "", err
	} else {
		return ctl.statDirectoryForVolume(ctx, vol, dirPath)
	}
}
func (d beegfsCtlDispatcher) setPatternForVolume(ctx context.Context, vol beegfsVolume, cfg stripePatternConfig) error {
	if ctl, err := d.detectCTLVersion(ctx, vol); err != nil {
		return err
	} else {
		return ctl.setPatternForVolume(ctx, vol, cfg)
	}
}

// execBeeGFSCmd provides a common approach to handling output from v7/v8 BeeGFS CTL commands.
func execBeeGFSCmd(ctx context.Context, cmd *exec.Cmd, isHelpCommand bool) (stdOut string, err error) {
	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer
	LogDebug(ctx, "Executing command", "command", cmd.Args)
	// Keep in mind we're running chwrapped commands here, which may return different error codes or
	// stdout/stderr than the non-chwrapped versions. See chwrap/main.go for details.
	err = cmd.Run()
	stdOutString := stdoutBuffer.String()
	stdErrString := stderrBuffer.String()
	if err != nil {
		if strings.Contains(stdErrString, "does not exist") || strings.Contains(stdErrString, "no such file or directory") {
			// In BeeGFS 8 if CTL is run with mount=none then the error is 'does not exist' (the same as
			// BeeGFS 7). However if CTL was ever executed against a mounted file system the error would
			// be "no such file or directory".
			err = newCtlNotExistError(stdOutString, stdErrString)
		} else if strings.Contains(stdErrString, "exists already") || strings.Contains(stdOutString, "exists already") {
			// In BeeGFS 8 since commands can act on multiple entries, "exists already" is only
			// returned in stdout.
			err = newCtlExistError(stdOutString, stdErrString)
		} else if strings.Contains(stdErrString, "No connAuthFile configured") {
			err = newCtlConnAuthError(stdOutString, stdErrString)
		} else {
			// This is also the codepath if CTL is not installed (i.e., beegfs: command not found).
			err = fmt.Errorf("error executing ctl: %w (stdOut: %q | stdErr: %q)", err, strings.TrimRight(stdOutString, "\n"), strings.TrimRight(stdErrString, "\n"))
		}
	}
	if stdOutString != "" && !isHelpCommand { // Don't log the --help output.
		LogVerbose(ctx, "stdout from command", "command", cmd.Args, "stdout", stdOutString)
	}
	if stdErrString != "" {
		LogVerbose(ctx, "stderr from command", "command", cmd.Args, "stderr", stdErrString)
	}

	return stdOutString, err
}

type beegfsCtlExecutorV8 struct{}

func (ctl beegfsCtlExecutorV8) createDirectoryForVolume(ctx context.Context, vol beegfsVolume, dirPath string, cfg permissionsConfig) error {
	LogDebug(ctx, "Creating BeeGFS directory", "path", dirPath, "volumeID", vol.volumeID)
	// Check if directory already exists.
	_, err := ctl.statDirectoryForVolume(ctx, vol, dirPath)
	if errors.As(err, &ctlNotExistError{}) {
		// We can't find the directory so we need to create it.
		LogDebug(ctx, "BeeGFS directory does not exist", "path", dirPath, "volumeID", vol.volumeID)

		// Construct the set of arguments that will be used to create any necessary directories.
		createDirArgs := constructCreateDirForVolumeArgs(cfg, true)

		// Multiple parent directories may need to be created.
		// Create a slice of paths where the first path is the most general and each subsequent path is less general.
		dirsToMake := []string{dirPath}
		for dir := path.Dir(dirPath); dir != "/"; { // path.Dir() returns "." if there is no parent.
			dirsToMake = append([]string{dir}, dirsToMake...) // Prepend so the more general path comes first.
			dir = path.Dir(dir)
		}
		// Starting with the most general path, create all directories required to eventually create dirPath.
		for _, dir := range dirsToMake {
			_, err = ctl.execute(ctx, vol, append(createDirArgs, dir))
			if err != nil && !errors.As(err, &ctlExistError{}) {
				// We can't create the volume.
				return errors.WithMessagef(err, "cannot create BeeGFS directory %s for %s", dir, vol.volumeID)
			}
		}
	} else if err != nil {
		return err
	} else {
		LogDebug(ctx, "BeeGFS directory already exists", "path", dirPath, "volumeID", vol.volumeID)
	}

	return nil
}
func (ctl beegfsCtlExecutorV8) statDirectoryForVolume(ctx context.Context, vol beegfsVolume, dirPath string) (string, error) {
	return ctl.execute(ctx, vol, []string{"--mount=none", "entry", "info", dirPath})
}
func (ctl beegfsCtlExecutorV8) setPatternForVolume(ctx context.Context, vol beegfsVolume, cfg stripePatternConfig) error {
	args, needToExecute := constructSetPatternForVolumeArgs(cfg, true)
	if needToExecute {
		args = append(args, vol.volDirPathBeegfsRoot)
		_, err := ctl.execute(ctx, vol, args)
		if err != nil {
			return errors.WithMessagef(err, "cannot set pattern for BeeGFS directory %s for volume %s", vol.volDirPathBeegfsRoot, vol.sysMgmtdHost)
		}
	}

	return nil
}

func (*beegfsCtlExecutorV8) execute(ctx context.Context, vol beegfsVolume, args []string) (stdOut string, err error) {
	if len(args) > 0 && args[0] == "--help" {
		// We want to log differently if this is just a --help command. There is also no reason to
		// parse out the mgmtd/auth/tls config in this case.
		return execBeeGFSCmd(ctx, exec.Command("beegfs", args...), true)
	}

	// The v8 CTL does not use the BeeGFS client config file. Provide all required configuration
	// using flags from the volume config. Set the default management gRPC port if unspecified.
	port := vol.config.GrpcPort
	if port == "" {
		port = "8010"
	}
	args = append(args, fmt.Sprintf("--mgmtd-addr=%s", net.JoinHostPort(vol.sysMgmtdHost, port)))
	if len(vol.config.ConnAuth) != 0 {
		args = append(args, fmt.Sprintf("--auth-file=%s", vol.getConnAuthPath()))
	} else {
		args = append(args, "--auth-disable")
	}

	if len(vol.config.TLSCert) != 0 {
		args = append(args, fmt.Sprintf("--tls-cert-file=%s", vol.getTLSCertPath()))
	} else {
		args = append(args, "--tls-disable")
	}
	return execBeeGFSCmd(ctx, exec.Command("beegfs", args...), false)
}

type beegfsCtlExecutorV7 struct{}

// createDirectoryForVolume uses a "beegfs-ctl --createdir" command to create the directory specified by dirPath on the
// BeeGFS file system specified by vol.sysMgmtdHost. createDirectoryForPath returns an error if it cannot create the
// directory, but does not return an error if the directory already exists.
func (ctlExec *beegfsCtlExecutorV7) createDirectoryForVolume(ctx context.Context, vol beegfsVolume, dirPath string, cfg permissionsConfig) error {
	LogDebug(ctx, "Creating BeeGFS directory", "path", dirPath, "volumeID", vol.volumeID)
	// Check if directory already exists.
	_, err := ctlExec.statDirectoryForVolume(ctx, vol, dirPath)
	if errors.As(err, &ctlNotExistError{}) {
		// We can't find the directory so we need to create it.
		LogDebug(ctx, "BeeGFS directory does not exist", "path", dirPath, "volumeID", vol.volumeID)

		// Construct the set of arguments that will be used to create any necessary directories.
		createDirArgs := constructCreateDirForVolumeArgs(cfg, false)

		// Multiple parent directories may need to be created.
		// Create a slice of paths where the first path is the most general and each subsequent path is less general.
		dirsToMake := []string{dirPath}
		for dir := path.Dir(dirPath); dir != "/"; { // path.Dir() returns "." if there is no parent.
			dirsToMake = append([]string{dir}, dirsToMake...) // Prepend so the more general path comes first.
			dir = path.Dir(dir)
		}
		// Starting with the most general path, create all directories required to eventually create dirPath.
		for _, dir := range dirsToMake {
			_, err = ctlExec.execute(ctx, vol.clientConfPath, append(createDirArgs, dir))
			if err != nil && !errors.As(err, &ctlExistError{}) {
				// We can't create the volume.
				return errors.WithMessagef(err, "cannot create BeeGFS directory %s for %s", dir, vol.volumeID)
			}
		}
	} else if err != nil {
		return err
	} else {
		LogDebug(ctx, "BeeGFS directory already exists", "path", dirPath, "volumeID", vol.volumeID)
	}
	return nil
}

// statDirectoryForVolume returns the information output by "beegfs-ctl --getentryinfo dirPath" as a string, or an empty
// string and an error if the stat fails.
func (ctlExec *beegfsCtlExecutorV7) statDirectoryForVolume(ctx context.Context, vol beegfsVolume, dirPath string) (string, error) {
	return ctlExec.execute(ctx, vol.clientConfPath, []string{"--unmounted", "--getentryinfo", dirPath})
}

// constructSetPatternForVolumeArgs constructs the slice of arguments that will be passed to ctlExec.execute() in a
// setPatternForVolume() call. We keep this logic in a separate function for easy testing.
func constructSetPatternForVolumeArgs(cfg stripePatternConfig, isV8 bool) ([]string, bool) {
	var needToExecute bool
	var args []string

	if isV8 {
		if cfg.stripePatternNumTargets != "" {
			args = append([]string{fmt.Sprintf("--num-targets=%s", cfg.stripePatternNumTargets)}, args...)
			needToExecute = true
		}
		if cfg.stripePatternChunkSize != "" {
			args = append([]string{fmt.Sprintf("--chunk-size=%s", cfg.stripePatternChunkSize)}, args...)
			needToExecute = true
		}
		if cfg.storagePoolID != "" {
			args = append([]string{fmt.Sprintf("--pool=%s", cfg.storagePoolID)}, args...)
			needToExecute = true
		}
		if needToExecute {
			args = append([]string{"--mount=none", "entry", "set"}, args...)
			return args, true
		}
		return []string{}, false
	}
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
func constructCreateDirForVolumeArgs(cfg permissionsConfig, isV8 bool) []string {
	var args []string
	// The v7 beegfs-ctl ignores special permissions (the first digit in the four digit octal
	// permissions schema). The v8 beegfs tool respect special permissions, but for clarity and
	// consistency the mode argument is always constructed without this digit. As a result only 3
	// digits padded with 0s to the left will be appended to the command.
	mode := cfg.mode & 0o777
	if isV8 {
		args = []string{"--mount=none", "entry", "create", "directory", fmt.Sprintf("--permissions=%03o", mode)}
	} else {
		args = []string{"--unmounted", "--createdir", fmt.Sprintf("--access=%03o", mode)}
	}

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
func (ctlExec *beegfsCtlExecutorV7) setPatternForVolume(ctx context.Context, vol beegfsVolume, cfg stripePatternConfig) error {
	args, needToExecute := constructSetPatternForVolumeArgs(cfg, false)
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
func (*beegfsCtlExecutorV7) execute(ctx context.Context, clientConfPath string, args []string) (stdOut string, err error) {
	isHelpCommand := false
	if len(args) > 0 && args[0] == "--help" {
		// We want to log differently if this is just a --help command. Still append cfgFile, even
		// though this might be an empty string (which still works) because that is how it was done
		// in the driver before adding support for v8.
		isHelpCommand = true
	}
	args = append([]string{fmt.Sprintf("--cfgFile=%s", clientConfPath)}, args...)
	return execBeeGFSCmd(ctx, exec.Command("beegfs-ctl", args...), isHelpCommand)
}

// ctlError defines a common structure for the more specific errors it is intended to be embedded into.
type ctlError struct {
	stdOutString string
	stdErrString string
}

func (err ctlError) Error() string {
	return fmt.Sprintf("beegfs-ctl failed with stdOut: %v and stdErr: %v", err.stdOutString, err.stdErrString)
}

// ctlNotExistError indicates that beegfs-ctl failed to stat or modify an entry that does not exist.
type ctlNotExistError struct {
	ctlError
}

func newCtlNotExistError(stdOutString, stdErrString string) ctlNotExistError {
	return ctlNotExistError{ctlError{stdOutString: stdOutString, stdErrString: stdErrString}}
}

// ctlExistError indicates that beegfs-ctl failed to create an entry that already exists.
type ctlExistError struct {
	ctlError
}

func newCtlExistError(stdOutString, stdErrString string) ctlExistError {
	return ctlExistError{ctlError{stdOutString: stdOutString, stdErrString: stdErrString}}
}

// ctlConnAuthError indicates that beegfs-ctl can't do anything useful because of a connAuth misconfiguration.
type ctlConnAuthError struct {
	ctlError
}

func newCtlConnAuthError(stdOutString, stdErrString string) ctlConnAuthError {
	return ctlConnAuthError{ctlError{stdOutString: stdOutString, stdErrString: stdErrString}}
}

// fakeBeeGFSCtlExecutor is a mock implementation of beegfsCtlExecutorInterface useful for testing.
type fakeBeegfsCtlExecutor struct{}

func (*fakeBeegfsCtlExecutor) createDirectoryForVolume(ctx context.Context, vol beegfsVolume, path string, cfg permissionsConfig) error {
	return nil
}

func (*fakeBeegfsCtlExecutor) statDirectoryForVolume(ctx context.Context, vol beegfsVolume, path string) (string, error) {
	return "", nil
}

func (*fakeBeegfsCtlExecutor) setPatternForVolume(ctx context.Context, vol beegfsVolume, cfg stripePatternConfig) error {
	return nil
}
