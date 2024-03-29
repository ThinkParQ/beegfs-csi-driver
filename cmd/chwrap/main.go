// https://github.com/NetApp/trident/blob/892e75d8ce216c9d024ce47ca5876ad89c08d312/chwrap/chwrap.go

/*
 *  Copyright (c) 2020 NetApp
 *  All rights reserved
 */

/*
Modifications Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package main

import (
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

func validBinary(path string) bool {
	var stat unix.Stat_t
	// This call was unix.Stat() in the original Trident file. However, unix.Stat() follows symbolic links. This
	// behavior made it impossible to stat /home/usr/sbin/beegfs-ctl (and any other binary reachable via absolute
	// symbolic link). The intent was likely to use unix.Lstat(), as the followup block checks whether the returned
	// Stat_t refers to a regular file OR a link. Regardless, unix.Lstat() is required for our use case.
	if err := unix.Lstat(path, &stat); nil != err {
		// Can't stat file
		return false
	}
	if (stat.Mode&unix.S_IFMT) != unix.S_IFREG && (stat.Mode&unix.S_IFMT) != unix.S_IFLNK {
		// Not a regular file or symlink
		return false
	}
	if 0 == stat.Mode&unix.S_IRUSR || 0 == stat.Mode&unix.S_IXUSR {
		// Not readable or not executable
		return false
	}
	return true
}

func findBinary(prefix, binary string) string {
	// Some automatic worker node prep workflows put BeeGFS utilities in the plugin-owned
	// /var/lib/kubelet/plugins/beegfs.csi.netapp.com/client/sbin directory to avoid base OS "contamination".
	for _, part1 := range []string{"var/lib/kubelet/plugins/beegfs.csi.netapp.com/client/", "usr/local/", "usr/", ""} {
		for _, part2 := range []string{"sbin", "bin"} {
			path := "/" + part1 + part2 + "/" + binary
			if validBinary(prefix + path) {
				return path
			}
		}
	}
	return ""
}

func modifyEnv(oldEnv []string) []string {
	var newEnv []string
	for _, e := range oldEnv {
		if !strings.HasPrefix(e, "PATH=") {
			newEnv = append(newEnv, e)
		}
	}
	// Some automatic worker node prep workflows put BeeGFS utilities in the plugin-owned
	// /var/lib/kubelet/plugins/beegfs.csi.netapp.com/client/sbin directory to avoid base OS "contamination".
	newEnv = append(newEnv, "PATH=/var/lib/kubelet/plugins/beegfs.csi.netapp.com/client/sbin:"+
		"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
	return newEnv
}

func main() {
	// First modify argv0 to strip off any absolute or relative paths
	argv := os.Args
	binary := argv[0]
	idx := strings.LastIndexByte(binary, '/')
	if 0 <= idx {
		binary = binary[idx+1:]
	}
	// Now implement the path search logic, but in the host's filesystem
	argv0 := findBinary("/host", binary)
	if "" == argv0 {
		panic(binary + " not found")
	}
	// Chroot in the the host's FS
	if err := unix.Chroot("/host"); nil != err {
		panic(err)
	}
	// Change cwd to the root
	if err := unix.Chdir("/"); nil != err {
		panic(err)
	}
	// Exec the intended binary
	if err := unix.Exec(argv0, argv, modifyEnv(os.Environ())); nil != err {
		panic(err)
	}
}
