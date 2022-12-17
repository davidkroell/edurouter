package main

import (
	"context"
	"github.com/davidkroell/edurouter/ifconfigv4"
	"log"
	"net"
)

func main() {
	wifiInterface, err := ifconfigv4.NewInterfaceConfig("wlp0s20f3", &net.IPNet{
		IP:   []byte{192, 168, 0, 80},
		Mask: net.CIDRMask(24, 32),
	})

	if err != nil {
		log.Println(err)
		return
	}

	dockerInterface, err := ifconfigv4.NewInterfaceConfig("docker0", &net.IPNet{
		IP:   []byte{172, 17, 0, 50},
		Mask: net.CIDRMask(16, 32),
	})

	ctx := context.Background()

	listener := ifconfigv4.NewListener(wifiInterface, dockerInterface)

	listener.ListenAndServe(ctx)
}
