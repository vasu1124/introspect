package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	apimachineryversion "k8s.io/apimachinery/pkg/version"
)

var (
	gitVersion   = "1.3.2" // x-release-please-version
	gitCommit    = "dev"
	gitTreeState = ""
	buildDate    = "1970-01-01T00:00:00Z"
	gitMajor     string
	gitMinor     string
	// Version is the exported git version string.
	Version = gitVersion
	// Flag is true when patch version is even (for UI toggling).
	Flag bool
)

// GetPatchVersion returns the patch version.
func GetPatchVersion() int {
	var version = strings.Split(gitVersion, ".")
	if len(version) >= 2 {
		patch, _ := strconv.Atoi(version[2])
		return patch
	}

	return 0
}

func init() {
	info, ok := debug.ReadBuildInfo()
	if ok {
		modified := false
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				gitCommit = setting.Value
			case "vcs.time":
				buildDate = setting.Value
			case "vcs.modified":
				modified, _ = strconv.ParseBool(setting.Value)
			}
		}
		if modified {
			gitCommit += "+CHANGES"
			gitVersion += "-dirty"
			gitTreeState = "dirty"
		}
	}
	version := strings.Split(gitVersion, ".")
	if len(version) >= 2 {
		gitMajor = version[0]
		gitMinor = version[1]
	}
	Version = gitVersion
	// Update exported variables
	Flag = GetPatchVersion()%2 == 0
}

// Get returns the overall codebase version. It's for detecting
// what code a binary was built from.
// These variables typically come from -ldflags or ReadBuildInfo() settings
func Get() apimachineryversion.Info {
	return apimachineryversion.Info{
		Major:        gitMajor,
		Minor:        gitMinor,
		GitVersion:   gitVersion,
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		BuildDate:    buildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
