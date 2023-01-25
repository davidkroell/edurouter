package edurouter

type LinkLayerHandler interface {
	SupplierC() chan<- FrameFromInterface
}
