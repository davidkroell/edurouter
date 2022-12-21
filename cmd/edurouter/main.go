package main

import (
	"context"
	"github.com/davidkroell/edurouter"
	"log"
	"net"
)

func main() {
	wifiInterface, err := edurouter.NewInterfaceConfig("wlp0s20f3", &net.IPNet{
		IP:   []byte{192, 168, 0, 80},
		Mask: net.CIDRMask(24, 32),
	})

	if err != nil {
		log.Println(err)
		return
	}

	dockerInterface, err := edurouter.NewInterfaceConfig("docker0", &net.IPNet{
		IP:   []byte{172, 17, 0, 50},
		Mask: net.CIDRMask(16, 32),
	})

	ctx := context.Background()

	listener := edurouter.NewLinkLayerListener(wifiInterface, dockerInterface)

	listener.ListenAndServe(ctx)
}
