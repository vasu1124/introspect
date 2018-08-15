// +build linux

package osinfo

import (
	"bufio"
	"errors"
	"log"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

/*
 * Emulated routines (syscalls on some other operating systems)
 */

// Uname an call to fill
func uname(buf *utsname) (errno error) {
	var uts unix.Utsname

	errno = unix.Uname(&uts)
	if errno != nil {
		return
	}

	buf.Sysname = charsToString(uts.Sysname)
	buf.Nodename = charsToString(uts.Nodename)
	buf.Release = charsToString(uts.Release)
	buf.Version = charsToString(uts.Version)
	buf.Machine = charsToString(uts.Machine)

	return
}

func charsToString(ca [65]byte) string {
	s := make([]byte, len(ca))
	var lens int
	for ; lens < len(ca); lens++ {
		if ca[lens] == 0 {
			break
		}
		s[lens] = uint8(ca[lens])
	}
	return string(s[0:lens])
}

var etcRelease = [...]string{
	"/etc/lsb-release",
	"/etc/os-release",
	"/etc/SUSE-release",
	"/etc/redhat-release",
	"/etc/fedora-release",
	"/etc/slackware-release",
	"/etc/debian_release",
	"/etc/mandrake-release",
	"/etc/yellowdog-release",
	"/etc/sun-release",
	"/etc/release",
	"/etc/gentoo-release",
	"/etc/UnitedLinux-release",
	"",
}

func parseOS() (ok bool) {
	var filename string
	for _, filename = range etcRelease {
		_, err := os.Stat(filename)
		if err == nil {
			break
		}
	}
	if filename == "" {
		log.Printf("No os-release file available!")
		return false
	}
	lines, err := parseFile(filename)
	if err != nil {
		return false
	}

	for _, v := range lines {
		key, value, err := parseLine(v)
		if err == nil {
			OSrelease[key] = value
		}
	}

	return true
}

func parseFile(filename string) (lines []string, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func parseLine(line string) (key string, value string, err error) {
	err = nil

	// skip empty lines
	if len(line) == 0 {
		err = errors.New("Skipping: zero-length")
		return
	}

	// skip comments
	if line[0] == '#' {
		err = errors.New("Skipping: comment")
		return
	}

	// try to split string at the first '='
	splitString := strings.SplitN(line, "=", 2)
	if len(splitString) != 2 {
		err = errors.New("Can not extract key=value")
		return
	}

	// trim white space from key and value
	key = splitString[0]
	key = strings.Trim(key, " ")
	value = splitString[1]
	value = strings.Trim(value, " ")

	// Handle double quotes
	if strings.ContainsAny(value, `"`) {
		first := string(value[0:1])
		last := string(value[len(value)-1:])

		if first == last && strings.ContainsAny(first, `"'`) {
			value = strings.TrimPrefix(value, `'`)
			value = strings.TrimPrefix(value, `"`)
			value = strings.TrimSuffix(value, `'`)
			value = strings.TrimSuffix(value, `"`)
		}
	}

	// expand anything else that could be escaped
	value = strings.Replace(value, `\"`, `"`, -1)
	value = strings.Replace(value, `\$`, `$`, -1)
	value = strings.Replace(value, `\\`, `\`, -1)
	value = strings.Replace(value, "\\`", "`", -1)
	return
}
