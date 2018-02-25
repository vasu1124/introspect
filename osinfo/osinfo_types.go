package osinfo

// Utsname somewhat more relaxed than <sys/utsname.h>, valuing the real data
// from sysctl(2) over the arbitrary lengths defined in struct utsname.
type utsname struct {
	Sysname  string
	Nodename string
	Release  string
	Version  string
	Machine  string
}

// Utsname .
var Utsname utsname

// OSrelease .
var OSrelease = make(map[string]string)

func init() {
	uname(&Utsname)
	parseOS()
	OSrelease["Release"] = Utsname.Release
	OSrelease["Sysname"] = Utsname.Sysname
	OSrelease["Version"] = Utsname.Version

}
