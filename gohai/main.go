package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/rackn/gohai/plugins/dmi"
)

func main() {
	dmiInfo, err := dmi.Gather()
	if err != nil {
		log.Fatalf("Failed to gather DMI information: %v", err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(dmiInfo)
}
