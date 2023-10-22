package edurouter

import "fmt"

const (
	majorVersion = 0
	minorVersion = 2
	patchVersion = 0
)

func Version() string {
	return fmt.Sprintf("v%d.%d.%d", majorVersion, minorVersion, patchVersion)
}
