package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/rackn/gohai/plugins/dmi"
	"github.com/rackn/gohai/plugins/net"
)

type info interface {
	Class() string
}

func main() {
	infos := map[string]info{}
	dmiInfo, err := dmi.Gather()
	if err != nil {
		log.Fatalf("Failed to gather DMI information: %v", err)
	}
	infos[dmiInfo.Class()] = dmiInfo
	netInfo, err := net.Gather()
	if err != nil {
		log.Fatalf("Failed to gather network info: %v", err)
	}
	infos[netInfo.Class()] = netInfo
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(infos)
}
