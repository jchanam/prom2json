package main

import (
	"fmt"

	"github.com/javierprovecho/prom2json"
)

func main() {
	json, err := prom2json.Parse("http://146.185.151.73:9100/metrics")

	if err != nil {
		panic(err)
	}

	fmt.Printf("%s", json["node_network_transmit_bytes"])
}
