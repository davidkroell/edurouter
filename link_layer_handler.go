package edurouter

import (
	"github.com/mdlayher/ethernet"
)

type LinkLayerHandler interface {
	Handle(*ethernet.Frame, *InterfaceConfig) (*ethernet.Frame, error)
}
