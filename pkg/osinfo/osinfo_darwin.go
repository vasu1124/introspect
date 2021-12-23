//go:build darwin
// +build darwin

package osinfo

import (
	"os/exec"
	"strings"
	"syscall"

	"github.com/vasu1124/introspect/pkg/logger"
)

/*
 * Emulated routines (syscalls on some other operating systems)
 */

// Uname an call to fill
func uname(buf *utsname) (errno error) {
	buf.Sysname, errno = syscall.Sysctl("kern.ostype")
	if errno != nil {
		return
	}

	buf.Nodename, errno = syscall.Sysctl("kern.hostname")
	if errno != nil {
		return
	}

	buf.Release, errno = syscall.Sysctl("kern.osrelease")
	if errno != nil {
		return
	}

	buf.Version, errno = syscall.Sysctl("kern.version")
	if errno != nil {
		return
	}

	buf.Machine, errno = syscall.Sysctl("hw.machine")
	return
}

func parseOS() {
	out, err := exec.Command("/usr/bin/sw_vers").Output()
	if err != nil {
		logger.Log.Error(err, "[osinfo darwin] exec")
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		splitString := strings.SplitN(line, ":", 2)
		if len(splitString) != 2 {
			continue
		}

		// trim white space from key and value
		key := splitString[0]
		value := splitString[1]
		value = strings.Trim(value, "\t ")
		OSrelease[key] = value
	}
}
