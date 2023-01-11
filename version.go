package edurouter

import "fmt"

const (
	majorVersion = 0
	minorVersion = 1
	patchVersion = 0
)

func Version() string {
	return fmt.Sprintf("v%d.%d.%d", majorVersion, minorVersion, patchVersion)
}
