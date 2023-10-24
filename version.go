package edurouter

import (
	"fmt"
	"runtime/debug"
)

const (
	majorVersion = 0
	minorVersion = 2
	patchVersion = 0
)

func Version() string {
	return fmt.Sprintf("v%d.%d.%d %s", majorVersion, minorVersion, patchVersion, vcsInfo())
}

func vcsInfo() string {
	var revision string
	var time string

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				revision = setting.Value
			}
			if setting.Key == "vcs.time" {
				time = setting.Value
			}
		}
	}
	return "git:" + revision + ", at " + time
}
