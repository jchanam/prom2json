package main

import (
	"fmt"

	"github.com/javierprovecho/prom2json"
)

func main() {
	json := prom2json.Parse("http://146.185.151.73:9100/metrics")

	if json == nil {
		panic("json == nil")
	}

	fmt.Printf("%s", json["node_network_transmit_bytes"])
}
