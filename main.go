package main

import (
	"context"
	"github.com/davidkroell/edurouter/ifconfigv4"
	"log"
	"net"
)

func main() {
	simulatedIp := &net.IPNet{
		IP:   []byte{192, 168, 0, 80},
		Mask: net.CIDRMask(24, 32),
	}

	interfaceConfig, err := ifconfigv4.NewInterfaceConfig("wlp0s20f3", simulatedIp)
	if err != nil {
		log.Println(err)
		return
	}

	ctx := context.Background()

	listener := ifconfigv4.NewListener(interfaceConfig)

	listener.ListenAndServe(ctx)
}
